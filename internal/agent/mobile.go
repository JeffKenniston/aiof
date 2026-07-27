package agent

import (
	"context"
	"log/slog"

	"google.golang.org/genai"
)

type MobileAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewMobileAgent(logger *slog.Logger, client *genai.Client) *MobileAgent {
	return &MobileAgent{logger: logger, client: client}
}

func (a *MobileAgent) Execute(ctx context.Context, host AgentHost, globalProposedChanges string, stateMutations []byte) (string, error) {
	host.EmitLog(ctx, "MobileDev", "info", "Analyzing cross-platform constraints and native execution...")
	return globalProposedChanges, nil
}
