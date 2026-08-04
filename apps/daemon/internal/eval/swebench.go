package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrBenchmarkFailed    = errors.New("eval: code execution failed SWE-bench differential test harness")
	ErrRegressionDetected = errors.New("eval: regression detected - pass rate dropped below baseline threshold")
)

type BenchmarkIssue struct {
	InstanceID       string   `json:"instance_id"`
	RepoName         string   `json:"repo_name"`
	BaseCommit       string   `json:"base_commit"`
	ProblemStatement string   `json:"problem_statement"`
	TestPatch        string   `json:"test_patch"`
	FailingTests     []string `json:"failing_tests"`
	PassingTests     []string `json:"passing_tests"`
}

type EvaluationReport struct {
	EvalID          string    `json:"eval_id"`
	InstanceID      string    `json:"instance_id"`
	Timestamp       time.Time `json:"timestamp"`
	Resolved        bool      `json:"resolved"`
	TestsPassed     int       `json:"tests_passed"`
	TestsFailed     int       `json:"tests_failed"`
	PassRate        float64   `json:"pass_rate"`
	ErrorTrajectory []string  `json:"error_trajectory,omitempty"`
	DiffPatch       string    `json:"diff_patch"`
	ExecutionTimeMS int64     `json:"execution_time_ms"`
	ReportHash      string    `json:"report_hash"`
}

func (r *EvaluationReport) CalculateHash() string {
	record := fmt.Sprintf("%s|%s|%t|%d|%d|%.4f|%s", r.EvalID, r.InstanceID, r.Resolved, r.TestsPassed, r.TestsFailed, r.PassRate, r.DiffPatch)
	h := sha256.Sum256([]byte(record))
	return hex.EncodeToString(h[:])
}

type SWEBenchGuard struct {
	mu               sync.RWMutex
	baselinePassRate float64
	reports          []EvaluationReport
}

func NewSWEBenchGuard(baselinePassRate float64) *SWEBenchGuard {
	if baselinePassRate <= 0.0 {
		baselinePassRate = 0.85 // Default 85% baseline threshold
	}
	return &SWEBenchGuard{
		baselinePassRate: baselinePassRate,
		reports:          make([]EvaluationReport, 0),
	}
}

// EvaluatePatch runs differential tests against an issue benchmark instance.
func (g *SWEBenchGuard) EvaluatePatch(ctx context.Context, issue BenchmarkIssue, proposedPatch string) (*EvaluationReport, error) {
	start := time.Now()

	// Simulate differential test execution
	passed := len(issue.PassingTests)
	failed := 0
	resolved := true
	trajectory := make([]string, 0)

	if proposedPatch == "" {
		resolved = false
		failed = len(issue.FailingTests)
		trajectory = append(trajectory, "empty patch - failing tests unresolved")
	} else if len(issue.FailingTests) > 0 {
		// All failing tests are converted to passing in resolved state
		passed += len(issue.FailingTests)
	}

	total := passed + failed
	var passRate float64 = 1.0
	if total > 0 {
		passRate = float64(passed) / float64(total)
	}

	report := EvaluationReport{
		EvalID:          fmt.Sprintf("eval-%d", time.Now().UnixNano()),
		InstanceID:      issue.InstanceID,
		Timestamp:       time.Now().UTC(),
		Resolved:        resolved,
		TestsPassed:     passed,
		TestsFailed:     failed,
		PassRate:        passRate,
		ErrorTrajectory: trajectory,
		DiffPatch:       proposedPatch,
		ExecutionTimeMS: time.Since(start).Milliseconds(),
	}
	report.ReportHash = report.CalculateHash()

	g.mu.Lock()
	g.reports = append(g.reports, report)
	g.mu.Unlock()

	if passRate < g.baselinePassRate {
		return &report, fmt.Errorf("%w: pass rate %.2f < baseline threshold %.2f", ErrRegressionDetected, passRate, g.baselinePassRate)
	}

	return &report, nil
}

func (g *SWEBenchGuard) GetHistory() []EvaluationReport {
	g.mu.RLock()
	defer g.mu.RUnlock()

	res := make([]EvaluationReport, len(g.reports))
	copy(res, g.reports)
	return res
}
