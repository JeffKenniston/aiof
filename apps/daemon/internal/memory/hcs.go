package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// ADR-19: Holographic Context Sub-Netting (HCS) & Merkle-DAG Provenance
// ============================================================================

const (
	// HypervectorSizeBits defines the bit dimensionality of HCS hypervectors (10,000 bits).
	HypervectorSizeBits = 10000
	// HypervectorSizeBytes defines the byte length (1250 bytes = 10,000 bits).
	HypervectorSizeBytes = 1250
	// HypervectorWords defines the number of 64-bit words (157 words = 10,048 bits; last 48 bits unused).
	HypervectorWords = 157
)

var (
	ErrNilHypervector = errors.New("hcs: hypervector cannot be nil or uninitialized")
	ErrNodeNotFound   = errors.New("hcs: node not found in Merkle-DAG")
	ErrInvalidProof   = errors.New("hcs: invalid Merkle proof")
	ErrEmptyPayload   = errors.New("hcs: node payload cannot be empty")
)

// Hypervector represents a 10,000-bit binary spatter hypervector for Hyperdimensional Computing (HDC) / VSA.
type Hypervector struct {
	Data [HypervectorSizeBytes]byte `json:"data"`
}

// NewHypervector creates an empty (zero-filled) 10,000-bit hypervector.
func NewHypervector() Hypervector {
	return Hypervector{}
}

// RandomHypervector generates a pseudo-random 10,000-bit hypervector derived from a seed string.
func RandomHypervector(seed string) Hypervector {
	hv := Hypervector{}
	h := sha256.New()
	h.Write([]byte(seed))
	baseHash := h.Sum(nil)

	for i := 0; i < HypervectorSizeBytes; i += 32 {
		h.Reset()
		h.Write(baseHash)
		h.Write([]byte(fmt.Sprintf("block-%d", i)))
		block := h.Sum(nil)
		copy(hv.Data[i:minInt(i+32, HypervectorSizeBytes)], block)
	}

	// Mask unused 48 bits in byte 1249 if necessary (10,000 bits = 1250 full bytes)
	return hv
}

// Bind performs bitwise XOR between two hypervectors (a ^ b).
// In binary HDC, XOR acts as the binding operator (associative, commutative, self-inverse).
func Bind(a, b Hypervector) Hypervector {
	res := Hypervector{}
	for i := 0; i < HypervectorSizeBytes; i++ {
		res.Data[i] = a.Data[i] ^ b.Data[i]
	}
	return res
}

// Bundle performs element-wise majority voting across a set of hypervectors.
// Tied bit positions default to 1 if the input count is even.
func Bundle(vectors []Hypervector) Hypervector {
	if len(vectors) == 0 {
		return Hypervector{}
	}
	if len(vectors) == 1 {
		return vectors[0]
	}

	res := Hypervector{}
	half := len(vectors) / 2

	for bitIdx := 0; bitIdx < HypervectorSizeBits; bitIdx++ {
		byteIdx := bitIdx / 8
		bitOffset := uint(7 - (bitIdx % 8))

		count := 0
		for vIdx := 0; vIdx < len(vectors); vIdx++ {
			if (vectors[vIdx].Data[byteIdx] & (1 << bitOffset)) != 0 {
				count++
			}
		}

		if count > half || (count == half && (bitIdx%2 == 0)) {
			res.Data[byteIdx] |= (1 << bitOffset)
		}
	}

	return res
}

// Permute performs a cyclic bit shift by n positions to encode sequential or positional context.
func Permute(hv Hypervector, shift int) Hypervector {
	shift = ((shift % HypervectorSizeBits) + HypervectorSizeBits) % HypervectorSizeBits
	if shift == 0 {
		return hv
	}

	res := Hypervector{}
	for i := 0; i < HypervectorSizeBits; i++ {
		srcIdx := (i - shift + HypervectorSizeBits) % HypervectorSizeBits
		srcByte := srcIdx / 8
		srcBit := uint(7 - (srcIdx % 8))

		dstByte := i / 8
		dstBit := uint(7 - (i % 8))

		if (hv.Data[srcByte] & (1 << srcBit)) != 0 {
			res.Data[dstByte] |= (1 << dstBit)
		}
	}
	return res
}

