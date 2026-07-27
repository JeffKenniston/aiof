package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"aiof/internal/sandbox"
	"google.golang.org/genai"
)

// ArchitectAgent implements the Agent interface for generating high-level system designs, ADRs, and architectures.
type ArchitectAgent struct {
	logger *slog.Logger
	client *genai.Client
}

// NewArchitectAgent creates a new Architecture agent.
func NewArchitectAgent(logger *slog.Logger, client *genai.Client) *ArchitectAgent {
	return &ArchitectAgent{
		logger: logger,
		client: client,
	}
}

func (a *ArchitectAgent) handleToolCall(ctx context.Context, host AgentHost, toolName string, args map[string]interface{}, projectDir string, sandboxEnv *sandbox.MCPContainer, proposedChanges *string) (string, bool) {
	var result string
	isComplete := false
	agentName := "architecture_agent"

	if toolName == "TaskComplete" {
		host.EmitLog(ctx, agentName, "info", "Architecture TaskComplete requested. Yielding to Orchestrator.")
		isComplete = true
	} else if toolName == "WriteFile" {
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)

		*proposedChanges += fmt.Sprintf("File %s:\n%s\n\n", path, content)

		fullPath := filepath.Join(projectDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			result = fmt.Sprintf("Error writing file: %v", err)
		} else {
			result = "Architectural document written successfully."
		}
		host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Wrote file: %s", path))
	} else if toolName == "RunCommand" {
		cmdStr, _ := args["command"].(string)
		host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Running architect command in sandbox: %s", cmdStr))

		cmdCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		outStr, err := sandboxEnv.ExecuteCode(cmdCtx, []byte(cmdStr))
		cancel()

		if err != nil && outStr == "" {
			result = fmt.Sprintf("Sandbox Error: %v", err)
		} else if err != nil {
			result = fmt.Sprintf("Error: %v\nOutput: %s", err, outStr)
		} else {
			result = fmt.Sprintf("Output: %s", outStr)
		}

		if len(result) > 2000 {
			result = result[:2000] + "\n...[truncated]"
		}
	} else if toolName == "ReadFile" {
		path, _ := args["path"].(string)
		fullPath := filepath.Join(projectDir, path)
		fileContent, err := os.ReadFile(fullPath)
		if err != nil {
			result = fmt.Sprintf("Error reading file: %v", err)
		} else {
			result = string(fileContent)
			if len(result) > 4000 {
				result = result[:4000] + "\n...[truncated]"
			}
		}
		host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Read file: %s", path))
	} else if toolName == "SendReply" {
		msg, _ := args["message"].(string)
		chatMsg := chatMessage{
			ID:        fmt.Sprintf("reply-%d", time.Now().UnixNano()),
			Role:      "agent",
			Content:   msg,
			AgentName: agentName,
			Timestamp: time.Now().UnixMilli(),
		}
		chatPayload, err := json.Marshal(chatMsg)
		if err == nil {
			host.EmitState(ctx, "chat_messages", chatMsg.ID, string(chatPayload))
		}
		host.EmitLog(ctx, agentName, "info", "Sent architecture proposal to user.")
		result = "Reply sent."
	}

	return result, isComplete
}

// Execute implements the Agent interface.
func (a *ArchitectAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	a.logger.Info("ArchitectAgent executing", slog.String("input", input))
	agentName := "architecture_agent"
	projectDir := host.GetProjectDir()

	if a.client == nil {
		return "", fmt.Errorf("ArchitectAgent requires a valid genai.Client")
	}

	host.EmitLog(ctx, agentName, "info", fmt.Sprintf("Architect Agent started task: %s", input))
	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "Architect", "model": "Pro", "color": "#ff79c6", "status": "running"}`, agentName))

	config := &genai.GenerateContentConfig{
		Tools: toolRegistry, // Architects need to write Architecture Decision Records (ADRs) and plans
	}

	sysPrompt := fmt.Sprintf(`%s.
Your task: %s
Working directory: %s

You must analyze requirements, design systemic architectures, write Architecture Decision Records (ADR.md), and define API contracts.
You must adhere strictly to the 2026 core mandates: enforce C++26/Rust2024 strict memory safety, apply OWASP LLM 2025 security principles, utilize Testcontainers over local mocks, and integrate eBPF observability over sidecars.
You operate at the macro level. Do not get bogged down in micro-implementations.
Always verify your steps. When done, call TaskComplete.`, GetPersonaPrompt(agentName, "specialist"), input, projectDir)

	config.SystemInstruction = &genai.Content{
		Role: "system",
		Parts: []*genai.Part{
			{Text: sysPrompt},
		},
	}

	chat, err := a.client.Chats.Create(ctx, "gemini-3.1-pro-preview", config, nil)
	if err != nil {
		host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Failed to create chat: %v", err))
		return "", err
	}

	var proposedChanges string

	sandboxEnv := sandbox.NewMCPContainer(a.logger, sandbox.RuntimeHost, projectDir)
	if err := sandboxEnv.Start(ctx); err != nil {
		host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Architect Sandbox failed to start: %v", err))
		return "", err
	}
	
	sandboxCtx, cancelSandbox := context.WithCancel(ctx)
	defer func() {
		cancelSandbox()
		sandboxEnv.Stop()
	}()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-sandboxCtx.Done():
				return
			case <-ticker.C:
				cpu, memMb, uptime := sandboxEnv.GetStats()
				sandboxState := fmt.Sprintf(`{"id": "sandbox-%s", "name": "mcp-architect-env", "status": "running", "cpuPercent": %.1f, "memoryPercent": %.1f, "memoryMb": %d, "memoryLimitMb": %d, "uptimeSeconds": %d}`,
					agentName, cpu, float64(memMb)/1024.0*100, memMb, 1024, uptime)
				host.EmitState(ctx, "sandbox_state", "sandbox-"+agentName, sandboxState)
			}
		}
	}()

	// Action Loop
	for i := 0; i < 150; i++ { // max 150 steps
		resp, err := chat.SendMessage(ctx, genai.Part{Text: "Proceed with your architectural task. Remember to use tools. If done, call TaskComplete."})
		if err != nil {
			host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Architect Chat error: %v", err))
			host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "Architect", "model": "Pro", "color": "#ff5555", "status": "error"}`, agentName))
			return proposedChanges, err
		}

		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
			break
		}

		var fc *genai.FunctionCall
		var agentText string

		for _, part := range resp.Candidates[0].Content.Parts {
			if part.FunctionCall != nil {
				fc = part.FunctionCall
			} else if part.Text != "" {
				agentText += part.Text
			}
		}

		if agentText != "" {
			host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Architect Agent thinking: %s", agentText))
		}

		if fc != nil {
			host.EmitLog(ctx, agentName, "info", fmt.Sprintf("Architect Executing tool: %s", fc.Name))
			
			result, isComplete := a.handleToolCall(ctx, host, fc.Name, fc.Args, projectDir, sandboxEnv, &proposedChanges)
			
			if isComplete {
				break
			}

			tr := toolResponse{Result: result}
			trMap := map[string]interface{}{"result": tr.Result}
			_, err = chat.SendMessage(ctx, genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     fc.Name,
					Response: trMap,
				},
			})
			if err != nil {
				host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Error sending tool response: %v", err))
				break
			}
		} else {
			_, err = chat.SendMessage(ctx, genai.Part{Text: "Please call a tool or TaskComplete."})
			if err != nil {
				break
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "Architect", "model": "Pro", "color": "#50fa7b", "status": "success"}`, agentName))
	return proposedChanges, nil
}
