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

var researchToolRegistry = []*genai.Tool{
	{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "RunCommand",
				Description: "Run a read-only terminal command in the project directory (e.g. ls, cat, grep, find).",
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

// ResearchAgent implements the Agent interface for exploring the codebase and answering queries.
type ResearchAgent struct {
	logger *slog.Logger
	client *genai.Client
}

// NewResearchAgent creates a new ResearchAgent.
func NewResearchAgent(logger *slog.Logger, client *genai.Client) *ResearchAgent {
	return &ResearchAgent{
		logger: logger,
		client: client,
	}
}

func (r *ResearchAgent) handleToolCall(ctx context.Context, host AgentHost, toolName string, args map[string]interface{}, projectDir string, sandboxEnv *sandbox.MCPContainer) (string, bool) {
	var result string
	isComplete := false
	agentName := "research_agent"

	if toolName == "TaskComplete" {
		host.EmitLog(ctx, agentName, "info", "Research TaskComplete requested.")
		isComplete = true
	} else if toolName == "RunCommand" {
		cmdStr, _ := args["command"].(string)
		host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Running read-only command in sandbox: %s", cmdStr))

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
		host.EmitLog(ctx, agentName, "info", "Sent research reply to user.")
		result = "Reply sent."
	}

	return result, isComplete
}

// Execute implements the Agent interface.
func (r *ResearchAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	r.logger.Info("ResearchAgent executing", slog.String("input", input))
	agentName := "research_agent"
	projectDir := host.GetProjectDir()

	if r.client == nil {
		return "", fmt.Errorf("ResearchAgent requires a valid genai.Client")
	}

	host.EmitLog(ctx, agentName, "info", fmt.Sprintf("Research Agent started task: %s", input))
	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "Researcher", "model": "Pro", "color": "#8be9fd", "status": "running"}`, agentName))

	config := &genai.GenerateContentConfig{
		Tools: researchToolRegistry,
	}

	sysPrompt := fmt.Sprintf(`You are an expert Research agent named %s.
Your task: %s
Working directory: %s

You must use the provided tools to explore the codebase, read files, and answer the user's architectural questions.
You are read-only. Do NOT attempt to modify files. 
Always verify your findings. When done, call SendReply to give a detailed answer, then call TaskComplete.`, agentName, input, projectDir)

	config.SystemInstruction = &genai.Content{
		Role: "system",
		Parts: []*genai.Part{
			{Text: sysPrompt},
		},
	}

	chat, err := r.client.Chats.Create(ctx, "gemini-2.5-pro", config, nil)
	if err != nil {
		host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Failed to create chat: %v", err))
		return "", err
	}

	sandboxEnv := sandbox.NewMCPContainer(r.logger, sandbox.RuntimeHost, projectDir)
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
				sandboxState := fmt.Sprintf(`{"id": "sandbox-%s", "name": "mcp-research-env", "status": "running", "cpuPercent": %.1f, "memoryPercent": %.1f, "memoryMb": %d, "memoryLimitMb": %d, "uptimeSeconds": %d}`,
					agentName, cpu, float64(memMb)/1024.0*100, memMb, 1024, uptime)
				host.EmitState(ctx, "sandbox_state", "sandbox-"+agentName, sandboxState)
			}
		}
	}()

	// Action Loop
	for i := 0; i < 150; i++ { // max 150 steps
		resp, err := chat.SendMessage(ctx, genai.Part{Text: "Proceed with your research task. Remember to use tools. If done, call TaskComplete."})
		if err != nil {
			host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Research Chat error: %v", err))
			host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "Researcher", "model": "Pro", "color": "#ff5555", "status": "error"}`, agentName))
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
			host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Research Agent thinking: %s", agentText))
		}

		if fc != nil {
			host.EmitLog(ctx, agentName, "info", fmt.Sprintf("Research Agent Executing tool: %s", fc.Name))
			
			result, isComplete := r.handleToolCall(ctx, host, fc.Name, fc.Args, projectDir, sandboxEnv)
			
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

	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "Researcher", "model": "Pro", "color": "#50fa7b", "status": "success"}`, agentName))
	
	// Research Tester does not propose direct code changes.
	return "", nil
}
