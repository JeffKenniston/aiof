package consensus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// ADR-09 & ADR-14: Weighted Byzantine Fault-Tolerant (BFT) Multi-Agent Consensus
// ============================================================================

var (
	ErrQuorumNotReached      = errors.New("bft: weighted quorum (> 66.7%) not reached")
	ErrProposalAlreadyExists = errors.New("bft: proposal with this ID already registered")
	ErrProposalNotFound      = errors.New("bft: proposal not found")
	ErrDuplicateVote           = errors.New("bft: agent has already voted on this proposal")
	ErrInvalidVoteWeight       = errors.New("bft: agent voting weight is zero or negative")
	ErrConsensusClosed         = errors.New("bft: consensus round is closed")
	ErrPassKVerificationFailed = errors.New("bft: proposal failed pass^k multi-attempt verification threshold")
)

// DecisionStatus represents the final outcome of a consensus round.
type DecisionStatus string

const (
	StatusPending  DecisionStatus = "PENDING"
	StatusApproved DecisionStatus = "APPROVED"
	StatusRejected DecisionStatus = "REJECTED"
	StatusTieBreak DecisionStatus = "TIE_BREAK_REQUIRED"
)

// AgentReputation tracks historical performance metrics per ADR-14.
type AgentReputation struct {
	AgentID        string    `json:"agent_id"`
	AccuracyScore  float64   `json:"accuracy_score"`  // 0.0 to 1.0
	TotalProposals uint64    `json:"total_proposals"`
	ValidCommits   uint64    `json:"valid_commits"`
	SlashedCount   uint64    `json:"slashed_count"`
	TotalPenalties float64   `json:"total_penalties"`
	LastUpdated    time.Time `json:"last_updated"`
}

// Proposal represents a code refactoring or execution plan submitted for consensus.
type Proposal struct {
	ID                 string          `json:"id"`
	ProposerID         string          `json:"proposer_id"`
	Title              string          `json:"title"`
	CodeDiff           string          `json:"code_diff"`
	ASTPassRate        float64         `json:"ast_pass_rate"`        // 0.0 to 1.0 (Linter / AST static analysis)
	UnitTestPassRate   float64         `json:"unit_test_pass_rate"`  // 0.0 to 1.0
	SMTProofScore      float64         `json:"smt_proof_score"`      // 0.0 to 1.0 (Formal SMT gate)
	PassKScore         float64         `json:"pass_k_score"`         // 0.0 to 1.0 ($pass^k$ statistical execution sampling)
	PassKTrials        int             `json:"pass_k_trials"`        // Number of k independent sandbox executions
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	SubmittedAt        time.Time       `json:"submitted_at"`
	PayloadHash        string          `json:"payload_hash"`
}

// Vote represents a signed vote from a sub-agent.
type Vote struct {
	AgentID     string    `json:"agent_id"`
	ProposalID  string    `json:"proposal_id"`
	Approve     bool      `json:"approve"`
	Confidence  float64   `json:"confidence"` // 0.0 to 1.0
	Reason      string    `json:"reason"`
	Signature   []byte    `json:"signature,omitempty"`
	VotedAt     time.Time `json:"voted_at"`
}

// VoteWeightBreakdown details the mathematical factors constituting an agent's voting power.
type VoteWeightBreakdown struct {
	AgentID          string  `json:"agent_id"`
	HistoricalWeight float64 `json:"historical_weight"` // W_history * Reputation
	ASTWeight        float64 `json:"ast_weight"`        // W_ast * ASTPassRate
	TestWeight       float64 `json:"test_weight"`       // W_test * UnitTestPassRate
	SMTWeight        float64 `json:"smt_weight"`        // W_smt * SMTProofScore
	TotalWeight      float64 `json:"total_weight"`
}

// ConsensusResult represents the final audited decision of a BFT voting round.
type ConsensusResult struct {
	ProposalID       string                `json:"proposal_id"`
	Status           DecisionStatus        `json:"status"`
	TotalWeight      float64               `json:"total_weight"`
	ApproveWeight    float64               `json:"approve_weight"`
	RejectWeight     float64               `json:"reject_weight"`
	ApproveRatio     float64               `json:"approve_ratio"` // ApproveWeight / TotalWeight
	QuorumThreshold  float64               `json:"quorum_threshold"` // Default 0.667 (66.7%)
	QuorumReached    bool                  `json:"quorum_reached"`
	TieBreakerUsed   bool                  `json:"tie_breaker_used"`
	TieBreakerAgent  string                `json:"tie_breaker_agent,omitempty"`
	VoteBreakdowns   []VoteWeightBreakdown `json:"vote_breakdowns"`
	EvaluatedAt      time.Time             `json:"evaluated_at"`
}

