package eval

import (
	"context"
	"testing"
)

func TestSWEBenchGuard(t *testing.T) {
	guard := NewSWEBenchGuard(0.80)

	issue := BenchmarkIssue{
		InstanceID:       "django-12345",
		RepoName:         "django/django",
		BaseCommit:       "commit-abc123",
		ProblemStatement: "Fix null pointer in ORM filter",
		FailingTests:     []string{"test_orm_null_filter"},
		PassingTests:     []string{"test_orm_basic", "test_orm_select"},
	}

	// Valid patch -> pass
	report, err := guard.EvaluatePatch(context.Background(), issue, "diff --git a/orm.py b/orm.py\n+ if val is None: return")
	if err != nil {
		t.Fatalf("unexpected failure: %v", err)
	}

	if !report.Resolved || report.PassRate != 1.0 {
		t.Errorf("expected 100%% pass rate, got %.2f", report.PassRate)
	}

	// Empty patch -> regression failure
	_, err = guard.EvaluatePatch(context.Background(), issue, "")
	if err == nil {
		t.Errorf("expected regression error for empty patch")
	}
}
