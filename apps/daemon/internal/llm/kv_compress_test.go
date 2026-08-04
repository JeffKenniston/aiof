package llm

import (
	"math/rand"
	"testing"
)

func TestBlockTableAllocatorAllocationAndFork(t *testing.T) {
	alloc := NewBlockTableAllocator(32, 16, 8, 64)

	// Allocate block for seq-1
	blk1, err := alloc.AllocateBlock("seq-1")
	if err != nil {
		t.Fatalf("AllocateBlock failed: %v", err)
	}

	if blk1.RefCount.Load() != 1 {
		t.Errorf("Expected RefCount = 1, got %d", blk1.RefCount.Load())
	}

	// Fork sequence (zero-copy prefix sharing per ADR-01)
	forkedBT, err := alloc.ForkSequence("seq-1", "seq-2")
	if err != nil {
		t.Fatalf("ForkSequence failed: %v", err)
	}

	if len(forkedBT.BlockIDs) != 1 || forkedBT.BlockIDs[0] != blk1.ID {
		t.Errorf("Unexpected forked block IDs: %v", forkedBT.BlockIDs)
	}

	if blk1.RefCount.Load() != 2 {
		t.Errorf("Expected RefCount = 2 after fork, got %d", blk1.RefCount.Load())
	}

	// Free seq-1
	if err := alloc.FreeSequence("seq-1"); err != nil {
		t.Fatalf("FreeSequence seq-1 failed: %v", err)
	}

	if blk1.RefCount.Load() != 1 {
		t.Errorf("Expected RefCount = 1 after freeing one sequence, got %d", blk1.RefCount.Load())
	}

	// Free seq-2
	if err := alloc.FreeSequence("seq-2"); err != nil {
		t.Fatalf("FreeSequence seq-2 failed: %v", err)
	}

	if blk1.RefCount.Load() != 0 {
		t.Errorf("Expected RefCount = 0 after freeing all sequences, got %d", blk1.RefCount.Load())
	}
}

func TestINT4QuantizationPrecision(t *testing.T) {
	quantizer := NewINT4KVQuantizer()

	numElements := 512
	original := make([]float32, numElements)
	for i := 0; i < numElements; i++ {
		original[i] = (rand.Float32() * 4.0) - 2.0 // Range [-2.0, 2.0]
	}

	// Quantize to INT4 packed bytes
	packed, params, err := quantizer.QuantizeFP32ToINT4(original)
	if err != nil {
		t.Fatalf("QuantizeFP32ToINT4 failed: %v", err)
	}

	expectedBytes := (numElements + 1) / 2
	if len(packed) != expectedBytes {
		t.Errorf("Expected packed byte size %d, got %d", expectedBytes, len(packed))
	}

	// Dequantize back to FP32
	dequantized, err := quantizer.DequantizeINT4ToFP32(packed, numElements, params)
	if err != nil {
		t.Fatalf("DequantizeINT4ToFP32 failed: %v", err)
	}

	// Calculate Mean Squared Error (MSE)
	var mse float64
	for i := 0; i < numElements; i++ {
		diff := float64(original[i] - dequantized[i])
		mse += diff * diff
	}
	mse /= float64(numElements)

	t.Logf("INT4 Quantization MSE: %.6f (Scale: %.4f, ZP: %d)", mse, params.Scale, params.ZeroPoint)

	if mse > 0.05 {
		t.Errorf("INT4 Quantization MSE too high: %.6f (expected < 0.05)", mse)
	}
}

func TestMemoryFootprintReductionCalculation(t *testing.T) {
	res := CalculateKVMemoryFootprint(32, 32, 128, 4096)

	t.Logf("FP16 VRAM Footprint: %.2f MB", res.FP16MB)
	t.Logf("OpenVINO 2026.2 INT4 VRAM Footprint: %.2f MB", res.INT4MB)
	t.Logf("Calculated VRAM Reduction: %.2f%%", res.VRAMReductionPercent)

	if res.FP16MB <= 0 || res.INT4MB <= 0 {
		t.Fatalf("Invalid footprint values calculated")
	}

	if res.VRAMReductionPercent < 60.0 || res.VRAMReductionPercent > 80.0 {
		t.Errorf("VRAM reduction %.2f%% outside expected 60%%-80%% range", res.VRAMReductionPercent)
	}
}