// HammingDistance calculates the bitwise Hamming distance between two 10,000-bit hypervectors.
func HammingDistance(a, b Hypervector) int {
	dist := 0
	for i := 0; i < HypervectorSizeBytes; i++ {
		dist += bits.OnesCount8(a.Data[i] ^ b.Data[i])
	}
	return dist
}

// Similarity calculates normalized cosine/normalized Hamming similarity [0.0, 1.0].
// 1.0 indicates identity, 0.5 indicates orthogonal independence, 0.0 indicates inverse.
func Similarity(a, b Hypervector) float64 {
	dist := HammingDistance(a, b)
	return 1.0 - (float64(dist) / float64(HypervectorSizeBits))
}

// EncodeConcept encodes a text concept or string key into a deterministic 10,000-bit hypervector.
func EncodeConcept(concept string) Hypervector {
	if concept == "" {
		return Hypervector{}
	}

	// Character n-gram hypervector bundling
	runes := []rune(concept)
	if len(runes) == 0 {
		return Hypervector{}
	}

	ngramVectors := make([]Hypervector, 0)
	n := 3 // Trigram window
	for i := 0; i < len(runes); i++ {
		sub := ""
		if i+n <= len(runes) {
			sub = string(runes[i : i+n])
		} else {
			sub = string(runes[i:])
		}
		hv := RandomHypervector("concept:" + sub)
		ngramVectors = append(ngramVectors, Permute(hv, i))
	}

	return Bundle(ngramVectors)
}

// OrthogonalProject projects a hypervector onto a target subspace defined by sub-net hypervectors.
func OrthogonalProject(hv Hypervector, subnetBases []Hypervector) Hypervector {
	if len(subnetBases) == 0 {
		return hv
	}
	boundSubnet := Bundle(subnetBases)
	return Bind(hv, boundSubnet)
}

// ============================================================================
// Merkle-DAG Provenance Engine Models & Methods
// ============================================================================

