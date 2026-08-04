package agent

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCoWSandboxState(t *testing.T) {
	cow, err := NewCoWSandboxState("test-branch-1")
	if err != nil {
		t.Fatalf("Failed to create CoW state: %v", err)
	}

	call := DryRunToolCall{
		ToolName:        "file_read",
		InputParameters: map[string]any{"path": "main.go"},
		OutputPreview:   "package main",
		ExecutedAt:      time.Now(),
	}

	if err := cow.RecordDryRunToolCall(call); err != nil {
		t.Fatalf("Failed to record tool call: %v", err)
	}

	if err := cow.RecordFileDiff("main.go", "package main // updated"); err != nil {
		t.Fatalf("Failed to record file diff: %v", err)
	}

	// Commit CoW branch
	calls, diffs, err := cow.Commit()
	if err != nil {
		t.Fatalf("Failed to commit CoW branch: %v", err)
	}

	if len(calls) != 1 || calls[0].ToolName != "file_read" {
		t.Errorf("Unexpected committed tool calls: %v", calls)
	}

	if diffs["main.go"] != "package main // updated" {
		t.Errorf("Unexpected committed file diffs: %v", diffs)
	}

	// Double commit fails
	if _, _, err := cow.Commit(); !errors.Is(err, ErrShadowCommitted) {
		t.Errorf("Expected ErrShadowCommitted, got %v", err)
	}
}

func TestShadowAgentBranching_Confirmation(t *testing.T) {
	orchestrator := NewShadowAgentOrchestrator(1500 * time.Millisecond)
	ctx := context.Background()

	outcome, resolver := orchestrator.SpawnSpeculativeShadow(
		ctx,
		"Speculative Tool Prefetch",
		func(shadowCtx context.Context, cow *CoWSandboxState) error {
			_ = cow.RecordDryRunToolCall(DryRunToolCall{
				ToolName:      "mcp_search",
				OutputPreview: "found 5 items",
			})
			return nil
		},
	)

	// Primary planner confirms speculative intent
	finalOutcome, err := resolver(true)
	if err != nil {
		t.Fatalf("Resolver error on confirmation: %v", err)
	}

	if !finalOutcome.Committed {
		t.Errorf("Expected branch to be committed")
	}

	if !finalOutcome.Purged {
		// Note: in resolver logic when confirmed, Purged remains false
	}

	_ = outcome
}

func TestShadowAgentBranching_Rejection(t *testing.T) {
	orchestrator := NewShadowAgentOrchestrator(1500 * time.Millisecond)
	ctx := context.Background()

	_, resolver := orchestrator.SpawnSpeculativeShadow(
		ctx,
		"Speculative Unverified Action",
		func(shadowCtx context.Context, cow *CoWSandboxState) error {
			_ = cow.RecordFileDiff("temp.txt", "speculative text")
			return nil
		},
	)

	// Primary planner rejects speculative intent
	finalOutcome, err := resolver(false)
	if !errors.Is(err, ErrShadowRejected) {
		t.Errorf("Expected ErrShadowRejected, got %v", err)
	}

	if !finalOutcome.Purged {
		t.Errorf("Expected branch to be purged on rejection")
	}

	if finalOutcome.Committed {
		t.Errorf("Expected branch not to be committed")
	}
}

func TestShadowAgentBranching_TTLExceeded(t *testing.T) {
	orchestrator := NewShadowAgentOrchestrator(50 * time.Millisecond) // Short 50ms TTL for testing
	ctx := context.Background()

	_, resolver := orchestrator.SpawnSpeculativeShadow(
		ctx,
		"Slow Speculative Task",
		func(shadowCtx context.Context, cow *CoWSandboxState) error {
			select {
			case <-shadowCtx.Done():
				return shadowCtx.Err()
			case <-time.After(200 * time.Millisecond): // Exceeds 50ms TTL
				return nil
			}
		},
	)

	time.Sleep(100 * time.Millisecond)

	finalOutcome, err := resolver(true)
	if !errors.Is(err, ErrShadowTTLExceeded) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected ErrShadowTTLExceeded or context.DeadlineExceeded, got %v", err)
	}

	if !finalOutcome.Purged {
		t.Errorf("Expected expired shadow branch to be purged")
	}
}
