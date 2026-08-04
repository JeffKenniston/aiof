package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

type mockAgent struct {
	response string
}

func (m *mockAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	return m.response, nil
}

func TestRouteLLMClassifier(t *testing.T) {
	classifier := NewRouteLLMClassifier()

	simplePrompt := "What is the capital of France?"
	complexPrompt := "Refactor the AST symbol graph to verify invariants, solve deadlocks, and optimize memory allocations.\n```go\nfunc Fix() {}\n```"

	simpleScore := classifier.PredictComplexity(simplePrompt)
	complexScore := classifier.PredictComplexity(complexPrompt)

	if simpleScore >= complexScore {
		t.Errorf("Expected complex prompt score (%f) > simple prompt score (%f)", complexScore, simpleScore)
	}

	if complexScore < 0.50 {
		t.Errorf("Expected complex prompt score >= 0.50, got %f", complexScore)
	}
}

func TestSetValuedRouter(t *testing.T) {
	kvManager := NewKVSnapshotManager(5 * time.Minute)
	router := NewSetValuedRouter(nil, kvManager)

	// Route simple query
	resSimple, err := router.RoutePrompt("Hello, write a simple hello world function in Go.")
	if err != nil {
		t.Fatalf("Failed to route simple prompt: %v", err)
	}

	if resSimple.ComplexityScore < 0.0 || resSimple.ComplexityScore > 1.0 {
		t.Errorf("Invalid complexity score: %f", resSimple.ComplexityScore)
	}

	// Route complex multi-domain refactoring query
	complexPrompt := "Analyze AST symbols, verify proof invariants using SMT z3 solver, refactor code, and generate SBOM audit reports.\n```go\ntype Node struct {}\n```"
	resComplex, err := router.RoutePrompt(complexPrompt)
	if err != nil {
		t.Fatalf("Failed to route complex prompt: %v", err)
	}

	if len(resComplex.TargetAgents) == 0 {
		t.Errorf("Expected target agents for complex prompt, got 0")
	}

	if resComplex.KVSnapshotID == "" {
		t.Errorf("Expected valid KVSnapshotID for fan-out dispatch")
	}
}

func TestSetValuedRouterParallelDispatch(t *testing.T) {
	router := NewSetValuedRouter(nil, nil)
	router.RegisterAgent("developer", &mockAgent{response: "developer_output\n"})
	router.RegisterAgent("architect", &mockAgent{response: "architect_output\n"})

	seq := CognitiveSequence{
		TargetNodes: []string{"developer", "architect"},
	}

	res, err := router.Dispatch(context.Background(), nil, "test input", seq)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	if !strings.Contains(res, "developer_output") || !strings.Contains(res, "architect_output") {
		t.Errorf("Dispatch output missing parallel agent results: %s", res)
	}
}

func TestKVSnapshotSharing(t *testing.T) {
	kvManager := NewKVSnapshotManager(2 * time.Minute)
	prompt := "Shared prefill prompt for multi-agent fan-out"
	promptHash := "hash1234567890"

	snap1, err := kvManager.AcquireOrCreateSnapshot(promptHash, prompt)
	if err != nil {
		t.Fatalf("Failed to create KV snapshot: %v", err)
	}

	if snap1.RefCount.Load() != 1 {
		t.Errorf("Expected ref count 1, got %d", snap1.RefCount.Load())
	}

	snap2, err := kvManager.AcquireOrCreateSnapshot(promptHash, prompt)
	if err != nil {
		t.Fatalf("Failed to acquire KV snapshot: %v", err)
	}

	if snap2.ID != snap1.ID {
		t.Errorf("Expected same snapshot ID %s, got %s", snap1.ID, snap2.ID)
	}

	if snap2.RefCount.Load() != 2 {
		t.Errorf("Expected ref count 2, got %d", snap2.RefCount.Load())
	}

	if err := kvManager.ReleaseDecrement(snap1.ID); err != nil {
		t.Errorf("Release failed: %v", err)
	}

	if kvManager.ActiveSnapshotCount() != 1 {
		t.Errorf("Expected 1 active snapshot, got %d", kvManager.ActiveSnapshotCount())
	}

	if err := kvManager.ReleaseDecrement(snap1.ID); err != nil {
		t.Errorf("Release failed: %v", err)
	}

	if kvManager.ActiveSnapshotCount() != 0 {
		t.Errorf("Expected 0 active snapshots after purge, got %d", kvManager.ActiveSnapshotCount())
	}
}
