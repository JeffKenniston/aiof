package orchestrator

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestProcessTask_NilGeminiClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	o := NewOrchestrator(logger, nil, nil, nil, nil)

	ctx := context.Background()
	// Should not panic, but should return early since geminiClient is nil
	o.processTask(ctx, "test prompt", ".", "default", "", "")
}

func TestStartBackgroundListener_NilStore(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	o := NewOrchestrator(logger, nil, nil, nil, nil)

	ctx := context.Background()
	// Should return immediately because store is nil
	o.StartBackgroundListener(ctx)
}
