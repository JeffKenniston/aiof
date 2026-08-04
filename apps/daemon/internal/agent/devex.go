package agent

import (
	"context"
	"log/slog"

	"google.golang.org/genai"
)

type DevExAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewDevExAgent(logger *slog.Logger, client *genai.Client) *DevExAgent {
	return &DevExAgent{logger: logger, client: client}
}

func (a *DevExAgent) Execute(ctx context.Context, host AgentHost, globalProposedChanges string, stateMutations []byte) (string, error) {
	host.EmitLog(ctx, "DevEx", "info", "Analyzing build times and CI/CD pipeline latency...")
	return globalProposedChanges, nil
}
