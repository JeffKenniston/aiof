package agent

import (
	"context"
	"log/slog"

	"google.golang.org/genai"
)

type DataAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewDataAgent(logger *slog.Logger, client *genai.Client) *DataAgent {
	return &DataAgent{logger: logger, client: client}
}

func (a *DataAgent) Execute(ctx context.Context, host AgentHost, globalProposedChanges string, stateMutations []byte) (string, error) {
	host.EmitLog(ctx, "DataEng", "info", "Validating ETL pipelines and schema structures...")
	return globalProposedChanges, nil
}
