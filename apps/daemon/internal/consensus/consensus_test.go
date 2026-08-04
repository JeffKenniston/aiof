package consensus

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestWeightedBFTConsensus(t *testing.T) {
	cfg := DefaultBFTConfig()
	engine := NewBFTConsensusEngine(cfg)

	// Register Agent Reputations
	engine.RegisterAgentReputation(AgentReputation{AgentID: "agent-1", AccuracyScore: 0.95})
	engine.RegisterAgentReputation(AgentReputation{AgentID: "agent-2", AccuracyScore: 0.85})
	engine.RegisterAgentReputation(AgentReputation{AgentID: "agent-3", AccuracyScore: 0.40}) // Low reputation

	proposal := Proposal{
		ID:               "prop-101",
		ProposerID:       "agent-1",
		Title:            "Refactor AST Dependency Graph",
		CodeDiff:         "+ func Refactor() { return }",
		ASTPassRate:      1.0,
		UnitTestPassRate: 1.0,
		SMTProofScore:    1.0,
	}

	err := engine.SubmitProposal(proposal)
	if err != nil {
		t.Fatalf("Failed to submit proposal: %v", err)
	}

	// Cast Votes
	_ = engine.CastVote(Vote{AgentID: "agent-1", ProposalID: "prop-101", Approve: true, Confidence: 1.0})
	_ = engine.CastVote(Vote{AgentID: "agent-2", ProposalID: "prop-101", Approve: true, Confidence: 0.9})
	_ = engine.CastVote(Vote{AgentID: "agent-3", ProposalID: "prop-101", Approve: false, Confidence: 0.5})

	res, err := engine.EvaluateConsensus(context.Background(), "prop-101")
	if err != nil {
		t.Fatalf("Consensus evaluation failed: %v", err)
	}

	if !res.QuorumReached || res.Status != StatusApproved {
		t.Errorf("Expected proposal to be approved with BFT quorum, got status %s", res.Status)
	}

	if res.ApproveRatio < 0.667 {
		t.Errorf("Expected approve ratio >= 0.667, got %f", res.ApproveRatio)
	}
}

func TestSMTSolverGateInProcess(t *testing.T) {
	gate := NewSMTSolverGate()

	req := SMTVerificationRequest{
		ProposalID: "prop-202",
		Language:   "go",
		CodeDiff:   "func Process(data []byte) { if data == nil { panic(\"nil pointer\") } }",
		Assertions: []IVLAssertion{
			{
				ID:          "inv-1",
				Type:        AssertNullSafety,
				Expression:  "(not ptr_is_null)",
				Description: "Code must not trigger nil pointer panic",
			},
		},
		Timeout: 2 * time.Second,
	}

	res, err := gate.VerifyProposal(context.Background(), req)
	if err != nil {
		t.Fatalf("SMT verification failed: %v", err)
	}

	if res.Verified {
		t.Errorf("Expected SMT verification to fail due to panic in code diff, but got verified")
	}

	if len(res.Violations) == 0 {
		t.Errorf("Expected violation details in result")
	}
}

func TestVRFTieBreakerAndSlashing(t *testing.T) {
	engine := NewBFTConsensusEngine(DefaultBFTConfig())

	engine.RegisterAgentReputation(AgentReputation{AgentID: "agent-a", AccuracyScore: 0.90})
	engine.RegisterAgentReputation(AgentReputation{AgentID: "agent-b", AccuracyScore: 0.88})

	// 1. VRF Tie-Breaker
	beacon := NewVRFBeacon([]byte("enclave-secret-1234"))
	candidates := []string{"agent-a", "agent-b"}

	winner, err := beacon.SelectTieBreakerArbitrator("round-1", "prop-303", candidates, engine)
	if err != nil {
		t.Fatalf("VRF tie-breaker failed: %v", err)
	}

	if winner.AgentID == "" || winner.CombinedRank <= 0.0 {
		t.Errorf("Invalid VRF tie-breaker winner result: %+v", winner)
	}

	// 2. Dynamic Slashing
	slasher := NewDynamicSlasher(engine)

	log, err := slasher.ApplySlashing("agent-a", "prop-303", SlashSMTViolation, "SMT solver invariant violation")
	if err != nil {
		t.Fatalf("Slashing failed: %v", err)
	}

	if log.NewScore >= 0.90 {
		t.Errorf("Expected agent-a score to decay after 25%% SMT slash, got %f", log.NewScore)
	}

	expectedScore := math.Round((0.90 - (0.90 * 0.25)) * 1000) / 1000
	if math.Abs(log.NewScore-expectedScore) > 0.01 {
		t.Errorf("Expected slashed score ~%f, got %f", expectedScore, log.NewScore)
	}
}

func TestPassKStatisticalVerification(t *testing.T) {
	engine := NewBFTConsensusEngine(DefaultBFTConfig())

	// Test 1: CalculatePassK estimator check
	if score := CalculatePassK(3, 3, 3); score != 1.0 {
		t.Fatalf("expected pass^k 1.0 for 3/3 passed, got %f", score)
	}
	if score := CalculatePassK(3, 1, 3); score != 0.0 {
		t.Fatalf("expected pass^k 0.0 for 1/3 passed when k=3, got %f", score)
	}

	// Test 2: EvaluatePassK multi-attempt verification
	prop := &Proposal{
		ID:         "prop-passk-1",
		ProposerID: "agent-evaluator",
		Title:      "Fix flakiness",
	}

	score, err := engine.EvaluatePassK(context.Background(), prop, 3, 0.80, func(ctx context.Context, trialIdx int) (bool, error) {
		return true, nil // All 3 trials succeed
	})
	if err != nil || score != 1.0 {
		t.Fatalf("expected successful pass^k verification, got err=%v score=%f", err, score)
	}
	if prop.PassKScore != 1.0 || prop.PassKTrials != 3 {
		t.Fatalf("expected proposal pass_k fields updated, got score=%f trials=%d", prop.PassKScore, prop.PassKTrials)
	}
}