// BFTConfig configures the BFT consensus engine weights and threshold.
type BFTConfig struct {
	QuorumThreshold   float64 `json:"quorum_threshold"`    // e.g. 0.667 (66.7%)
	MinPassKThreshold float64 `json:"min_pass_k_threshold"` // e.g. 0.80 ($pass^k$ statistical consistency)
	PassKTrials       int     `json:"pass_k_trials"`       // e.g. 3 independent sandbox executions
	WeightHistory     float64 `json:"weight_history"`      // Default 0.40
	WeightAST         float64 `json:"weight_ast"`          // Default 0.20
	WeightUnitTest    float64 `json:"weight_unit_test"`    // Default 0.20
	WeightSMT         float64 `json:"weight_smt"`          // Default 0.20
}

func DefaultBFTConfig() BFTConfig {
	return BFTConfig{
		QuorumThreshold:   0.667,
		MinPassKThreshold: 0.80,
		PassKTrials:       3,
		WeightHistory:     0.40,
		WeightAST:         0.20,
		WeightUnitTest:    0.20,
		WeightSMT:         0.20,
	}
}

// BFTConsensusEngine coordinates multi-agent voting rounds with BFT quorum evaluation.
type BFTConsensusEngine struct {
	mu          sync.RWMutex
	cfg         BFTConfig
	proposals   map[string]*Proposal
	votes       map[string]map[string]*Vote // proposalID -> agentID -> Vote
	reputations map[string]*AgentReputation
}

func NewBFTConsensusEngine(cfg BFTConfig) *BFTConsensusEngine {
	if cfg.QuorumThreshold <= 0 || cfg.QuorumThreshold > 1.0 {
		cfg.QuorumThreshold = 0.667
	}
	return &BFTConsensusEngine{
		cfg:         cfg,
		proposals:   make(map[string]*Proposal),
		votes:       make(map[string]map[string]*Vote),
		reputations: make(map[string]*AgentReputation),
	}
}

// RegisterAgentReputation sets or updates an agent's historical accuracy score.
func (e *BFTConsensusEngine) RegisterAgentReputation(rep AgentReputation) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rep.AccuracyScore < 0.0 {
		rep.AccuracyScore = 0.0
	} else if rep.AccuracyScore > 1.0 {
		rep.AccuracyScore = 1.0
	}
	rep.LastUpdated = time.Now().UTC()
	e.reputations[rep.AgentID] = &rep
}

func (e *BFTConsensusEngine) GetAgentReputation(agentID string) AgentReputation {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if rep, exists := e.reputations[agentID]; exists {
		return *rep
	}
	// Default baseline reputation for new agents
	return AgentReputation{
		AgentID:       agentID,
		AccuracyScore: 0.75,
		LastUpdated:   time.Now().UTC(),
	}
}

// SubmitProposal registers a new proposal for multi-agent BFT voting.
func (e *BFTConsensusEngine) SubmitProposal(p Proposal) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if p.ID == "" {
		return errors.New("bft: proposal ID cannot be empty")
	}
	if _, exists := e.proposals[p.ID]; exists {
		return ErrProposalAlreadyExists
	}

	if p.SubmittedAt.IsZero() {
		p.SubmittedAt = time.Now().UTC()
	}

	// Compute payload hash for cryptographic integrity
	h := sha256.New()
	h.Write([]byte(p.ID + p.ProposerID + p.CodeDiff))
	p.PayloadHash = hex.EncodeToString(h.Sum(nil))

	e.proposals[p.ID] = &p
	e.votes[p.ID] = make(map[string]*Vote)
	return nil
}

// CastVote records a signed vote from a participating sub-agent.
func (e *BFTConsensusEngine) CastVote(v Vote) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.proposals[v.ProposalID]; !exists {
		return ErrProposalNotFound
	}

	if v.VotedAt.IsZero() {
		v.VotedAt = time.Now().UTC()
	}

	if _, voted := e.votes[v.ProposalID][v.AgentID]; voted {
		return ErrDuplicateVote
	}

	e.votes[v.ProposalID][v.AgentID] = &v
	return nil
}

// CalculateVoteWeight computes the weighted voting power of an agent for a specific proposal.
//
// Formula (ADR-09):
// W_agent = W_history * Reputation + W_ast * ASTPass + W_test * UnitTestPass + W_smt * SMTScore
func (e *BFTConsensusEngine) CalculateVoteWeight(agentID string, p *Proposal) VoteWeightBreakdown {
	rep := e.GetAgentReputation(agentID)

	histW := e.cfg.WeightHistory * rep.AccuracyScore
	astW := e.cfg.WeightAST * p.ASTPassRate
	testW := e.cfg.WeightUnitTest * p.UnitTestPassRate
	smtW := e.cfg.WeightSMT * p.SMTProofScore

	total := histW + astW + testW + smtW

	return VoteWeightBreakdown{
		AgentID:          agentID,
		HistoricalWeight: math.Round(histW*1000) / 1000,
		ASTWeight:        math.Round(astW*1000) / 1000,
		TestWeight:       math.Round(testW*1000) / 1000,
		SMTWeight:        math.Round(smtW*1000) / 1000,
		TotalWeight:      math.Round(total*1000) / 1000,
	}
}

