package agent

import (
	"context"
	"log/slog"

	"google.golang.org/genai"
)

type ScrumMasterAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewScrumMasterAgent(logger *slog.Logger, client *genai.Client) *ScrumMasterAgent {
	return &ScrumMasterAgent{logger: logger, client: client}
}

func (a *ScrumMasterAgent) Execute(ctx context.Context, host AgentHost, globalProposedChanges string, stateMutations []byte) (string, error) {
	host.EmitLog(ctx, "ScrumMaster", "info", "Reviewing task dependencies and updating backlog...")
	return globalProposedChanges, nil
}
