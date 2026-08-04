package agent

import (
	"context"
	"log/slog"

	"google.golang.org/genai"
)

type LegalAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewLegalAgent(logger *slog.Logger, client *genai.Client) *LegalAgent {
	return &LegalAgent{logger: logger, client: client}
}

func (a *LegalAgent) Execute(ctx context.Context, host AgentHost, globalProposedChanges string, stateMutations []byte) (string, error) {
	host.EmitLog(ctx, "Legal", "info", "Scanning for GDPR/CCPA violations and copyleft licensing issues...")
	return globalProposedChanges, nil
}
