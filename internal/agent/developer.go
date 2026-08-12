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

var toolRegistry = []*genai.Tool{
	{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "WriteFile",
				Description: "Write content to a file at the specified path. This creates folders if they do not exist.",
				Parameters: &genai.Schema{
					Type:       genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"path":    {Type: genai.TypeString},
						"content": {Type: genai.TypeString},
					},
					Required: []string{"path", "content"},
				},
			},
			{
				Name:        "RunCommand",
				Description: "Run a terminal command in the project directory (e.g. npm install, mkdir).",
				Parameters: &genai.Schema{
					Type:       genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"command": {Type: genai.TypeString},
					},
					Required: []string{"command"},
				},
			},
			{
				Name:        "ReadFile",
				Description: "Read content of a file at the specified path.",
				Parameters: &genai.Schema{
					Type:       genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"path": {Type: genai.TypeString},
					},
					Required: []string{"path"},
				},
			},
			{
				Name:        "SendReply",
				Description: "Send a final chat reply to the user with your analysis/findings.",
				Parameters: &genai.Schema{
					Type:       genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"message": {Type: genai.TypeString},
					},
					Required: []string{"message"},
				},
			},
			{
				Name:        "TaskComplete",
				Description: "Call this when the task is fully completed to end your execution loop.",
			},
		},
	},
}

type toolResponse struct {
	Result string `json:"result"`
}

type chatMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	AgentName string `json:"agentName"`
	Timestamp int64  `json:"timestamp"`
}

// CodeDeveloperAgent implements the Agent interface for generating code and making system changes.
type CodeDeveloperAgent struct {
	logger *slog.Logger
	client *genai.Client
}

// NewCodeDeveloperAgent creates a new code developer agent.
func NewCodeDeveloperAgent(logger *slog.Logger, client *genai.Client) *CodeDeveloperAgent {
	return &CodeDeveloperAgent{
		logger: logger,
		client: client,
	}
}

func (c *CodeDeveloperAgent) handleToolCall(ctx context.Context, host AgentHost, toolName string, args map[string]interface{}, projectDir string, sandboxEnv *sandbox.MCPContainer, proposedChanges *string) (string, bool) {
	var result string
	isComplete := false
	agentName := "code_developer_agent"

	if toolName == "TaskComplete" {
		host.EmitLog(ctx, agentName, "info", "TaskComplete requested.")
		// In decoupled arch, Validator will run after Dispatch returns, so we just complete.
		host.EmitLog(ctx, agentName, "info", "Task complete. Yielding back to Orchestrator.")
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
			result = "File written successfully."
		}
		host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Wrote file: %s", path))
	} else if toolName == "RunCommand" {
		cmdStr, _ := args["command"].(string)
		host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Running in sandbox: %s", cmdStr))

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
		if err != nil {
			host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Failed to marshal chat message: %v", err))
		} else {
			host.EmitState(ctx, "chat_messages", chatMsg.ID, string(chatPayload))
		}
		host.EmitLog(ctx, agentName, "info", "Sent reply to user.")
		result = "Reply sent."
	}

	return result, isComplete
}

// Execute implements the Agent interface.
func (c *CodeDeveloperAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	c.logger.Info("CodeDeveloperAgent executing", slog.String("input", input))
	agentName := "code_developer_agent"
	projectDir := host.GetProjectDir()

	if c.client == nil {
		return "", fmt.Errorf("CodeDeveloperAgent requires a valid genai.Client")
	}

	host.EmitLog(ctx, agentName, "info", fmt.Sprintf("Agent %s started task: %s", agentName, input))
	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "%s", "model": "Pro", "color": "#f1fa8c", "status": "running"}`, agentName, agentName))

	config := &genai.GenerateContentConfig{
		Tools: toolRegistry,
	}

	sysPrompt := fmt.Sprintf(`You are an expert developer agent named %s.
Your task: %s
Working directory: %s

You must use the provided tools to scaffold the project, write code, or execute terminal commands.
IMPORTANT: When executing terminal commands, you MUST use --yes or -y flags so they run non-interactively.
Always verify your steps. When done, call TaskComplete.`, agentName, input, projectDir)

	config.SystemInstruction = &genai.Content{
		Role: "system",
		Parts: []*genai.Part{
			{Text: sysPrompt},
		},
	}

	chat, err := c.client.Chats.Create(ctx, "gemini-2.5-pro", config, nil)
	if err != nil {
		host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Failed to create chat: %v", err))
		return "", err
	}

	var proposedChanges string

	sandboxEnv := sandbox.NewMCPContainer(c.logger, sandbox.RuntimeHost, projectDir)
	if err := sandboxEnv.Start(ctx); err != nil {
		host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Sandbox failed to start: %v", err))
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
				sandboxState := fmt.Sprintf(`{"id": "sandbox-%s", "name": "mcp-execution-env", "status": "running", "cpuPercent": %.1f, "memoryPercent": %.1f, "memoryMb": %d, "memoryLimitMb": %d, "uptimeSeconds": %d}`,
					agentName, cpu, float64(memMb)/1024.0*100, memMb, 1024, uptime)
				host.EmitState(ctx, "sandbox_state", "sandbox-"+agentName, sandboxState)
			}
		}
	}()

	// Action Loop
	for i := 0; i < 150; i++ { // max 150 steps
		resp, err := chat.SendMessage(ctx, genai.Part{Text: "Proceed with your task. Remember to use tools. If done, call TaskComplete."})
		if err != nil {
			host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Chat error: %v", err))
			host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "%s", "model": "Pro", "color": "#ff5555", "status": "error"}`, agentName, agentName))
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
			host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Agent thinking: %s", agentText))
		}

		if fc != nil {
			host.EmitLog(ctx, agentName, "info", fmt.Sprintf("Executing tool: %s", fc.Name))
			
			result, isComplete := c.handleToolCall(ctx, host, fc.Name, fc.Args, projectDir, sandboxEnv, &proposedChanges)
			
			if isComplete {
				break
			}

			// Send the tool response back to the LLM
			tr := toolResponse{Result: result}
			trMap := map[string]interface{}{"result": tr.Result} // GenAI SDK expects map
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
			// If there's no function call, ask it to continue or complete
			_, err = chat.SendMessage(ctx, genai.Part{Text: "Please call a tool or TaskComplete."})
			if err != nil {
				break
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "%s", "model": "Pro", "color": "#50fa7b", "status": "success"}`, agentName, agentName))
	return proposedChanges, nil
}