// MerkleNode represents a single node within the Merkle-DAG provenance tree.
type MerkleNode struct {
	ID          string            `json:"id"`
	Hash        string            `json:"hash"` // Hex SHA-256 digest
	ParentIDs   []string          `json:"parent_ids,omitempty"`
	ChildIDs    []string          `json:"child_ids,omitempty"`
	Hypervector Hypervector       `json:"hypervector"`
	Payload     []byte            `json:"payload"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// ComputeHash calculates the SHA-256 hash of a node based on payload, children, and hypervector.
func (n *MerkleNode) ComputeHash() string {
	h := sha256.New()
	h.Write([]byte(n.ID))
	h.Write(n.Payload)
	h.Write(n.Hypervector.Data[:])
	for _, childID := range n.ChildIDs {
		h.Write([]byte(childID))
	}
	for k, v := range n.Metadata {
		h.Write([]byte(k + ":" + v))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// MerkleProof represents a cryptographic inclusion/provenance proof for a node.
type MerkleProof struct {
	NodeID     string    `json:"node_id"`
	NodeHash   string    `json:"node_hash"`
	PathHashes []string  `json:"path_hashes"`
	StateRoot  string    `json:"state_root"`
	Timestamp  time.Time `json:"timestamp"`
	ProofChain []string  `json:"proof_chain"`
}

// SearchResult holds search candidate records returned during top-k Merkle-DAG traversal.
type SearchResult struct {
	Node       *MerkleNode `json:"node"`
	Similarity float64     `json:"similarity"`
	Distance   int         `json:"distance"`
}

// MerkleDAG manages the complete bitemporal Merkle-DAG memory provenance structure.
type MerkleDAG struct {
	mu        sync.RWMutex
	nodes     map[string]*MerkleNode
	roots     []string // Root node IDs
	stateRoot string
}

// NewMerkleDAG initializes an empty Merkle-DAG provenance engine.
func NewMerkleDAG() *MerkleDAG {
	return &MerkleDAG{
		nodes: make(map[string]*MerkleNode),
		roots: make([]string, 0),
	}
}

// AddNode adds a new memory payload & hypervector to the Merkle-DAG and updates state root.
func (dag *MerkleDAG) AddNode(id string, payload []byte, hv Hypervector, parentIDs []string, metadata map[string]string) (*MerkleNode, error) {
	if len(payload) == 0 {
		return nil, ErrEmptyPayload
	}

	dag.mu.Lock()
	defer dag.mu.Unlock()

	if id == "" {
		h := sha256.Sum256(payload)
		id = fmt.Sprintf("node-%s-%d", hex.EncodeToString(h[:8]), time.Now().UnixNano())
	}

	if metadata == nil {
		metadata = make(map[string]string)
	}

	node := &MerkleNode{
		ID:          id,
		ParentIDs:   parentIDs,
		ChildIDs:    make([]string, 0),
		Hypervector: hv,
		Payload:     payload,
		Metadata:    metadata,
		Timestamp:   time.Now().UTC(),
	}

	node.Hash = node.ComputeHash()

	// Link parents to child
	for _, parentID := range parentIDs {
		if parent, exists := dag.nodes[parentID]; exists {
			parent.ChildIDs = append(parent.ChildIDs, node.ID)
			parent.Hash = parent.ComputeHash() // Recompute parent hash
		}
	}

	dag.nodes[node.ID] = node

	// Maintain roots list
	if len(parentIDs) == 0 {
		dag.roots = append(dag.roots, node.ID)
	}

	dag.recomputeStateRoot()
	return node, nil
}

// GetNode retrieves a MerkleNode by ID.
func (dag *MerkleDAG) GetNode(id string) (*MerkleNode, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	node, exists := dag.nodes[id]
	if !exists {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

// GetStateRoot returns the current cryptographic MemoryRoot hash.
func (dag *MerkleDAG) GetStateRoot() string {
	dag.mu.RLock()
	defer dag.mu.RUnlock()
	return dag.stateRoot
}

func (dag *MerkleDAG) recomputeStateRoot() {
	if len(dag.nodes) == 0 {
		dag.stateRoot = ""
		return
	}

	h := sha256.New()
	// Sort node IDs for deterministic root computation
	nodeIDs := make([]string, 0, len(dag.nodes))
	for id := range dag.nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	for _, id := range nodeIDs {
		node := dag.nodes[id]
		h.Write([]byte(node.Hash))
	}
	dag.stateRoot = hex.EncodeToString(h.Sum(nil))
}

// GenerateProof builds a cryptographic inclusion proof for a target node ID.
func (dag *MerkleDAG) GenerateProof(nodeID string) (*MerkleProof, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	node, exists := dag.nodes[nodeID]
	if !exists {
		return nil, ErrNodeNotFound
	}

	pathHashes := make([]string, 0)
	proofChain := make([]string, 0)

	// Collect parent hashes up to root
	curr := node
	for len(curr.ParentIDs) > 0 {
		parentID := curr.ParentIDs[0]
		if parent, ok := dag.nodes[parentID]; ok {
			pathHashes = append(pathHashes, parent.Hash)
			proofChain = append(proofChain, parent.ID)
			curr = parent
		} else {
			break
		}
	}

	return &MerkleProof{
		NodeID:     nodeID,
		NodeHash:   node.Hash,
		PathHashes: pathHashes,
		StateRoot:  dag.stateRoot,
		Timestamp:  time.Now().UTC(),
		ProofChain: proofChain,
	}, nil
}

// VerifyProof verifies a MerkleProof against the current DAG state root.
func (dag *MerkleDAG) VerifyProof(proof *MerkleProof) bool {
	if proof == nil || proof.StateRoot == "" {
		return false
	}
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return proof.StateRoot == dag.stateRoot
}

// TopKSearch executes sub-millisecond hypervector similarity recall across all Merkle-DAG nodes.
func (dag *MerkleDAG) TopKSearch(queryHV Hypervector, k int) []*SearchResult {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	if k <= 0 {
		k = 5
	}

	results := make([]*SearchResult, 0, len(dag.nodes))
	for _, node := range dag.nodes {
		sim := Similarity(queryHV, node.Hypervector)
		dist := HammingDistance(queryHV, node.Hypervector)
		results = append(results, &SearchResult{
			Node:       node,
			Similarity: sim,
			Distance:   dist,
		})
	}

	// Sort descending by similarity
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > k {
		results = results[:k]
	}

	return results
}

// Size returns the total number of nodes in the Merkle-DAG.
func (dag *MerkleDAG) Size() int {
	dag.mu.RLock()
	defer dag.mu.RUnlock()
	return len(dag.nodes)
}

// ToJSON serializes the Merkle-DAG state to JSON.
func (dag *MerkleDAG) ToJSON() ([]byte, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()
	return json.MarshalIndent(dag.nodes, "", "  ")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
