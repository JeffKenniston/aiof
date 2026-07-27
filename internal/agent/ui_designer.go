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

// UIDesignerAgent implements the Agent interface for UX/UI frontend styling.
type UIDesignerAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewUIDesignerAgent(logger *slog.Logger, client *genai.Client) *UIDesignerAgent {
	return &UIDesignerAgent{logger: logger, client: client}
}

func (u *UIDesignerAgent) handleToolCall(ctx context.Context, host AgentHost, toolName string, args map[string]interface{}, projectDir string, sandboxEnv *sandbox.MCPContainer, proposedChanges *string) (string, bool) {
	var result string
	isComplete := false
	agentName := "ui_designer_agent"

	if toolName == "TaskComplete" {
		host.EmitLog(ctx, agentName, "info", "UI Designer TaskComplete requested.")
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

func (u *UIDesignerAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	u.logger.Info("UIDesignerAgent executing", slog.String("input", input))
	agentName := "ui_designer_agent"
	projectDir := host.GetProjectDir()

	if u.client == nil { return "", fmt.Errorf("UIDesignerAgent requires genai.Client") }

	host.EmitLog(ctx, agentName, "info", fmt.Sprintf("UI Designer Agent started task: %s", input))
	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "UI Designer", "model": "Pro", "color": "#ff79c6", "status": "running"}`, agentName))

	config := &genai.GenerateContentConfig{Tools: toolRegistry}
	sysPrompt := fmt.Sprintf(`%s.
Your task: %s
Working directory: %s

You must focus exclusively on styling, Tailwind CSS, accessibility (a11y), Framer Motion, and aesthetic React components.
Leave complex state management to the Code Developer.
Always verify your steps. When done, call TaskComplete.`, GetPersonaPrompt(agentName, "specialist"), input, projectDir)
	config.SystemInstruction = &genai.Content{Role: "system", Parts: []*genai.Part{{Text: sysPrompt}}}

	chat, err := u.client.Chats.Create(ctx, "gemini-3.1-pro-preview", config, nil)
	if err != nil { return "", err }

	var proposedChanges string
	sandboxEnv := sandbox.NewMCPContainer(u.logger, sandbox.RuntimeHost, projectDir)
	if err := sandboxEnv.Start(ctx); err != nil { return "", err }
	defer sandboxEnv.Stop()

	for i := 0; i < 150; i++ {
		resp, err := chat.SendMessage(ctx, genai.Part{Text: "Proceed with your UI styling task. Remember to use tools. If done, call TaskComplete."})
		if err != nil {
			host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "UI Designer", "status": "error"}`, agentName))
			return proposedChanges, err
		}
		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil { break }
		var fc *genai.FunctionCall
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.FunctionCall != nil { fc = part.FunctionCall }
		}
		if fc != nil {
			result, isComplete := u.handleToolCall(ctx, host, fc.Name, fc.Args, projectDir, sandboxEnv, &proposedChanges)
			if isComplete { break }
			trMap := map[string]interface{}{"result": result}
			_, _ = chat.SendMessage(ctx, genai.Part{FunctionResponse: &genai.FunctionResponse{Name: fc.Name, Response: trMap}})
		}
		time.Sleep(500 * time.Millisecond)
	}

	host.EmitState(ctx, "agent_graph_state", agentName, fmt.Sprintf(`{"id": "%s", "label": "UI Designer", "model": "Pro", "color": "#50fa7b", "status": "success"}`, agentName))
	return proposedChanges, nil
}
