package agent

import (
	"context"
	"log/slog"

	"google.golang.org/genai"
)

type FinOpsAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewFinOpsAgent(logger *slog.Logger, client *genai.Client) *FinOpsAgent {
	return &FinOpsAgent{logger: logger, client: client}
}

func (a *FinOpsAgent) Execute(ctx context.Context, host AgentHost, globalProposedChanges string, stateMutations []byte) (string, error) {
	host.EmitLog(ctx, "FinOps", "info", "Monitoring cloud costs and generating Terraform scale-down scripts...")
	return globalProposedChanges, nil
}
