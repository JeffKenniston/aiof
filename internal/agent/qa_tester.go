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

// QATesterAgent implements the Agent interface for running automated UI testing and Playwright tasks.
type QATesterAgent struct {
	logger *slog.Logger
	client *genai.Client
}

// NewQATesterAgent creates a new QA Tester agent.
func NewQATesterAgent(logger *slog.Logger, client *genai.Client) *QATesterAgent {
	return &QATesterAgent{
		logger: logger,
		client: client,
	}
}

func (q *QATesterAgent) handleToolCall(ctx context.Context, host AgentHost, toolName string, args map[string]interface{}, projectDir string, sandboxEnv *sandbox.MCPContainer) (string, bool) {
	var result string
	isComplete := false
	agentName := "qa_tester_agent"

	if toolName == "TaskComplete" {
		host.EmitLog(ctx, agentName, "info", "QA Testing TaskComplete requested.")
		isComplete = true
	} else if toolName == "WriteFile" {
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		fullPath := filepath.Join(projectDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			result = fmt.Sprintf("Error writing test file: %v", err)
		} else {
			result = "Test file written successfully."
		}
		host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Wrote test file: %s", path))
	} else if toolName == "RunCommand" {
		cmdStr, _ := args["command"].(string)
		host.EmitLog(ctx, agentName, "info", fmt.Sprintf("Running QA task in sandbox: %s", cmdStr))

		// QA tasks (like playwright) might take longer, allow 2 minutes
		cmdCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		outStr, err := sandboxEnv.ExecuteCode(cmdCtx, []byte(cmdStr))
		cancel()

		if err != nil && outStr == "" {
			result = fmt.Sprintf("QA Sandbox Error: %v", err)
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
		host.EmitLog(ctx, agentName, "info", "Sent QA report to user.")
		result = "Reply sent."
	}

	return result, isComplete
}

// Execute implements the Agent interface.
func (q *QATesterAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	q.logger.Info("QATesterAgent executing", slog.String("input", input))
	agentName := "qa_tester_agent"
	projectDir := host.GetProjectDir()

	if q.client == nil {
		return "", fmt.Errorf("QATesterAgent requires a valid genai.Client")
	}

	host.EmitLog(ctx, agentName, "info", fmt.Sprintf("QA Agent started task: %s", input))
	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "QA Tester", "model": "Pro", "color": "#bd93f9", "status": "running"}`, agentName))

	config := &genai.GenerateContentConfig{
		Tools: toolRegistry, // Shares the same tools as the Code Developer (WriteFile, RunCommand, etc.)
	}

	sysPrompt := fmt.Sprintf(`You are an expert QA Tester agent named %s.
Your task: %s
Working directory: %s

You must use the provided tools to write and execute automated tests (e.g. Playwright, Vitest).
IMPORTANT: When executing terminal commands, you MUST use --yes or -y flags so they run non-interactively.
If you need to install testing frameworks, do so.
Always verify your tests pass. When done, call TaskComplete.`, agentName, input, projectDir)

	config.SystemInstruction = &genai.Content{
		Role: "system",
		Parts: []*genai.Part{
			{Text: sysPrompt},
		},
	}

	chat, err := q.client.Chats.Create(ctx, "gemini-2.5-pro", config, nil)
	if err != nil {
		host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Failed to create chat: %v", err))
		return "", err
	}

	sandboxEnv := sandbox.NewMCPContainer(q.logger, sandbox.RuntimeHost, projectDir)
	if err := sandboxEnv.Start(ctx); err != nil {
		host.EmitLog(ctx, agentName, "error", fmt.Sprintf("QA Sandbox failed to start: %v", err))
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
				sandboxState := fmt.Sprintf(`{"id": "sandbox-%s", "name": "mcp-qa-env", "status": "running", "cpuPercent": %.1f, "memoryPercent": %.1f, "memoryMb": %d, "memoryLimitMb": %d, "uptimeSeconds": %d}`,
					agentName, cpu, float64(memMb)/1024.0*100, memMb, 1024, uptime)
				host.EmitState(ctx, "sandbox_state", "sandbox-"+agentName, sandboxState)
			}
		}
	}()

	// QA Action Loop
	for i := 0; i < 150; i++ { // max 150 steps
		resp, err := chat.SendMessage(ctx, genai.Part{Text: "Proceed with your testing task. Remember to use tools. If done, call TaskComplete."})
		if err != nil {
			host.EmitLog(ctx, agentName, "error", fmt.Sprintf("QA Chat error: %v", err))
			host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "QA Tester", "model": "Pro", "color": "#ff5555", "status": "error"}`, agentName))
			return "", err
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
			host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("QA Agent thinking: %s", agentText))
		}

		if fc != nil {
			host.EmitLog(ctx, agentName, "info", fmt.Sprintf("QA Executing tool: %s", fc.Name))
			
			result, isComplete := q.handleToolCall(ctx, host, fc.Name, fc.Args, projectDir, sandboxEnv)
			
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

	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "QA Tester", "model": "Pro", "color": "#50fa7b", "status": "success"}`, agentName))
	
	// QA Tester does not propose direct code changes, it just runs tests, so we return empty string for changes.
	return "", nil
}
