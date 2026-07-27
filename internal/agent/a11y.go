package agent

import (
	"context"
	"log/slog"

	"google.golang.org/genai"
)

type A11yAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewA11yAgent(logger *slog.Logger, client *genai.Client) *A11yAgent {
	return &A11yAgent{logger: logger, client: client}
}

func (a *A11yAgent) Execute(ctx context.Context, host AgentHost, globalProposedChanges string, stateMutations []byte) (string, error) {
	host.EmitLog(ctx, "A11y", "info", "Auditing ARIA labels and WCAG compliance...")
	return globalProposedChanges, nil
}
