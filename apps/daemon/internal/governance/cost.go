package governance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Common Governance Errors
var (
	ErrHardBudgetExceeded = errors.New("governance: hard budget limit exceeded for organization/project")
	ErrSoftBudgetWarning  = errors.New("governance: soft budget limit warning threshold reached")
	ErrInvalidModel       = errors.New("governance: unsupported or unconfigured LLM provider model")
	ErrTamperedLedger     = errors.New("governance: cryptographic audit ledger hash chain tampering detected")
	ErrInvalidBudget      = errors.New("governance: budget limits must be positive numbers")
)

// TaskComplexity classifies incoming workload requirements per ADR-25.
type TaskComplexity int

const (
	ComplexityRoutine TaskComplexity = iota
	ComplexitySummarization
	ComplexityCodeSynthesis
	ComplexityComplexReasoning
	ComplexityLocalOnly
)

func (c TaskComplexity) String() string {
	switch c {
	case ComplexityRoutine:
		return "ROUTINE"
	case ComplexitySummarization:
		return "SUMMARIZATION"
	case ComplexityCodeSynthesis:
		return "CODE_SYNTHESIS"
	case ComplexityComplexReasoning:
		return "COMPLEX_REASONING"
	case ComplexityLocalOnly:
		return "LOCAL_ONLY"
	default:
		return "UNKNOWN"
	}
}

// ModelCostRate defines per-million token pricing rates for LLM providers.
type ModelCostRate struct {
	ModelName            string  `json:"model_name"`
	Provider             string  `json:"provider"`              // "gemini", "claude", "openvino", "ollama"
	InputUSDPerM         float64 `json:"input_usd_per_m"`      // USD per 1M input tokens
	OutputUSDPerM        float64 `json:"output_usd_per_m"`     // USD per 1M output tokens
	IsLocal              bool    `json:"is_local"`
	SupportsContextCache bool    `json:"supports_context_cache"`
}

// DefaultModelRates provides baseline rate cards for framework models per ADR-25.
var DefaultModelRates = map[string]ModelCostRate{
	"gemini-1.5-pro": {
		ModelName:            "gemini-1.5-pro",
		Provider:             "gemini",
		InputUSDPerM:         3.50,
		OutputUSDPerM:        10.50,
		IsLocal:              false,
		SupportsContextCache: true,
	},
	"gemini-1.5-flash": {
		ModelName:            "gemini-1.5-flash",
		Provider:             "gemini",
		InputUSDPerM:         0.35,
		OutputUSDPerM:        1.05,
		IsLocal:              false,
		SupportsContextCache: true,
	},
	"claude-3-5-sonnet": {
		ModelName:            "claude-3-5-sonnet",
		Provider:             "claude",
		InputUSDPerM:         3.00,
		OutputUSDPerM:        15.00,
		IsLocal:              false,
		SupportsContextCache: false,
	},
	"openvino-qwen2.5-coder-7b": {
		ModelName:            "openvino-qwen2.5-coder-7b",
		Provider:             "openvino",
		InputUSDPerM:         0.0,
		OutputUSDPerM:        0.0,
		IsLocal:              true,
		SupportsContextCache: true,
	},
	"ollama-llama3.1-8b": {
		ModelName:            "ollama-llama3.1-8b",
		Provider:             "ollama",
		InputUSDPerM:         0.0,
		OutputUSDPerM:        0.0,
		IsLocal:              true,
		SupportsContextCache: false,
	},
}

// BudgetCap defines soft and hard financial caps per scope.
type BudgetCap struct {
	ScopeID            string    `json:"scope_id"`             // OrgID or ProjectID
	SoftLimitUSD       float64   `json:"soft_limit_usd"`       // Warning threshold (e.g., $80.00)
	HardLimitUSD       float64   `json:"hard_limit_usd"`       // Hard stop threshold (e.g., $100.00)
	CurrentSpendUSD    float64   `json:"current_spend_usd"`
	TotalInputTokens   uint64    `json:"total_input_tokens"`
	TotalOutputTokens  uint64    `json:"total_output_tokens"`
	ForceLocalFallback bool      `json:"force_local_fallback"` // Auto-downgrade to OpenVINO on budget breach
	LastUpdated        time.Time `json:"last_updated"`
}

