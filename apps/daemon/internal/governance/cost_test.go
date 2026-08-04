package governance

import (
	"strings"
	"testing"
)

func TestCalculateCost(t *testing.T) {
	router := NewModelTierPolicyRouter("gemini-1.5-pro", "openvino-qwen2.5-coder-7b")
	engine := NewCostGovernanceEngine(router)

	// Gemini 1.5 Pro: 1,000,000 input = $3.50, 1,000,000 output = $10.50 -> total $14.00
	cost, err := engine.CalculateCost("gemini-1.5-pro", 1000000, 1000000)
	if err != nil {
		t.Fatalf("unexpected error calculating cost: %v", err)
	}

	expected := 14.00
	if cost != expected {
		t.Errorf("expected cost $%.2f, got $%.2f", expected, cost)
	}

	// Local model should cost $0.00
	localCost, err := engine.CalculateCost("openvino-qwen2.5-coder-7b", 5000000, 5000000)
	if err != nil {
		t.Fatalf("unexpected error for local model: %v", err)
	}
	if localCost != 0.0 {
		t.Errorf("expected local model cost $0.00, got $%.2f", localCost)
	}
}

func TestModelTierPolicyRouter(t *testing.T) {
	router := NewModelTierPolicyRouter("gemini-1.5-pro", "openvino-qwen2.5-coder-7b")

	budget := &BudgetCap{
		ScopeID:            "org-1",
		SoftLimitUSD:       80.0,
		HardLimitUSD:       100.0,
		CurrentSpendUSD:    10.0,
		ForceLocalFallback: true,
	}

	// Complex reasoning within budget -> frontier cloud model
	target1 := router.SelectModel(ComplexityComplexReasoning, budget)
	if target1.SelectedModel != "gemini-1.5-pro" || target1.IsLocal {
		t.Errorf("expected cloud model gemini-1.5-pro, got %s (local: %v)", target1.SelectedModel, target1.IsLocal)
	}

	// Routine task -> local model
	target2 := router.SelectModel(ComplexityRoutine, budget)
	if target2.SelectedModel != "openvino-qwen2.5-coder-7b" || !target2.IsLocal {
		t.Errorf("expected local model openvino-qwen2.5-coder-7b, got %s", target2.SelectedModel)
	}

	// Soft limit breach -> complex reasoning falls back to local model
	budget.CurrentSpendUSD = 85.0
	target3 := router.SelectModel(ComplexityComplexReasoning, budget)
	if !target3.IsLocal || !target3.FallbackApplied {
		t.Errorf("expected fallback to local model on soft budget breach, got %s", target3.SelectedModel)
	}
}

func TestCostGovernanceEngineLedgerAndBudget(t *testing.T) {
	router := NewModelTierPolicyRouter("gemini-1.5-pro", "openvino-qwen2.5-coder-7b")
	engine := NewCostGovernanceEngine(router)

	err := engine.SetBudgetCap("org-acme", 50.0, 100.0, true)
	if err != nil {
		t.Fatalf("failed to set budget cap: %v", err)
	}

	// Record transaction 1
	tx1, err := engine.RecordTransaction("tx-001", "org-acme", "proj-alpha", "user-alice", "gemini-1.5-pro", 1000000, 1000000, ComplexityComplexReasoning, false)
	if err != nil {
		t.Fatalf("failed to record tx1: %v", err)
	}

	if tx1.CostUSD != 14.00 {
		t.Errorf("expected tx1 cost $14.00, got $%.2f", tx1.CostUSD)
	}

	// Check updated budget
	b, ok := engine.GetBudgetCap("org-acme")
	if !ok {
		t.Fatalf("expected budget cap to exist for org-acme")
	}
	if b.CurrentSpendUSD != 14.00 {
		t.Errorf("expected budget spend $14.00, got $%.2f", b.CurrentSpendUSD)
	}

	// Record transaction 2
	tx2, err := engine.RecordTransaction("tx-002", "org-acme", "proj-alpha", "user-bob", "claude-3-5-sonnet", 2000000, 1000000, ComplexityCodeSynthesis, false)
	if err != nil {
		t.Fatalf("failed to record tx2: %v", err)
	}
	_ = tx2

	// Verify ledger integrity
	valid, err := engine.VerifyLedgerIntegrity()
	if err != nil || !valid {
		t.Fatalf("ledger integrity verification failed: %v", err)
	}

	// Tamper with ledger entry and verify detection
	ledger := engine.GetLedger()
	if len(ledger) < 3 {
		t.Fatalf("expected at least 3 ledger entries (including genesis)")
	}

	// Simulate tampered ledger copy check
	tamperedEntry := ledger[1]
	tamperedEntry.CostUSD = 999.99
	if tamperedEntry.CalculateHash() == tamperedEntry.Hash {
		t.Errorf("expected hash to change when entry is tampered")
	}
}

func TestHardBudgetCapExceeded(t *testing.T) {
	router := NewModelTierPolicyRouter("gemini-1.5-pro", "openvino-qwen2.5-coder-7b")
	engine := NewCostGovernanceEngine(router)

	// Hard budget $10.00 without force local fallback
	err := engine.SetBudgetCap("org-tight", 8.0, 10.0, false)
	if err != nil {
		t.Fatalf("failed to set budget: %v", err)
	}

	// Fill budget to $14.00
	_, _ = engine.RecordTransaction("tx-over", "org-tight", "proj-beta", "user-charlie", "gemini-1.5-pro", 1000000, 1000000, ComplexityComplexReasoning, false)

	// PreCheck should return hard budget exceeded error
	_, err = engine.PreCheckBudget("org-tight", ComplexityComplexReasoning)
	if err == nil || !strings.Contains(err.Error(), "hard budget limit exceeded") {
		t.Errorf("expected hard budget exceeded error, got: %v", err)
	}
}
