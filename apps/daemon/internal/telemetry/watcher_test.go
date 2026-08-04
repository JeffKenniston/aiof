package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWatcherAgentMetricsAndReport(t *testing.T) {
	watcher := NewWatcherAgent()

	// Record sample metric in Cloud Mode (Gemini 3.1 Flash)
	watcher.RecordMetric(ExecutionMetric{
		TaskID:               "task-001",
		Mode:                 "CLOUD_GEMINI",
		ModelName:            "gemini-3.1-flash",
		HardwareDevice:       "CLOUD_API",
		TTFT:                 180 * time.Millisecond,
		TotalDuration:        850 * time.Millisecond,
		OrchestrationLatency: 15 * time.Millisecond,
		LLMComputeLatency:    835 * time.Millisecond,
		PromptTokens:         120,
		CompletionTokens:     45,
		PassKScore:           1.0,
		MemoryUsageMB:        145.0,
	})

	metrics := watcher.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 recorded metric, got %d", len(metrics))
	}

	if metrics[0].TokensPerSecond <= 0 {
		t.Fatalf("expected positive tokens_per_second calculation, got %f", metrics[0].TokensPerSecond)
	}

	report := watcher.GenerateComparisonReport()
	if !strings.Contains(report, "AIOF Framework Execution & Telemetry Benchmark Report") {
		t.Fatalf("expected report title in output report")
	}

	if !strings.Contains(report, "LangChain (Python)") {
		t.Fatalf("expected industry baseline table in report")
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "benchmark_results.md")
	if err := watcher.SaveReportArtifact(outPath); err != nil {
		t.Fatalf("SaveReportArtifact failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil || len(data) == 0 {
		t.Fatalf("failed to read saved artifact file: %v", err)
	}
}