// TransactionEntry records an immutable transaction in the cryptographic audit ledger.
type TransactionEntry struct {
	Index           uint64    `json:"index"`
	TransactionID   string    `json:"transaction_id"`
	Timestamp       time.Time `json:"timestamp"`
	OrgID           string    `json:"org_id"`
	ProjectID       string    `json:"project_id"`
	UserID          string    `json:"user_id"`
	ModelName       string    `json:"model_name"`
	InputTokens     uint64    `json:"input_tokens"`
	OutputTokens    uint64    `json:"output_tokens"`
	CostUSD         float64   `json:"cost_usd"`
	TaskComplexity  string    `json:"task_complexity"`
	FallbackApplied bool      `json:"fallback_applied"`
	PrevHash        string    `json:"prev_hash"`
	Hash            string    `json:"hash"`
}

// CalculateHash computes the SHA-256 digest of a transaction chained to the previous hash.
func (t *TransactionEntry) CalculateHash() string {
	record := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%d|%d|%.6f|%s|%t|%s",
		t.Index, t.TransactionID, t.Timestamp.Format(time.RFC3339Nano),
		t.OrgID, t.ProjectID, t.UserID, t.ModelName,
		t.InputTokens, t.OutputTokens, t.CostUSD,
		t.TaskComplexity, t.FallbackApplied, t.PrevHash,
	)
	h := sha256.Sum256([]byte(record))
	return hex.EncodeToString(h[:])
}

// ModelTierPolicyRouter selects the optimal model tier based on task complexity and budget limits.
type ModelTierPolicyRouter struct {
	mu           sync.RWMutex
	rateCards    map[string]ModelCostRate
	defaultLocal string
	defaultCloud string
}

func NewModelTierPolicyRouter(defaultCloud, defaultLocal string) *ModelTierPolicyRouter {
	if defaultCloud == "" {
		defaultCloud = "gemini-1.5-pro"
	}
	if defaultLocal == "" {
		defaultLocal = "openvino-qwen2.5-coder-7b"
	}

	rates := make(map[string]ModelCostRate)
	for k, v := range DefaultModelRates {
		rates[k] = v
	}

	return &ModelTierPolicyRouter{
		rateCards:    rates,
		defaultLocal: defaultLocal,
		defaultCloud: defaultCloud,
	}
}

// RouteTarget specifies the selected execution target model.
type RouteTarget struct {
	SelectedModel   string `json:"selected_model"`
	Provider        string `json:"provider"`
	IsLocal         bool   `json:"is_local"`
	FallbackApplied bool   `json:"fallback_applied"`
	Reason          string `json:"reason"`
}

// SelectModel determines the optimal model based on complexity and current budget standing.
func (r *ModelTierPolicyRouter) SelectModel(complexity TaskComplexity, budget *BudgetCap) RouteTarget {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Check if budget hard cap is breached
	if budget != nil && budget.CurrentSpendUSD >= budget.HardLimitUSD {
		if budget.ForceLocalFallback {
			return RouteTarget{
				SelectedModel:   r.defaultLocal,
				Provider:        r.rateCards[r.defaultLocal].Provider,
				IsLocal:         true,
				FallbackApplied: true,
				Reason:          "hard budget cap breached - forced fallback to local OpenVINO model",
			}
		}
	}

	// 2. Check if budget soft limit is breached -> auto-downgrade non-critical tasks
	softBreached := budget != nil && budget.CurrentSpendUSD >= budget.SoftLimitUSD

	switch complexity {
	case ComplexityLocalOnly:
		return RouteTarget{
			SelectedModel:   r.defaultLocal,
			Provider:        r.rateCards[r.defaultLocal].Provider,
			IsLocal:         true,
			FallbackApplied: false,
			Reason:          "task explicitly requested local model execution",
		}

	case ComplexityRoutine, ComplexitySummarization:
		return RouteTarget{
			SelectedModel:   r.defaultLocal,
			Provider:        r.rateCards[r.defaultLocal].Provider,
			IsLocal:         true,
			FallbackApplied: false,
			Reason:          "routine/summarization task routed to local zero-cost model",
		}

	case ComplexityCodeSynthesis, ComplexityComplexReasoning:
		if softBreached && budget.ForceLocalFallback {
			return RouteTarget{
				SelectedModel:   r.defaultLocal,
				Provider:        r.rateCards[r.defaultLocal].Provider,
				IsLocal:         true,
				FallbackApplied: true,
				Reason:          "soft budget limit breached - degrading reasoning tier to local model",
			}
		}

		return RouteTarget{
			SelectedModel:   r.defaultCloud,
			Provider:        r.rateCards[r.defaultCloud].Provider,
			IsLocal:         false,
			FallbackApplied: false,
			Reason:          "complex reasoning task routed to primary cloud frontier model",
		}

	default:
		return RouteTarget{
			SelectedModel:   r.defaultLocal,
			Provider:        r.rateCards[r.defaultLocal].Provider,
			IsLocal:         true,
			FallbackApplied: false,
			Reason:          "default fallback route to local model",
		}
	}
}

