package llm

import (
	"context"
	"testing"
)

func TestLMCacheHighWatermarkEviction(t *testing.T) {
	_ = uint64(1024 * 1024)                // 1 MB page size (reserved for future multi-tier tests)
	totalVRAM := uint64(10 * 1024 * 1024)  // 10 MB total VRAM
	totalDRAM := uint64(20 * 1024 * 1024)  // 20 MB total DRAM

	cache := NewLMCacheManager(totalVRAM, totalDRAM)
	ctx := context.Background()

	// Allocate pages until VRAM watermark > 88%
	for i := 0; i < 5; i++ {
		_, err := cache.AllocatePage(ctx, 100, true)
		if err != nil {
			t.Fatalf("Allocation failed for page %d: %v", i, err)
		}
	}

	stats := cache.GetStats()
	if stats["allocated_pages"].(int) == 0 {
		t.Errorf("Expected allocated pages > 0")
	}
}

func TestLMCacheTouchAndFree(t *testing.T) {
	cache := NewLMCacheManager(4*1024*1024, 10*1024*1024)
	ctx := context.Background()

	page, err := cache.AllocatePage(ctx, 50, false)
	if err != nil {
		t.Fatalf("AllocatePage failed: %v", err)
	}

	if err := cache.TouchPage(page.PageID); err != nil {
		t.Fatalf("TouchPage failed: %v", err)
	}

	cache.FreePage(page.PageID)
	stats := cache.GetStats()
	if stats["allocated_pages"].(int) != 0 {
		t.Errorf("Expected 0 allocated pages after free, got %v", stats["allocated_pages"])
	}
}
