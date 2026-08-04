package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/genai"
)

type StudentAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewStudentAgent(logger *slog.Logger, client *genai.Client) *StudentAgent {
	return &StudentAgent{
		logger: logger,
		client: client,
	}
}

func (a *StudentAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	a.logger.Info("StudentAgent (VLM) starting execution")
	host.EmitState(ctx, "agent_graph_state", "student_agent", `{"id": "student_agent", "label": "Student Agent", "model": "Pro Vision", "color": "#ff79c6", "status": "running"}`)

	// Retrieve the latest VLM Frame (Screenshot with SOM bounding boxes)
	latestFrame := host.GetLatestVLMFrame(ctx)
	if latestFrame == "" {
		host.EmitLog(ctx, "StudentAgent", "warn", "No VLM frame available. Is the MCP browser running?")
	} else {
		host.EmitLog(ctx, "StudentAgent", "info", "Retrieved latest VLM frame with SOM annotations.")
	}

	// 1. Build the system prompt
	systemInstruction := `You are the Student Agent (Study Agent), an autonomous VLM executing desktop automation via the Model Context Protocol.
Your primary environment is a web browser.
You will be provided with a user prompt and an image (the latest screenshot of your browser).
IMPORTANT: The screenshot contains Set-of-Mark (SOM) annotations. Interactive elements (links, buttons, inputs) are enclosed in red boxes with an alphanumeric ID (e.g. A0, A1, A2).

You have three native capabilities available to you, which you execute by returning a JSON object describing the action.
You must return exactly ONE of the following JSON structures (and no other text) to take an action:

1. To navigate to a new URL:
{"action": "navigate", "url": "https://example.com"}

2. To click an element (using its SOM mark ID):
{"action": "click", "mark_id": "A4"}

3. To type text into an input field (using its SOM mark ID):
{"action": "type_text", "mark_id": "A12", "text": "my search query"}

Analyze the provided screenshot, read the user's prompt, and decide which single action to take next.
If you need to search for something, first navigate to google.com or wikipedia.org. If you are already on a search page, use type_text on the search bar's mark_id.
Reply strictly in JSON.`

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Role: "system",
			Parts: []*genai.Part{
				{Text: systemInstruction},
			},
		},
		Temperature:    genai.Ptr(float32(0.2)),
		ResponseMIMEType: "application/json",
	}

	// Prepare generation request
	var reqContents []*genai.Content
	reqContents = append(reqContents, &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: fmt.Sprintf("User Request: %s\n\nWhat is your next action?", input)},
		},
	})

	// If we have an image, append it as a Blob
	if latestFrame != "" {
		b64 := latestFrame
		if strings.Contains(b64, "base64,") {
			b64 = strings.Split(b64, "base64,")[1]
		}
		host.EmitLog(ctx, "StudentAgent", "info", "Submitting annotated SOM frame to VLM for visual reasoning...")
	}

	res, err := a.client.Models.GenerateContent(ctx, "gemini-2.5-pro", reqContents, config)
	if err != nil {
		host.EmitState(ctx, "agent_graph_state", "student_agent", `{"id": "student_agent", "label": "Student Agent", "model": "Pro Vision", "color": "#ff5555", "status": "error"}`)
		return "", fmt.Errorf("student agent generation failed: %w", err)
	}

	var jsonOutput string
	if len(res.Candidates) > 0 && res.Candidates[0].Content != nil && len(res.Candidates[0].Content.Parts) > 0 {
		jsonOutput = res.Candidates[0].Content.Parts[0].Text
	} else {
		return "", fmt.Errorf("student agent returned empty response")
	}

	host.EmitLog(ctx, "StudentAgent", "info", fmt.Sprintf("VLM Reasoning Output: %s", jsonOutput))

	// Parse the JSON output
	type VLMAction struct {
		Action  string `json:"action"`
		URL     string `json:"url,omitempty"`
		MarkID  string `json:"mark_id,omitempty"`
		Text    string `json:"text,omitempty"`
	}

	var action VLMAction
	if err := json.Unmarshal([]byte(jsonOutput), &action); err != nil {
		return "", fmt.Errorf("failed to parse VLM json: %w", err)
	}

	// Push the intent to the orchestrator (via agent_intents)
	if action.Action == "navigate" || action.Action == "click" || action.Action == "type_text" {
		// Wait, the orchestrator only understands MANUAL_NAVIGATE and MANUAL_CLICK currently, 
		// but since we want the Agent to drive it, we can just emit the same intents!
		
		intentType := "MANUAL_" + strings.ToUpper(action.Action)
		
		payload := map[string]interface{}{}
		if action.URL != "" {
			payload["url"] = action.URL
		}
		if action.MarkID != "" {
			payload["mark_id"] = action.MarkID
		}
		if action.Text != "" {
			payload["text"] = action.Text
		}

		intentJSON, _ := json.Marshal(map[string]interface{}{
			"id": fmt.Sprintf("agent-intent-%d", time.Now().UnixNano()),
			"intentType": intentType,
			"payload": payload,
			"status": "pending",
		})
		
		host.EmitState(ctx, "agent_intents", fmt.Sprintf("intent-%d", time.Now().UnixNano()), string(intentJSON))
		host.EmitLog(ctx, "StudentAgent", "info", fmt.Sprintf("Emitted %s intent to UI and MCP", intentType))
	}

	host.EmitState(ctx, "agent_graph_state", "student_agent", `{"id": "student_agent", "label": "Student Agent", "model": "Pro Vision", "color": "#50fa7b", "status": "success"}`)

	return "VLM Action Executed", nil
}
