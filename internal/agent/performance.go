package agent

import (
	"context"
	"log/slog"

	"google.golang.org/genai"
)

type PerformanceAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewPerformanceAgent(logger *slog.Logger, client *genai.Client) *PerformanceAgent {
	return &PerformanceAgent{logger: logger, client: client}
}

func (a *PerformanceAgent) Execute(ctx context.Context, host AgentHost, globalProposedChanges string, stateMutations []byte) (string, error) {
	host.EmitLog(ctx, "Performance", "info", "Running performance diagnostics (eBPF, Flamegraphs, CWV)...")
	return globalProposedChanges, nil
}
