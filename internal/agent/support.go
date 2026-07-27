package agent

import (
	"context"
	"log/slog"

	"google.golang.org/genai"
)

type SupportAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewSupportAgent(logger *slog.Logger, client *genai.Client) *SupportAgent {
	return &SupportAgent{logger: logger, client: client}
}

func (a *SupportAgent) Execute(ctx context.Context, host AgentHost, globalProposedChanges string, stateMutations []byte) (string, error) {
	host.EmitLog(ctx, "Support", "info", "Drafting contextual user help articles based on recent code changes...")
	return globalProposedChanges, nil
}
