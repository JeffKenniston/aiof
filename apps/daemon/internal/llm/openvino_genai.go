package llm

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// ADR-02 & ADR-16: OpenVINO GenAI Provider & Heterogeneous Silicon Switchboard
// ============================================================================

var (
	ErrDeviceUnavailable = errors.New("openvino: specified hardware acceleration device unavailable")
	ErrSequenceNotFound  = errors.New("openvino: sequence ID not found in active batch")
	ErrBatchFull         = errors.New("openvino: continuous batch capacity exceeded")
)

// ExecutionPhase identifies whether a request is in Prefill or Decode phase.
type ExecutionPhase int

const (
	PhasePrefill ExecutionPhase = iota
	PhaseDecode
)

func (p ExecutionPhase) String() string {
	if p == PhasePrefill {
		return "PREFILL_NPU"
	}
	return "DECODE_IGPU"
}

// SequenceState represents a single request sequence managed by Continuous Batching.
type SequenceState struct {
	SequenceID   string
	PromptTokens []int
	GenTokens    []int
	Phase        ExecutionPhase
	TargetDevice string
	PrefixHash   uint64
	BlockIDs     []int
	MaxTokens    int
	Temperature  float64
	Finished     bool
	CreatedAt    time.Time
}

// OpenVINOGenAIConfig configures the OpenVINO GenAI heterogeneous switchboard.
type OpenVINOGenAIConfig struct {
	ModelPath            string
	PrefillDevice        string // Default: "NPU"
	DecodeDevice         string // Default: "GPU.0"
	MaxBatchSize         int
	BlockSize            int // PagedAttention block size (default 16)
	EnablePrefixCaching  bool
	EnableSpeculativeDec bool
}

// PrefixCacheNode represents a node in the Radix Prefix Caching tree.
type PrefixCacheNode struct {
	Tokens   []int
	BlockID  int
	RefCount int
	Children map[int]*PrefixCacheNode
}

// PrefixCacheManager manages prompt prefix caching across requests.
type PrefixCacheManager struct {
	mu          sync.RWMutex
	root        *PrefixCacheNode
	blockAlloc  atomic.Int32
	cachedCount atomic.Uint64
}

func NewPrefixCacheManager() *PrefixCacheManager {
	return &PrefixCacheManager{
		root: &PrefixCacheNode{
			Children: make(map[int]*PrefixCacheNode),
		},
	}
}

func (pcm *PrefixCacheManager) MatchPrefix(tokens []int) ([]int, int) {
	pcm.mu.RLock()
	defer pcm.mu.RUnlock()

	curr := pcm.root
	matchedBlocks := make([]int, 0)
	matchedTokens := 0

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		next, exists := curr.Children[tok]
		if !exists {
			break
		}
		curr = next
		matchedTokens++
		if curr.BlockID != -1 {
			matchedBlocks = append(matchedBlocks, curr.BlockID)
		}
	}

	if matchedTokens > 0 {
		pcm.cachedCount.Add(uint64(matchedTokens))
	}
	return matchedBlocks, matchedTokens
}

func (pcm *PrefixCacheManager) InsertPrefix(tokens []int, blockID int) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	curr := pcm.root
	for _, tok := range tokens {
		next, exists := curr.Children[tok]
		if !exists {
			next = &PrefixCacheNode{
				Tokens:   []int{tok},
				BlockID:  -1,
				Children: make(map[int]*PrefixCacheNode),
			}
			curr.Children[tok] = next
		}
		curr = next
	}
	curr.BlockID = blockID
	curr.RefCount++
}

// ContinuousBatchScheduler orchestrates continuous batching across NPU and iGPU.
type ContinuousBatchScheduler struct {
	mu           sync.Mutex
	activeSeqs   map[string]*SequenceState
	prefillQueue []*SequenceState
	decodeQueue  []*SequenceState
	maxBatchSize int
	prefixCache  *PrefixCacheManager
}

func NewContinuousBatchScheduler(maxBatch int, pcm *PrefixCacheManager) *ContinuousBatchScheduler {
	return &ContinuousBatchScheduler{
		activeSeqs:   make(map[string]*SequenceState),
		prefillQueue: make([]*SequenceState, 0),
		decodeQueue:  make([]*SequenceState, 0),
		maxBatchSize: maxBatch,
		prefixCache:  pcm,
	}
}

func (cbs *ContinuousBatchScheduler) AddSequence(seq *SequenceState) error {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	if len(cbs.activeSeqs) >= cbs.maxBatchSize {
		return ErrBatchFull
	}

	seq.Phase = PhasePrefill
	seq.TargetDevice = DeviceNPU // Offload prefill to Intel AI Boost NPU per ADR-02

	if cbs.prefixCache != nil {
		matchedBlocks, matchedTokens := cbs.prefixCache.MatchPrefix(seq.PromptTokens)
		if matchedTokens > 0 {
			seq.BlockIDs = matchedBlocks
		}
	}

	cbs.activeSeqs[seq.SequenceID] = seq
	cbs.prefillQueue = append(cbs.prefillQueue, seq)
	return nil
}