// CostGovernanceEngine manages financial budgets, token spend, and cryptographic audit ledgers per ADR-25.
type CostGovernanceEngine struct {
	mu           sync.RWMutex
	budgets      map[string]*BudgetCap // Key: scope (org_id or project_id)
	router       *ModelTierPolicyRouter
	auditLedger  []TransactionEntry
	lastHash     string
	totalSpend   float64
	totalTxCount uint64
}

func NewCostGovernanceEngine(router *ModelTierPolicyRouter) *CostGovernanceEngine {
	if router == nil {
		router = NewModelTierPolicyRouter("gemini-1.5-pro", "openvino-qwen2.5-coder-7b")
	}

	genesisEntry := TransactionEntry{
		Index:          0,
		TransactionID:  "tx-genesis",
		Timestamp:      time.Now().UTC(),
		OrgID:          "system",
		ProjectID:      "root",
		UserID:         "system",
		ModelName:      "genesis",
		InputTokens:    0,
		OutputTokens:   0,
		CostUSD:        0.0,
		TaskComplexity: "GENESIS",
		PrevHash:       "0000000000000000000000000000000000000000000000000000000000000000",
	}
	genesisEntry.Hash = genesisEntry.CalculateHash()

	return &CostGovernanceEngine{
		budgets:     make(map[string]*BudgetCap),
		router:      router,
		auditLedger: []TransactionEntry{genesisEntry},
		lastHash:    genesisEntry.Hash,
	}
}

