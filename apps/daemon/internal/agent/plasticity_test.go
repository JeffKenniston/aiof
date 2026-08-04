package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCellFusionTrigger(t *testing.T) {
	engine := NewPlasticityEngine(PlasticityOptions{
		FusionThreshold:    50.0, // 50 msg/min
		StallTimeout:       10 * time.Minute,
		ErrorRateThreshold: 0.5,
		WindowSize:         1 * time.Minute,
	})
	defer engine.Close()

	agentA := &NodeState{
		AgentID:      "agent-code-parser",
		Role:         "Code Parser",
		SystemPrompt: "Parse source code into AST trees.",
		AllowedTools: []string{"ast_parse", "file_read"},
		Status:       "ACTIVE",
	}

	agentB := &NodeState{
		AgentID:      "agent-type-checker",
		Role:         "Type Checker",
		SystemPrompt: "Verify struct and interface type links.",
		AllowedTools: []string{"type_check", "symbol_lookup"},
		Status:       "ACTIVE",
	}

	if err := engine.RegisterNode(agentA); err != nil {
		t.Fatalf("Failed to register agentA: %v", err)
	}
	if err := engine.RegisterNode(agentB); err != nil {
		t.Fatalf("Failed to register agentB: %v", err)
	}
	if err := engine.AddEdge("agent-code-parser", "agent-type-checker", 1.0); err != nil {
		t.Fatalf("Failed to add edge: %v", err)
	}

	// Record 55 messages in window -> triggers Cell Fusion (>50 msg/min)
	for i := 0; i < 55; i++ {
		engine.RecordMessage("agent-code-parser", "agent-type-checker", 10*time.Millisecond, false, 150)
	}

	ctx := context.Background()
	actions, err := engine.EvaluatePlasticity(ctx)
	if err != nil {
		t.Fatalf("Plasticity evaluation failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("Expected 1 plasticity action (Cell Fusion), got %d", len(actions))
	}

	if !strings.Contains(actions[0], "CELL_FUSION") {
		t.Errorf("Expected CELL_FUSION action string, got: %s", actions[0])
	}

	// Verify fused node state
	nodeA, _ := engine.GetNode("agent-code-parser")
	if nodeA.Status != "FUSED" {
		t.Errorf("Expected agentA status FUSED, got %s", nodeA.Status)
	}

	fusedID := "fused-agent-code-parser-agent-type-checker"
	fusedNode, err := engine.GetNode(fusedID)
	if err != nil {
		t.Fatalf("Failed to find fused node %s: %v", fusedID, err)
	}

	if !fusedNode.IsFused {
		t.Errorf("Expected fusedNode.IsFused == true")
	}

	if !strings.Contains(fusedNode.SystemPrompt, "Parse source code into AST trees.") ||
		!strings.Contains(fusedNode.SystemPrompt, "Verify struct and interface type links.") {
		t.Errorf("Fused prompt missing prompt components: %s", fusedNode.SystemPrompt)
	}

	// Check merged tools
	toolMap := make(map[string]bool)
	for _, tool := range fusedNode.AllowedTools {
		toolMap[tool] = true
	}
	for _, expected := range []string{"ast_parse", "file_read", "type_check", "symbol_lookup"} {
		if !toolMap[expected] {
			t.Errorf("Fused node missing expected tool %s", expected)
		}
	}
}

func TestCellFissionAndSelfHealing(t *testing.T) {
	engine := NewPlasticityEngine(PlasticityOptions{
		FusionThreshold:    100.0,
		StallTimeout:       100 * time.Millisecond, // Fast stall timeout for test
		ErrorRateThreshold: 0.30,
		WindowSize:         1 * time.Minute,
	})
	defer engine.Close()

	parent := &NodeState{
		AgentID:      "parent-orchestrator",
		Role:         "Orchestrator",
		SystemPrompt: "Orchestrate pipeline.",
		Status:       "ACTIVE",
	}

	worker := &NodeState{
		AgentID:      "worker-agent-1",
		Role:         "Refactor Worker",
		SystemPrompt: "Perform surgical code refactoring.",
		AllowedTools: []string{"file_edit", "git_commit"},
		Status:       "ACTIVE",
		LastActive:   time.Now().UTC().Add(-500 * time.Millisecond), // Stalled (>100ms)
	}

	if err := engine.RegisterNode(parent); err != nil {
		t.Fatalf("Failed to register parent: %v", err)
	}
	if err := engine.RegisterNode(worker); err != nil {
		t.Fatalf("Failed to register worker: %v", err)
	}
	if err := engine.AddEdge("parent-orchestrator", "worker-agent-1", 1.0); err != nil {
		t.Fatalf("Failed to add edge: %v", err)
	}

	ctx := context.Background()
	actions, err := engine.EvaluatePlasticity(ctx)
	if err != nil {
		t.Fatalf("Plasticity evaluation failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("Expected 1 plasticity action (Cell Fission), got %d", len(actions))
	}

	if !strings.Contains(actions[0], "CELL_FISSION") {
		t.Errorf("Expected CELL_FISSION action string, got: %s", actions[0])
	}

	// Verify failed worker node status
	oldWorker, _ := engine.GetNode("worker-agent-1")
	if oldWorker.Status != "FAILED" {
		t.Errorf("Expected worker-agent-1 status FAILED, got %s", oldWorker.Status)
	}

	// Verify DAG edge re-wiring
	activeEdges := engine.GetActiveEdges()
	if len(activeEdges) != 1 {
		t.Fatalf("Expected 1 active edge in DAG after re-wiring, got %d", len(activeEdges))
	}

	edge := activeEdges[0]
	if edge.FromAgentID != "parent-orchestrator" {
		t.Errorf("Expected edge from parent-orchestrator, got %s", edge.FromAgentID)
	}
	if !strings.HasPrefix(edge.ToAgentID, "replacement-worker-agent-1-") {
		t.Errorf("Expected edge to replacement node, got %s", edge.ToAgentID)
	}
}

func TestTelemetryObserverSlidingWindow(t *testing.T) {
	observer := NewTelemetryObserver(100 * time.Millisecond)

	now := time.Now().UTC()
	observer.RecordSample(MetricSample{
		SourceAgentID: "node-A",
		TargetAgentID: "node-B",
		Latency:       20 * time.Millisecond,
		IsError:       false,
		TokenCount:    100,
		Timestamp:     now.Add(-200 * time.Millisecond), // Expired
	})

	observer.RecordSample(MetricSample{
		SourceAgentID: "node-A",
		TargetAgentID: "node-B",
		Latency:       50 * time.Millisecond,
		IsError:       false,
		TokenCount:    200,
		Timestamp:     now, // Valid
	})

	telemetry := observer.GetInterNodeTelemetry("node-A", "node-B")
	if telemetry.TotalMessages != 1 {
		t.Errorf("Expected 1 active message in window, got %d", telemetry.TotalMessages)
	}
	if telemetry.TotalTokens != 200 {
		t.Errorf("Expected 200 total tokens, got %d", telemetry.TotalTokens)
	}
}

func TestPlasticityMultiTenantOPAGating(t *testing.T) {
	engine := NewPlasticityEngine(PlasticityOptions{
		FusionThreshold: 10.0,
	})
	defer engine.Close()

	_ = engine.RegisterNode(&NodeState{
		AgentID:      "agent-eng",
		Role:         "Engineer",
		SystemPrompt: "Write code",
		TenantID:     "TENANT-ENG",
	})
	_ = engine.RegisterNode(&NodeState{
		AgentID:      "agent-fin",
		Role:         "Accountant",
		SystemPrompt: "Manage finances",
		TenantID:     "TENANT-FINANCE",
	})

	_ = engine.AddEdge("agent-eng", "agent-fin", 1.0)

	// Simulate high traffic > threshold
	for i := 0; i < 20; i++ {
		engine.RecordMessage("agent-eng", "agent-fin", 10*time.Millisecond, false, 100)
	}

	actions, err := engine.EvaluatePlasticity(context.Background())
	if err != nil {
		t.Fatalf("EvaluatePlasticity unexpected error: %v", err)
	}
	for _, a := range actions {
		if strings.Contains(a, "CELL_FUSION") {
			t.Fatalf("Expected cross-tenant fusion to be rejected by OPA tenant gating, but got fusion action: %s", a)
		}
	}
}

