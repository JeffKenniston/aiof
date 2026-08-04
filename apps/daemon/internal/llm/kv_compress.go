package llm

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
)

// ============================================================================
// ADR-03: OpenVINO 2026.2 INT4 Paged Attention KV Cache Compression Engine
// ============================================================================

const (
	DefaultBlockTokens = 16 // Standard OpenVINO 2026.2 paged attention block size (16 tokens)
	BitsPerINT4Element = 4  // 4 bits per element (2 elements packed per byte)
	BytesPerFP16       = 2  // 16 bits = 2 bytes
	BytesPerFP32       = 4  // 32 bits = 4 bytes
)

var (
	ErrBlockTableFull    = errors.New("kv_compress: physical block pool exhausted")
	ErrInvalidBlockID    = errors.New("kv_compress: invalid physical block ID")
	ErrDimensionMismatch = errors.New("kv_compress: tensor dimension mismatch for quantization")
	ErrNilTensor         = errors.New("kv_compress: input tensor slice cannot be nil or empty")
)

// QuantParams holds asymmetric INT4 scale and zero-point parameters per channel/head.
type QuantParams struct {
	Scale     float32 `json:"scale"`
	ZeroPoint int32   `json:"zero_point"` // Range [0, 15]
}

// INT4QuantBlock stores 4-bit packed Key and Value tensors along with scale/zero-point metadata.
type INT4QuantBlock struct {
	KeyPacked   []byte        `json:"key_packed"`   // 2 nibbles per byte
	ValuePacked []byte        `json:"value_packed"` // 2 nibbles per byte
	KeyParams   []QuantParams `json:"key_params"`   // Per-head scale/zero-point
	ValueParams []QuantParams `json:"value_params"` // Per-head scale/zero-point
}

// PhysicalBlock represents a single paged KV memory block allocated in physical VRAM/DRAM.
type PhysicalBlock struct {
	ID         int32          `json:"id"`
	RefCount   atomic.Int32   `json:"ref_count"`
	TokenCount int            `json:"token_count"`
	NumHeads   int            `json:"num_heads"`
	HeadDim    int            `json:"head_dim"`
	Data       INT4QuantBlock `json:"data"`
}

// BlockTable maps logical sequence token indices to physical memory block IDs.
type BlockTable struct {
	SequenceID string  `json:"sequence_id"`
	BlockIDs   []int32 `json:"block_ids"`
}

// BlockTableAllocator manages physical paged memory blocks and sequence mappings per ADR-03.
type BlockTableAllocator struct {
	mu            sync.RWMutex
	totalBlocks   int
	blockSize     int // Tokens per block (default 16)
	numHeads      int
	headDim       int
	freeBlocks    []int32
	physicalPool  map[int32]*PhysicalBlock
	sequenceTable map[string]*BlockTable
}

// NewBlockTableAllocator initializes an OpenVINO 2026.2 paged block allocator.
func NewBlockTableAllocator(totalBlocks, blockSize, numHeads, headDim int) *BlockTableAllocator {
	if blockSize <= 0 {
		blockSize = DefaultBlockTokens
	}
	if totalBlocks <= 0 {
		totalBlocks = 1024
	}

	alloc := &BlockTableAllocator{
		totalBlocks:   totalBlocks,
		blockSize:     blockSize,
		numHeads:      numHeads,
		headDim:       headDim,
		freeBlocks:    make([]int32, totalBlocks),
		physicalPool:  make(map[int32]*PhysicalBlock, totalBlocks),
		sequenceTable: make(map[string]*BlockTable),
	}

	for i := 0; i < totalBlocks; i++ {
		id := int32(i)
		alloc.freeBlocks[i] = id
		alloc.physicalPool[id] = &PhysicalBlock{
			ID:         id,
			NumHeads:   numHeads,
			HeadDim:    headDim,
			TokenCount: 0,
		}
	}

	return alloc
}

