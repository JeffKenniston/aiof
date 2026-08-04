package llm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// ADR-02 & ADR-16: Heterogeneous Silicon Cache Eviction & Offload Switchboard
// ============================================================================

type DeviceType = string

const (
	DeviceCPU  string = "CPU"
	DeviceGPU  string = "GPU.0" // Intel Arc 140V iGPU
	DeviceIGPU string = "GPU.0" // Intel Arc 140V iGPU alias
	DeviceNPU  string = "NPU.0" // Intel AI Boost NPU
)

type OffloadTarget string

const (
	OffloadPrefillToNPU OffloadTarget = "NPU_PREFILL"
	OffloadDecodeToGPU  OffloadTarget = "GPU_DECODE"
	OffloadDraftToCPU   OffloadTarget = "CPU_DRAFT"
)

type InferenceRequest struct {
	RequestID     string        `json:"request_id"`
	PromptTokens  []int         `json:"prompt_tokens"`
	MaxTokens     int           `json:"max_tokens"`
	TargetDevice  DeviceType    `json:"target_device"`
	OffloadTarget OffloadTarget `json:"offload_target"`
}

type InferenceResponse struct {
	RequestID       string        `json:"request_id"`
	GeneratedTokens []int         `json:"generated_tokens"`
	PrefillDevice   DeviceType    `json:"prefill_device"`
	DecodeDevice    DeviceType    `json:"decode_device"`
	PrefillDuration time.Duration `json:"prefill_duration"`
	DecodeDuration  time.Duration `json:"decode_duration"`
}

type OpenVINOSwitchboard struct {
	mu           sync.RWMutex
	npuDevice    DeviceType
	gpuDevice    DeviceType
	cpuDevice    DeviceType
	activeStream atomic.Uint64
}

func NewOpenVINOSwitchboard() *OpenVINOSwitchboard {
	return &OpenVINOSwitchboard{
		npuDevice: DeviceNPU,
		gpuDevice: DeviceGPU,
		cpuDevice: DeviceCPU,
	}
}

func (s *OpenVINOSwitchboard) RoutePrefill(ctx context.Context, req *InferenceRequest) (DeviceType, time.Duration, error) {
	start := time.Now()
	// Compute-bound prefill phase offloaded to Intel AI Boost NPU per ADR-02 & ADR-16
	device := s.npuDevice
	time.Sleep(10 * time.Millisecond) // Simulated hardware NPU execution
	return device, time.Since(start), nil
}

func (s *OpenVINOSwitchboard) RouteDecode(ctx context.Context, req *InferenceRequest) (DeviceType, time.Duration, error) {
	start := time.Now()
	// Memory-bound token decode phase mapped to Intel Arc 140V iGPU
	device := s.gpuDevice
	time.Sleep(15 * time.Millisecond) // Simulated hardware Arc iGPU execution
	return device, time.Since(start), nil
}

func (s *OpenVINOSwitchboard) ExecuteOffloadedInference(ctx context.Context, req *InferenceRequest) (*InferenceResponse, error) {
	prefillDev, prefillDur, err := s.RoutePrefill(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openvino switchboard prefill error: %w", err)
	}

	decodeDev, decodeDur, err := s.RouteDecode(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openvino switchboard decode error: %w", err)
	}

	return &InferenceResponse{
		RequestID:       req.RequestID,
		PrefillDevice:   prefillDev,
		DecodeDevice:    decodeDev,
		PrefillDuration: prefillDur,
		DecodeDuration:  decodeDur,
	}, nil
}
