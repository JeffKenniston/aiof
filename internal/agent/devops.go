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

// DevOpsAgent implements the Agent interface for infrastructure, pipelines, and Docker/k8s configurations.
type DevOpsAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewDevOpsAgent(logger *slog.Logger, client *genai.Client) *DevOpsAgent {
	return &DevOpsAgent{logger: logger, client: client}
}

func (d *DevOpsAgent) handleToolCall(ctx context.Context, host AgentHost, toolName string, args map[string]interface{}, projectDir string, sandboxEnv *sandbox.MCPContainer, proposedChanges *string) (string, bool) {
	var result string
	isComplete := false
	agentName := "devops_agent"

	if toolName == "TaskComplete" {
		host.EmitLog(ctx, agentName, "info", "DevOps TaskComplete requested.")
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
		host.EmitLog(ctx, agentName, "debug", fmt.Sprintf("Running DevOps command: %s", cmdStr))
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
		result = "Reply sent."
	}
	return result, isComplete
}

func (d *DevOpsAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	d.logger.Info("DevOpsAgent executing", slog.String("input", input))
	agentName := "devops_agent"
	projectDir := host.GetProjectDir()

	if d.client == nil {
		return "", fmt.Errorf("DevOpsAgent requires a valid genai.Client")
	}

	host.EmitLog(ctx, agentName, "info", fmt.Sprintf("DevOps Agent started task: %s", input))
	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "DevOps", "model": "Pro", "color": "#f1fa8c", "status": "running"}`, agentName))

	config := &genai.GenerateContentConfig{Tools: toolRegistry}
	sysPrompt := fmt.Sprintf(`You are an expert DevOps and Infrastructure agent named %s.
Your task: %s
Working directory: %s

You must focus exclusively on Dockerfiles, Kubernetes manifests, CI/CD pipelines, and Terraform IaC configurations.
Ensure all deployments follow cloud-native best practices.
Always verify your steps. When done, call TaskComplete.`, agentName, input, projectDir)

	config.SystemInstruction = &genai.Content{
		Role: "system",
		Parts: []*genai.Part{{Text: sysPrompt}},
	}

	chat, err := d.client.Chats.Create(ctx, "gemini-2.5-pro", config, nil)
	if err != nil {
		host.EmitLog(ctx, agentName, "error", fmt.Sprintf("Failed to create chat: %v", err))
		return "", err
	}

	var proposedChanges string
	sandboxEnv := sandbox.NewMCPContainer(d.logger, sandbox.RuntimeHost, projectDir)
	if err := sandboxEnv.Start(ctx); err != nil {
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
				host.EmitState(ctx, "sandbox_state", "sandbox-"+agentName, fmt.Sprintf(`{"id": "sandbox-%s", "name": "mcp-devops-env", "status": "running", "cpuPercent": %.1f, "memoryPercent": %.1f, "memoryMb": %d, "memoryLimitMb": %d, "uptimeSeconds": %d}`, agentName, cpu, float64(memMb)/1024.0*100, memMb, 1024, uptime))
			}
		}
	}()

	for i := 0; i < 150; i++ {
		resp, err := chat.SendMessage(ctx, genai.Part{Text: "Proceed with your infrastructure task. Remember to use tools. If done, call TaskComplete."})
		if err != nil {
			host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "DevOps", "model": "Pro", "color": "#ff5555", "status": "error"}`, agentName))
			return proposedChanges, err
		}
		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil { break }
		var fc *genai.FunctionCall
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.FunctionCall != nil { fc = part.FunctionCall }
		}
		if fc != nil {
			result, isComplete := d.handleToolCall(ctx, host, fc.Name, fc.Args, projectDir, sandboxEnv, &proposedChanges)
			if isComplete { break }
			trMap := map[string]interface{}{"result": result}
			_, _ = chat.SendMessage(ctx, genai.Part{FunctionResponse: &genai.FunctionResponse{Name: fc.Name, Response: trMap}})
		}
		time.Sleep(500 * time.Millisecond)
	}

	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "DevOps", "model": "Pro", "color": "#50fa7b", "status": "success"}`, agentName))
	return proposedChanges, nil
}