// AllocateBlock allocates a physical block for a given sequence ID.
func (a *BlockTableAllocator) AllocateBlock(seqID string) (*PhysicalBlock, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.freeBlocks) == 0 {
		return nil, ErrBlockTableFull
	}

	// Pop free block from stack
	blockID := a.freeBlocks[len(a.freeBlocks)-1]
	a.freeBlocks = a.freeBlocks[:len(a.freeBlocks)-1]

	block := a.physicalPool[blockID]
	block.RefCount.Store(1)
	block.TokenCount = 0

	// Allocate INT4 packed data buffers
	elementsPerBlock := a.blockSize * a.numHeads * a.headDim
	packedBytes := (elementsPerBlock + 1) / 2 // 2 elements per byte

	block.Data = INT4QuantBlock{
		KeyPacked:   make([]byte, packedBytes),
		ValuePacked: make([]byte, packedBytes),
		KeyParams:   make([]QuantParams, a.numHeads),
		ValueParams: make([]QuantParams, a.numHeads),
	}

	// Register in sequence table
	seqBT, exists := a.sequenceTable[seqID]
	if !exists {
		seqBT = &BlockTable{
			SequenceID: seqID,
			BlockIDs:   make([]int32, 0, 4),
		}
		a.sequenceTable[seqID] = seqBT
	}
	seqBT.BlockIDs = append(seqBT.BlockIDs, blockID)

	return block, nil
}

// ForkSequence performs zero-copy prefix sharing by incrementing reference counts ("Prefill Once, Fan Out" ADR-01).
func (a *BlockTableAllocator) ForkSequence(srcSeqID, dstSeqID string) (*BlockTable, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	srcTable, exists := a.sequenceTable[srcSeqID]
	if !exists {
		return nil, ErrSequenceNotFound
	}

	dstTable := &BlockTable{
		SequenceID: dstSeqID,
		BlockIDs:   make([]int32, len(srcTable.BlockIDs)),
	}
	copy(dstTable.BlockIDs, srcTable.BlockIDs)

	for _, blockID := range dstTable.BlockIDs {
		if block, ok := a.physicalPool[blockID]; ok {
			block.RefCount.Add(1)
		}
	}

	a.sequenceTable[dstSeqID] = dstTable
	return dstTable, nil
}

// FreeSequence frees sequence block mappings and reclaims physical blocks when reference counts reach zero.
func (a *BlockTableAllocator) FreeSequence(seqID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	table, exists := a.sequenceTable[seqID]
	if !exists {
		return ErrSequenceNotFound
	}

	for _, blockID := range table.BlockIDs {
		if block, ok := a.physicalPool[blockID]; ok {
			if block.RefCount.Add(-1) <= 0 {
				block.TokenCount = 0
				block.Data = INT4QuantBlock{}
				a.freeBlocks = append(a.freeBlocks, blockID)
			}
		}
	}

	delete(a.sequenceTable, seqID)
	return nil
}

// GetBlockTable returns the logical-to-physical block table for a sequence.
func (a *BlockTableAllocator) GetBlockTable(seqID string) ([]int32, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	table, exists := a.sequenceTable[seqID]
	if !exists {
		return nil, ErrSequenceNotFound
	}
	res := make([]int32, len(table.BlockIDs))
	copy(res, table.BlockIDs)
	return res, nil
}

// GetPhysicalBlock retrieves a physical block by ID.
func (a *BlockTableAllocator) GetPhysicalBlock(blockID int32) (*PhysicalBlock, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	block, ok := a.physicalPool[blockID]
	if !ok {
		return nil, ErrInvalidBlockID
	}
	return block, nil
}

// ============================================================================
// Asymmetric INT4 Quantization Engine (4-bit Packed Nibbles)
// ============================================================================

type INT4KVQuantizer struct{}

func NewINT4KVQuantizer() *INT4KVQuantizer {
	return &INT4KVQuantizer{}
}

// QuantizeFP32ToINT4 quantizes flat FP32 tensor data into 4-bit packed nibble byte slices (2 elements per byte).
// Returns packed byte slice and QuantParams (scale, zero-point).
func (q *INT4KVQuantizer) QuantizeFP32ToINT4(data []float32) ([]byte, QuantParams, error) {
	if len(data) == 0 {
		return nil, QuantParams{}, ErrNilTensor
	}

	minVal := float32(math.MaxFloat32)
	maxVal := float32(-math.MaxFloat32)

	for _, v := range data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	if minVal == maxVal {
		maxVal = minVal + 1e-5
	}

	scale := (maxVal - minVal) / 15.0
	if scale == 0 {
		scale = 1.0
	}

	zpFloat := math.Round(float64(-minVal / scale))
	zp := int32(math.Max(0, math.Min(15, zpFloat)))

	packedSize := (len(data) + 1) / 2
	packed := make([]byte, packedSize)

	for i, v := range data {
		qValFloat := math.Round(float64(v/scale)) + float64(zp)
		qVal := byte(math.Max(0, math.Min(15, qValFloat))) & 0x0F

		byteIdx := i / 2
		if i%2 == 0 {
			packed[byteIdx] = (packed[byteIdx] & 0xF0) | qVal
		} else {
			packed[byteIdx] = (packed[byteIdx] & 0x0F) | (qVal << 4)
		}
	}

	return packed, QuantParams{Scale: scale, ZeroPoint: zp}, nil
}