func (cbs *ContinuousBatchScheduler) Step() ([]*SequenceState, []*SequenceState) {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	// 1. Process NPU Prefill Batch
	prefillBatch := make([]*SequenceState, len(cbs.prefillQueue))
	copy(prefillBatch, cbs.prefillQueue)
	cbs.prefillQueue = cbs.prefillQueue[:0]

	// Transition completed prefills to iGPU Decode Phase
	for _, seq := range prefillBatch {
		seq.Phase = PhaseDecode
		seq.TargetDevice = DeviceIGPU // Route decode phase to Intel Arc 140V iGPU
		cbs.decodeQueue = append(cbs.decodeQueue, seq)
	}

	// 2. Process iGPU Decode Batch
	decodeBatch := make([]*SequenceState, 0, len(cbs.decodeQueue))
	nextDecodeQueue := make([]*SequenceState, 0, len(cbs.decodeQueue))

	for _, seq := range cbs.decodeQueue {
		if !seq.Finished {
			decodeBatch = append(decodeBatch, seq)
			nextDecodeQueue = append(nextDecodeQueue, seq)
		} else {
			delete(cbs.activeSeqs, seq.SequenceID)
		}
	}
	cbs.decodeQueue = nextDecodeQueue

	return prefillBatch, decodeBatch
}

// OpenVINOGenAIProvider is the primary switchboard implementing ADR-02 & ADR-16.
type OpenVINOGenAIProvider struct {
	config      OpenVINOGenAIConfig
	scheduler   *ContinuousBatchScheduler
	prefixCache *PrefixCacheManager
	stats       struct {
		PrefillCount atomic.Uint64
		DecodeTokens atomic.Uint64
		NPULoad      atomic.Uint64
		iGPULoad     atomic.Uint64
	}
}

func NewOpenVINOGenAIProvider(cfg OpenVINOGenAIConfig) (*OpenVINOGenAIProvider, error) {
	if cfg.PrefillDevice == "" {
		cfg.PrefillDevice = DeviceNPU
	}
	if cfg.DecodeDevice == "" {
		cfg.DecodeDevice = DeviceIGPU
	}
	if cfg.MaxBatchSize <= 0 {
		cfg.MaxBatchSize = 64
	}
	if cfg.BlockSize <= 0 {
		cfg.BlockSize = 16
	}

	pcm := NewPrefixCacheManager()
	cbs := NewContinuousBatchScheduler(cfg.MaxBatchSize, pcm)

	p := &OpenVINOGenAIProvider{
		config:      cfg,
		scheduler:   cbs,
		prefixCache: pcm,
	}

	return p, nil
}

func (p *OpenVINOGenAIProvider) SubmitRequest(ctx context.Context, seqID string, promptTokens []int, maxTokens int) (*SequenceState, error) {
	seq := &SequenceState{
		SequenceID:   seqID,
		PromptTokens: promptTokens,
		GenTokens:    make([]int, 0),
		MaxTokens:    maxTokens,
		CreatedAt:    time.Now(),
	}

	if err := p.scheduler.AddSequence(seq); err != nil {
		return nil, err
	}

	p.stats.PrefillCount.Add(1)
	p.stats.NPULoad.Add(uint64(len(promptTokens)))
	return seq, nil
}

func (p *OpenVINOGenAIProvider) ExecuteStep(ctx context.Context) (int, error) {
	prefillBatch, decodeBatch := p.scheduler.Step()

	// Simulate Step Processing on Heterogeneous Targets
	for _, seq := range prefillBatch {
		// NPU prefill complete -> append initial token
		seq.GenTokens = append(seq.GenTokens, 101) // Token placeholder
	}

	generatedCount := 0
	for _, seq := range decodeBatch {
		if len(seq.GenTokens) >= seq.MaxTokens {
			seq.Finished = true
		} else {
			seq.GenTokens = append(seq.GenTokens, 200+len(seq.GenTokens))
			generatedCount++
			p.stats.DecodeTokens.Add(1)
			p.stats.iGPULoad.Add(1)
		}
	}

	return generatedCount, nil
}

func (p *OpenVINOGenAIProvider) GetStats() map[string]uint64 {
	return map[string]uint64{
		"prefill_requests": p.stats.PrefillCount.Load(),
		"decode_tokens":    p.stats.DecodeTokens.Load(),
		"npu_prefill_load": p.stats.NPULoad.Load(),
		"igpu_decode_load": p.stats.iGPULoad.Load(),
		"prefix_hits":      p.prefixCache.cachedCount.Load(),
	}
}
