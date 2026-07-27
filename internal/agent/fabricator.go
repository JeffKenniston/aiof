package agent

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/genai"
)

type FabricatorAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewFabricatorAgent(logger *slog.Logger, client *genai.Client) *FabricatorAgent {
	return &FabricatorAgent{
		logger: logger,
		client: client,
	}
}

func (a *FabricatorAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	a.logger.Info("FabricatorAgent executing Arbiter Pattern to dynamically instantiate ephemeral agents")
	host.EmitLog(ctx, "FabricatorAgent", "info", "Evaluating complex intent and authoring ephemeral agent prompt...")

	host.EmitState(ctx, "agent_graph_state", "fabricator", fmt.Sprintf(`{"id": "fabricator", "label": "Fabricator", "model": "Pro", "color": "#bd93f9", "status": "running"}`))

	// Prompt generation for dynamic agent
	prompt := fmt.Sprintf(`You are the Fabricator Agent. A task has exceeded static catalog capabilities.
Task Intent: %s
Generate a highly specialized system prompt for an ephemeral agent that can solve this exact task.`, input)

	var agentPrompt string
	if a.client != nil {
		resp, err := a.client.Models.GenerateContent(ctx, "gemini-3.1-pro-preview", genai.Text(prompt), nil)
		if err != nil {
			host.EmitState(ctx, "agent_graph_state", "fabricator", fmt.Sprintf(`{"id": "fabricator", "label": "Fabricator", "model": "Pro", "color": "#ff5555", "status": "error"}`))
			return "", fmt.Errorf("failed to fabricate agent prompt: %v", err)
		}
		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			part := resp.Candidates[0].Content.Parts[0]
			if part.Text != "" {
				agentPrompt = part.Text
			}
		}
	} else {
		agentPrompt = "System Prompt: You are a specialized ephemeral agent generated to solve this task."
	}

	host.EmitLog(ctx, "FabricatorAgent", "info", fmt.Sprintf("Instantiating ephemeral agent with prompt length: %d", len(agentPrompt)))
	host.EmitState(ctx, "agent_graph_state", "fabricator", fmt.Sprintf(`{"id": "fabricator", "label": "Fabricator", "model": "Pro", "color": "#50fa7b", "status": "success"}`))

	// Here we could dynamically dispatch to the newly generated agent, but for now we'll just pass the system prompt downstream.
	return fmt.Sprintf("FabricatorAgent deployed Ephemeral Agent with Prompt:\n%s", agentPrompt), nil
}
