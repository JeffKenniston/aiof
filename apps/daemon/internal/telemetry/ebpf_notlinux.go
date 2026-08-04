//go:build !linux

package telemetry

import (
	"context"
	"fmt"
	"log/slog"
)

type EBPFHook struct {
	logger *slog.Logger
}

func NewEBPFHook(logger *slog.Logger) *EBPFHook {
	return &EBPFHook{
		logger: logger,
	}
}

func (e *EBPFHook) Attach(ctx context.Context, pid int) error {
	e.logger.Info("eBPF hooks not supported on this platform, skipping.")
	return nil
}

func (e *EBPFHook) Correlate(ctx context.Context) error {
	return fmt.Errorf("correlate: perf reader is nil, call Attach() before Correlate()")
}

func (e *EBPFHook) Detach() {
}