// EvaluateConsensus computes the weighted BFT quorum decision for a proposal.
func (e *BFTConsensusEngine) EvaluateConsensus(ctx context.Context, proposalID string) (*ConsensusResult, error) {
	e.mu.RLock()
	p, exists := e.proposals[proposalID]
	if !exists {
		e.mu.RUnlock()
		return nil, ErrProposalNotFound
	}

	agentVotes := make([]*Vote, 0, len(e.votes[proposalID]))
	for _, v := range e.votes[proposalID] {
		agentVotes = append(agentVotes, v)
	}
	e.mu.RUnlock()

	var totalWeight float64
	var approveWeight float64
	var rejectWeight float64
	breakdowns := make([]VoteWeightBreakdown, 0, len(agentVotes))

	for _, v := range agentVotes {
		wb := e.CalculateVoteWeight(v.AgentID, p)
		// Scale weight by agent confidence
		effectiveWeight := wb.TotalWeight * math.Max(0.1, v.Confidence)

		totalWeight += effectiveWeight
		if v.Approve {
			approveWeight += effectiveWeight
		} else {
			rejectWeight += effectiveWeight
		}
		breakdowns = append(breakdowns, wb)
	}

	sort.Slice(breakdowns, func(i, j int) bool {
		return breakdowns[i].TotalWeight > breakdowns[j].TotalWeight
	})

	res := &ConsensusResult{
		ProposalID:      proposalID,
		Status:          StatusPending,
		TotalWeight:     math.Round(totalWeight*1000) / 1000,
		ApproveWeight:   math.Round(approveWeight*1000) / 1000,
		RejectWeight:    math.Round(rejectWeight*1000) / 1000,
		QuorumThreshold: e.cfg.QuorumThreshold,
		VoteBreakdowns:  breakdowns,
		EvaluatedAt:     time.Now().UTC(),
	}

	if totalWeight > 0 {
		res.ApproveRatio = math.Round((approveWeight/totalWeight)*1000) / 1000
	}

	// 50/50 Equal Split check -> requires VRF tie-breaker
	if totalWeight > 0 && math.Abs(approveWeight-rejectWeight) < 0.001 {
		res.Status = StatusTieBreak
		return res, nil
	}

	if res.ApproveRatio >= e.cfg.QuorumThreshold {
		res.QuorumReached = true
		res.Status = StatusApproved
		if p, ok := e.proposals[proposalID]; ok && p.PassKTrials > 0 && p.PassKScore < e.cfg.MinPassKThreshold {
			res.Status = StatusRejected
			res.QuorumReached = false
		}
	} else {
		res.QuorumReached = false
		res.Status = StatusRejected
	}

	return res, nil
}

// CalculatePassK computes the unbiased pass^k estimator: 1.0 - C(n-c, k) / C(n, k).
func CalculatePassK(n, c, k int) float64 {
	if n <= 0 || k <= 0 || k > n {
		return 0.0
	}
	if c == n {
		return 1.0
	}
	if c < k {
		return 0.0
	}
	num := 1.0
	den := 1.0
	for i := 0; i < k; i++ {
		num *= float64(n - c - i)
		den *= float64(n - i)
	}
	if den == 0.0 {
		return 0.0
	}
	return 1.0 - (num / den)
}

// EvaluatePassK runs n concurrent sandbox verification trials and computes the statistical pass^k score.
func (e *BFTConsensusEngine) EvaluatePassK(ctx context.Context, proposal *Proposal, k int, targetThreshold float64, trialFunc func(ctx context.Context, trialIdx int) (bool, error)) (float64, error) {
	if k <= 0 {
		k = e.cfg.PassKTrials
		if k <= 0 {
			k = 3
		}
	}
	if targetThreshold <= 0 {
		targetThreshold = e.cfg.MinPassKThreshold
		if targetThreshold <= 0 {
			targetThreshold = 0.80
		}
	}

	n := k
	var c int32
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			success, err := trialFunc(ctx, idx)
			if err == nil && success {
				atomic.AddInt32(&c, 1)
			}
		}(i)
	}
	wg.Wait()

	score := CalculatePassK(n, int(c), k)
	proposal.PassKScore = score
	proposal.PassKTrials = k

	if score < targetThreshold {
		return score, fmt.Errorf("%w: pass^k=%.2f < threshold=%.2f (%d/%d passed)", ErrPassKVerificationFailed, score, targetThreshold, c, n)
	}
	return score, nil
}

