package llm

import (
	"context"
	"testing"
)

func TestSpeculativeDecodingAllAccepted(t *testing.T) {
	engine, err := NewFastDraftEngine(FastDraftConfig{Gamma: 4, Temperature: 0.7})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	prefix := []int{1, 2, 3}
	drafts := engine.GenerateDraftTokens(context.Background(), prefix)
	if len(drafts) != 4 {
		t.Fatalf("Expected 4 draft tokens, got %d", len(drafts))
	}

	result := engine.VerifyTokensAndAccept(context.Background(), prefix, drafts)
	if len(result.AcceptedTokens) == 0 {
		t.Errorf("Expected at least 1 accepted token, got 0")
	}

	stats := engine.GetStats()
	if stats["total_steps"].(uint64) != 1 {
		t.Errorf("Expected 1 total step, got %v", stats["total_steps"])
	}
}
