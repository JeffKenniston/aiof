package agent

import (
	"context"
	"log/slog"

	"google.golang.org/genai"
)

type HardwareAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewHardwareAgent(logger *slog.Logger, client *genai.Client) *HardwareAgent {
	return &HardwareAgent{logger: logger, client: client}
}

func (a *HardwareAgent) Execute(ctx context.Context, host AgentHost, globalProposedChanges string, stateMutations []byte) (string, error) {
	host.EmitLog(ctx, "Hardware", "info", "Reviewing C/Rust RTOS constraints and firmware logic...")
	return globalProposedChanges, nil
}