// DequantizeINT4ToFP32 dequantizes packed 4-bit nibbles back to FP32 tensor data.
func (q *INT4KVQuantizer) DequantizeINT4ToFP32(packed []byte, numElements int, params QuantParams) ([]float32, error) {
	if len(packed) == 0 || numElements <= 0 {
		return nil, ErrNilTensor
	}

	output := make([]float32, numElements)

	for i := 0; i < numElements; i++ {
		byteIdx := i / 2
		if byteIdx >= len(packed) {
			break
		}

		var qVal byte
		if i%2 == 0 {
			qVal = packed[byteIdx] & 0x0F
		} else {
			qVal = (packed[byteIdx] >> 4) & 0x0F
		}

		output[i] = (float32(qVal) - float32(params.ZeroPoint)) * params.Scale
	}

	return output, nil
}

// ============================================================================
// OpenVINO 2026.2 Memory Footprint Calculator (ADR-03)
// ============================================================================

type MemoryFootprintResult struct {
	NumLayers             int     `json:"num_layers"`
	NumHeads              int     `json:"num_heads"`
	HeadDim               int     `json:"head_dim"`
	SequenceLength        int     `json:"sequence_length"`
	FP16SizeBytes         int64   `json:"fp16_size_bytes"`
	INT8SizeBytes         int64   `json:"int8_size_bytes"`
	OpenVINOINT4SizeBytes int64   `json:"openvino_int4_size_bytes"`
	FP16MB                float64 `json:"fp16_mb"`
	INT4MB                float64 `json:"int4_mb"`
	VRAMReductionPercent  float64 `json:"vram_reduction_percent"`
}

// CalculateKVMemoryFootprint computes VRAM requirements across FP16, INT8, and OpenVINO 2026.2 INT4 Paged Attention.
func CalculateKVMemoryFootprint(numLayers, numHeads, headDim, seqLen int) MemoryFootprintResult {
	if numLayers <= 0 {
		numLayers = 32
	}
	if numHeads <= 0 {
		numHeads = 32
	}
	if headDim <= 0 {
		headDim = 128
	}
	if seqLen <= 0 {
		seqLen = 4096
	}

	totalElements := int64(2) * int64(numLayers) * int64(numHeads) * int64(headDim) * int64(seqLen)

	fp16Bytes := totalElements * int64(BytesPerFP16)
	int8Bytes := totalElements*1 + (int64(numLayers*numHeads) * 4)

	rawINT4Bytes := (totalElements + 1) / 2
	numBlocks := (seqLen + DefaultBlockTokens - 1) / DefaultBlockTokens
	metadataBytes := int64(2) * int64(numLayers) * int64(numHeads) * int64(numBlocks) * 8
	pagedPaddingBytes := int64(float64(rawINT4Bytes) * 0.02)

	openvinoINT4Bytes := rawINT4Bytes + metadataBytes + pagedPaddingBytes
	reduction := (1.0 - (float64(openvinoINT4Bytes) / float64(fp16Bytes))) * 100.0

	return MemoryFootprintResult{
		NumLayers:             numLayers,
		NumHeads:              numHeads,
		HeadDim:               headDim,
		SequenceLength:        seqLen,
		FP16SizeBytes:         fp16Bytes,
		INT8SizeBytes:         int8Bytes,
		OpenVINOINT4SizeBytes: openvinoINT4Bytes,
		FP16MB:                float64(fp16Bytes) / (1024 * 1024),
		INT4MB:                float64(openvinoINT4Bytes) / (1024 * 1024),
		VRAMReductionPercent:  reduction,
	}
}