// SetBudgetCap provisions or updates a budget cap for an org or project scope.
func (e *CostGovernanceEngine) SetBudgetCap(scopeID string, softLimitUSD, hardLimitUSD float64, forceLocalFallback bool) error {
	if softLimitUSD < 0 || hardLimitUSD <= 0 || softLimitUSD > hardLimitUSD {
		return ErrInvalidBudget
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	b, exists := e.budgets[scopeID]
	if !exists {
		b = &BudgetCap{
			ScopeID: scopeID,
		}
		e.budgets[scopeID] = b
	}

	b.SoftLimitUSD = softLimitUSD
	b.HardLimitUSD = hardLimitUSD
	b.ForceLocalFallback = forceLocalFallback
	b.LastUpdated = time.Now().UTC()

	return nil
}

// GetBudgetCap retrieves the current budget standing for a given scope.
func (e *CostGovernanceEngine) GetBudgetCap(scopeID string) (*BudgetCap, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	b, exists := e.budgets[scopeID]
	if !exists {
		return nil, false
	}

	// Return a copy
	cp := *b
	return &cp, true
}

// CalculateCost computes the dollar cost for a given model and token counts.
func (e *CostGovernanceEngine) CalculateCost(modelName string, inputTokens, outputTokens uint64) (float64, error) {
	rate, exists := DefaultModelRates[modelName]
	if !exists {
		// Default to zero if local/unknown
		if strings.HasPrefix(modelName, "local") || strings.HasPrefix(modelName, "openvino") || strings.HasPrefix(modelName, "ollama") {
			return 0.0, nil
		}
		return 0.0, ErrInvalidModel
	}

	inputCost := (float64(inputTokens) / 1000000.0) * rate.InputUSDPerM
	outputCost := (float64(outputTokens) / 1000000.0) * rate.OutputUSDPerM
	return inputCost + outputCost, nil
}

// PreCheckBudget evaluates budget availability before model invocation.
func (e *CostGovernanceEngine) PreCheckBudget(scopeID string, complexity TaskComplexity) (RouteTarget, error) {
	e.mu.RLock()
	budget := e.budgets[scopeID]
	e.mu.RUnlock()

	target := e.router.SelectModel(complexity, budget)

	if budget != nil && budget.CurrentSpendUSD >= budget.HardLimitUSD && !budget.ForceLocalFallback {
		return target, fmt.Errorf("%w: current spend $%.2f >= hard cap $%.2f for scope %s", ErrHardBudgetExceeded, budget.CurrentSpendUSD, budget.HardLimitUSD, scopeID)
	}

	return target, nil
}

// RecordTransaction records token spend, updates budgets, and appends a cryptographic entry to the audit ledger.
func (e *CostGovernanceEngine) RecordTransaction(
	txID, orgID, projectID, userID, modelName string,
	inputTokens, outputTokens uint64,
	complexity TaskComplexity,
	fallbackApplied bool,
) (*TransactionEntry, error) {
	costUSD, err := e.CalculateCost(modelName, inputTokens, outputTokens)
	if err != nil {
		costUSD = 0.0
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Update Org & Project Budgets
	scopes := []string{orgID, projectID}
	for _, scope := range scopes {
		if scope == "" {
			continue
		}
		b, exists := e.budgets[scope]
		if !exists {
			b = &BudgetCap{
				ScopeID:            scope,
				SoftLimitUSD:       1000.0, // Default soft limit
				HardLimitUSD:       1250.0, // Default hard limit
				ForceLocalFallback: true,
			}
			e.budgets[scope] = b
		}

		b.CurrentSpendUSD += costUSD
		b.TotalInputTokens += inputTokens
		b.TotalOutputTokens += outputTokens
		b.LastUpdated = time.Now().UTC()
	}

	// Construct Cryptographic Ledger Entry
	idx := uint64(len(e.auditLedger))
	entry := TransactionEntry{
		Index:           idx,
		TransactionID:   txID,
		Timestamp:       time.Now().UTC(),
		OrgID:           orgID,
		ProjectID:       projectID,
		UserID:          userID,
		ModelName:       modelName,
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		CostUSD:         costUSD,
		TaskComplexity:  complexity.String(),
		FallbackApplied: fallbackApplied,
		PrevHash:        e.lastHash,
	}
	entry.Hash = entry.CalculateHash()

	e.auditLedger = append(e.auditLedger, entry)
	e.lastHash = entry.Hash
	e.totalSpend += costUSD
	e.totalTxCount++

	return &entry, nil
}

// VerifyLedgerIntegrity audits the cryptographic hash chain of the transaction ledger.
func (e *CostGovernanceEngine) VerifyLedgerIntegrity() (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.auditLedger) == 0 {
		return true, nil
	}

	for i := 0; i < len(e.auditLedger); i++ {
		entry := e.auditLedger[i]

		// Check self-hash
		computedHash := entry.CalculateHash()
		if entry.Hash != computedHash {
			return false, fmt.Errorf("%w: hash mismatch at index %d (stored %s != computed %s)", ErrTamperedLedger, i, entry.Hash, computedHash)
		}

		// Check prev-hash chain
		if i > 0 {
			prevEntry := e.auditLedger[i-1]
			if entry.PrevHash != prevEntry.Hash {
				return false, fmt.Errorf("%w: chain broken at index %d (prev_hash %s != prev.hash %s)", ErrTamperedLedger, i, entry.PrevHash, prevEntry.Hash)
			}
		}
	}

	return true, nil
}

// GetLedger returns a copy of the transaction audit ledger.
func (e *CostGovernanceEngine) GetLedger() []TransactionEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()

	res := make([]TransactionEntry, len(e.auditLedger))
	copy(res, e.auditLedger)
	return res
}

// GetSummary returns real-time total spend and transaction count metrics.
func (e *CostGovernanceEngine) GetSummary() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]interface{}{
		"total_spend_usd":    e.totalSpend,
		"total_transactions": e.totalTxCount,
		"ledger_size":        len(e.auditLedger),
		"active_budgets":     len(e.budgets),
		"latest_hash":        e.lastHash,
	}
}
