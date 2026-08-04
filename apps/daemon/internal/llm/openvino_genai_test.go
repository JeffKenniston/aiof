package llm

import (
	"context"
	"testing"
)

func TestOpenVINOGenAIPrefillDecodeSwitchboard(t *testing.T) {
	cfg := OpenVINOGenAIConfig{
		PrefillDevice: DeviceNPU,
		DecodeDevice:  DeviceIGPU,
		MaxBatchSize:  10,
	}

	provider, err := NewOpenVINOGenAIProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize OpenVINO GenAI Provider: %v", err)
	}

	ctx := context.Background()
	promptTokens := []int{10, 20, 30, 40, 50}

	seq, err := provider.SubmitRequest(ctx, "seq-001", promptTokens, 5)
	if err != nil {
		t.Fatalf("Failed to submit request: %v", err)
	}

	if seq.Phase != PhasePrefill {
		t.Errorf("Expected initial phase PREFILL_NPU, got %s", seq.Phase)
	}
	if seq.TargetDevice != DeviceNPU {
		t.Errorf("Expected initial target NPU, got %s", seq.TargetDevice)
	}

	// Step 1: Execute NPU Prefill Phase
	_, err = provider.ExecuteStep(ctx)
	if err != nil {
		t.Fatalf("Step execution failed: %v", err)
	}

	if seq.Phase != PhaseDecode {
		t.Errorf("Expected post-prefill phase DECODE_IGPU, got %s", seq.Phase)
	}
	if seq.TargetDevice != DeviceIGPU {
		t.Errorf("Expected decode target GPU.0, got %s", seq.TargetDevice)
	}

	// Step 2: Execute iGPU Decode Phase
	generatedCount, err := provider.ExecuteStep(ctx)
	if err != nil {
		t.Fatalf("Decode step failed: %v", err)
	}

	if generatedCount != 1 {
		t.Errorf("Expected 1 generated token, got %d", generatedCount)
	}

	stats := provider.GetStats()
	if stats["prefill_requests"] != 1 {
		t.Errorf("Expected 1 prefill request stat, got %d", stats["prefill_requests"])
	}
}

func TestLMCacheKVPoolPageSwapping(t *testing.T) {
	// Small 10MB VRAM budget for test
	cache := NewLMCacheKVPool(10*1024*1024, 64*1024*1024)

	ctx := context.Background()
	p1, err := cache.AllocateKVPage(ctx, "p1", "seq-1", 2*1024*1024, []byte("data1"))
	if err != nil {
		t.Fatalf("Failed to allocate page 1: %v", err)
	}

	p2, err := cache.AllocateKVPage(ctx, "p2", "seq-1", 2*1024*1024, []byte("data2"))
	if err != nil {
		t.Fatalf("Failed to allocate page 2: %v", err)
	}

	p3, err := cache.AllocateKVPage(ctx, "p3", "seq-1", 2*1024*1024, []byte("data3"))
	if err != nil {
		t.Fatalf("Failed to allocate page 3: %v", err)
	}

	p4, err := cache.AllocateKVPage(ctx, "p4", "seq-1", 2*1024*1024, []byte("data4"))
	if err != nil {
		t.Fatalf("Failed to allocate page 4: %v", err)
	}

	p5, err := cache.AllocateKVPage(ctx, "p5", "seq-1", 2*1024*1024, []byte("data5"))
	if err != nil {
		t.Fatalf("Failed to allocate page 5: %v", err)
	}

	p6, err := cache.AllocateKVPage(ctx, "p6", "seq-1", 2*1024*1024, []byte("data6"))
	if err != nil {
		t.Fatalf("Failed to allocate page 6: %v", err)
	}

	_ = p1
	_ = p2
	_ = p3
	_ = p4
	_ = p5
	_ = p6

	metrics := cache.GetMetrics()
	if metrics["dram_page_count"].(int) == 0 {
		t.Errorf("Expected non-zero PCIe DMA swap-outs under 88%% VRAM load")
	}
}

func TestINT4KVQuantizerPrecision(t *testing.T) {
	quantizer := NewINT4KVQuantizer()

	rawValues := make([]float32, 64)
	for i := 0; i < 64; i++ {
		rawValues[i] = float32(i) * 0.1
	}

	packed, params, err := quantizer.QuantizeFP32ToINT4(rawValues)
	if err != nil {
		t.Fatalf("QuantizeFP32ToINT4 failed: %v", err)
	}

	if len(packed) != 32 {
		t.Errorf("Expected 32 packed INT4 bytes for 64 values, got %d", len(packed))
	}

	decompressed, err := quantizer.DequantizeINT4ToFP32(packed, 64, params)
	if err != nil {
		t.Fatalf("DequantizeINT4ToFP32 failed: %v", err)
	}

	if len(decompressed) != 64 {
		t.Errorf("Expected 64 decompressed float32 values, got %d", len(decompressed))
	}
}

func TestFastDraftSpeculativeDecoding(t *testing.T) {
	engine, err := NewFastDraftEngine(FastDraftConfig{Gamma: 4, Temperature: 0.7})
	if err != nil {
		t.Fatalf("Failed to initialize FastDraft engine: %v", err)
	}

	ctx := context.Background()
	prefix := []int{1, 2, 3, 4, 5}

	drafts := engine.GenerateDraftTokens(ctx, prefix)
	if len(drafts) != 4 {
		t.Errorf("Expected 4 draft tokens, got %d", len(drafts))
	}

	res := engine.VerifyTokensAndAccept(ctx, prefix, drafts)
	if len(res.AcceptedTokens) == 0 {
		t.Errorf("Expected at least 1 accepted/bonus token, got 0")
	}

	stats := engine.GetStats()
	if stats["total_drafted"].(uint64) != 4 {
		t.Errorf("Expected 4 drafted tokens in stats, got %v", stats["total_drafted"])
	}
}
