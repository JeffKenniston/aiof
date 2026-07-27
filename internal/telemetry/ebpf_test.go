package telemetry

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestEBPFHook_CorrelateBeforeAttach(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	hook := NewEBPFHook(logger)

	ctx := context.Background()
	
	// Test Correlate safely fails if unattached
	err := hook.Correlate(ctx)
	if err == nil {
		t.Fatal("Expected error when calling Correlate before Attach")
	}
	if !strings.Contains(err.Error(), "perf reader is nil") {
		t.Errorf("Unexpected error message: %v", err)
	}
	
	// We do not require a successful Attach since it requires root/eBPF permissions,
	// but we test the Detach logic handles unattached states gracefully.
	hook.Detach()
}
