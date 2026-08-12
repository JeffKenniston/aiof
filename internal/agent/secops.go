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

// SecOpsAgent implements the Agent interface for security vulnerability scanning and hardening.
type SecOpsAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewSecOpsAgent(logger *slog.Logger, client *genai.Client) *SecOpsAgent {
	return &SecOpsAgent{logger: logger, client: client}
}

func (s *SecOpsAgent) handleToolCall(ctx context.Context, host AgentHost, toolName string, args map[string]interface{}, projectDir string, sandboxEnv *sandbox.MCPContainer, proposedChanges *string) (string, bool) {
	var result string
	isComplete := false
	agentName := "secops_agent"

	if toolName == "TaskComplete" {
		host.EmitLog(ctx, agentName, "info", "SecOps TaskComplete requested.")
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
	} else if toolName == "RunCommand" {
		cmdStr, _ := args["command"].(string)
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
		if len(result) > 2000 { result = result[:2000] + "\n...[truncated]" }
	} else if toolName == "ReadFile" {
		path, _ := args["path"].(string)
		fileContent, err := os.ReadFile(filepath.Join(projectDir, path))
		if err != nil {
			result = fmt.Sprintf("Error reading file: %v", err)
		} else {
			result = string(fileContent)
			if len(result) > 4000 { result = result[:4000] + "\n...[truncated]" }
		}
	} else if toolName == "SendReply" {
		msg, _ := args["message"].(string)
		chatMsg := chatMessage{ID: fmt.Sprintf("reply-%d", time.Now().UnixNano()), Role: "agent", Content: msg, AgentName: agentName, Timestamp: time.Now().UnixMilli()}
		chatPayload, _ := json.Marshal(chatMsg)
		host.EmitState(ctx, "chat_messages", chatMsg.ID, string(chatPayload))
		result = "Reply sent."
	}
	return result, isComplete
}

func (s *SecOpsAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	s.logger.Info("SecOpsAgent executing", slog.String("input", input))
	agentName := "secops_agent"
	projectDir := host.GetProjectDir()

	if s.client == nil { return "", fmt.Errorf("SecOpsAgent requires genai.Client") }

	host.EmitLog(ctx, agentName, "info", fmt.Sprintf("SecOps Agent started task: %s", input))
	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "SecOps", "model": "Pro", "color": "#ff5555", "status": "running"}`, agentName))

	config := &genai.GenerateContentConfig{Tools: toolRegistry}
	sysPrompt := fmt.Sprintf(`You are an expert Security Operations (SecOps) agent named %s.
Your task: %s
Working directory: %s

You must focus exclusively on finding and patching security vulnerabilities (e.g. OWASP top 10, SQLi, leaked secrets, weak permissions).
Always verify your steps. When done, call TaskComplete.`, agentName, input, projectDir)
	config.SystemInstruction = &genai.Content{Role: "system", Parts: []*genai.Part{{Text: sysPrompt}}}

	chat, err := s.client.Chats.Create(ctx, "gemini-2.5-pro", config, nil)
	if err != nil { return "", err }

	var proposedChanges string
	sandboxEnv := sandbox.NewMCPContainer(s.logger, sandbox.RuntimeHost, projectDir)
	if err := sandboxEnv.Start(ctx); err != nil { return "", err }
	defer sandboxEnv.Stop()

	for i := 0; i < 150; i++ {
		resp, err := chat.SendMessage(ctx, genai.Part{Text: "Proceed with your security audit and patching task. Remember to use tools. If done, call TaskComplete."})
		if err != nil {
			host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "SecOps", "status": "error"}`, agentName))
			return proposedChanges, err
		}
		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil { break }
		var fc *genai.FunctionCall
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.FunctionCall != nil { fc = part.FunctionCall }
		}
		if fc != nil {
			result, isComplete := s.handleToolCall(ctx, host, fc.Name, fc.Args, projectDir, sandboxEnv, &proposedChanges)
			if isComplete { break }
			trMap := map[string]interface{}{"result": result}
			_, _ = chat.SendMessage(ctx, genai.Part{FunctionResponse: &genai.FunctionResponse{Name: fc.Name, Response: trMap}})
		}
		time.Sleep(500 * time.Millisecond)
	}

	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "SecOps", "model": "Pro", "color": "#50fa7b", "status": "success"}`, agentName))
	return proposedChanges, nil
}
