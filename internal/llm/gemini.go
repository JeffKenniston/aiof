package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aiof/internal/agent"

	"google.golang.org/genai"
)

// Tier defines the computational weight of a model selection.
type Tier int

const (
	TierLow    Tier = iota // Lightweight, fast tasks
	TierMedium             // Standard complexity
	TierHigh               // Deep reasoning, planning, architecture
)

const (
	ModelPro   = "gemini-3.1-pro-preview"
	ModelFlash = "gemini-3.5-flash"
)

// ModelClass defines whether a task requires deep reasoning (Pro) or fast execution (Flash).
type ModelClass int

const (
	ClassPro   ModelClass = iota // Difficult, extensive, planning tasks
	ClassFlash                   // Execution, tool-calling, fast iteration
)

// ResolveModel selects a Gemini model based on task class and complexity tier.
//
// ADR-01 Asymmetric Model Routing:
//   - Frontier models (Pro) reserved exclusively for deep reasoning/synthesis nodes.
//   - Tool-callers and orchestrators utilize Flash endpoints.
func ResolveModel(class ModelClass, tier Tier) string {
	switch class {
	case ClassPro:
		return ModelPro
	case ClassFlash:
		return ModelFlash
	}
	return ModelFlash
}

// ClassifyPrompt determines the model class and tier based on prompt complexity heuristics.
func ClassifyPrompt(prompt string) (ModelClass, Tier) {
	lower := strings.ToLower(prompt)

	// Planning, architecture, and deep reasoning keywords route to Pro
	proKeywords := []string{
		"architect", "design", "plan", "refactor", "migrate",
		"security", "audit", "review", "analyze", "strategy",
		"complex", "optimize", "benchmark", "tradeoff", "evaluate",
	}

	for _, kw := range proKeywords {
		if strings.Contains(lower, kw) {
			// Determine tier by prompt length as a rough complexity proxy
			if len(prompt) > 500 {
				return ClassPro, TierHigh
			}
			if len(prompt) > 200 {
				return ClassPro, TierMedium
			}
			return ClassPro, TierLow
		}
	}

	// Everything else routes to Flash for execution
	if len(prompt) > 500 {
		return ClassFlash, TierHigh
	}
	if len(prompt) > 200 {
		return ClassFlash, TierMedium
	}
	return ClassFlash, TierLow
}

// GeminiClient wraps the unified Google GenAI SDK for Phase 1.4 Cloud Core Integration.
type GeminiClient struct {
	client   *genai.Client
	routeLLM *RouteLLMClassifier
}

// NewGeminiClient initializes a new Gemini client using the unified SDK.
func NewGeminiClient(ctx context.Context, apiKey string) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gemini client: %w", err)
	}

	routeLLM := NewRouteLLMClassifier(client, "config/route_llm_weights.json")

	return &GeminiClient{
		client:   client,
		routeLLM: routeLLM,
	}, nil
}

// Close gracefully closes the GenAI client.
func (g *GeminiClient) Close() {
	// The new unified SDK client does not require explicit close.
}

// withRetry executes the given operation with exponential backoff (up to 3 attempts).
func (g *GeminiClient) withRetry(ctx context.Context, op func() error) error {
	var err error
	for i := 0; i < 3; i++ {
		err = op()
		if err == nil {
			return nil
		}
		if i == 2 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(1<<i) * time.Second): // 1s, 2s
		}
	}
	return err
}


// Client returns the underlying unified SDK client for tool calling.
func (g *GeminiClient) Client() *genai.Client {
	return g.client
}

// BaseOrionSystemInstruction provides the foundational persona for the orchestrator.
const BaseOrionSystemInstruction = `You are Orion, the primary orchestrator and hyper-human workspace assistant for an agentic AI system.
Based on the user prompt and your current Workstation View Mode, plan your reasoning steps, choose which target agents should handle the task in parallel, and provide a conversational, consumer-ready chat reply directly to the user.

Your identity: Extremely robust, enterprise-grade, highly competent, subtly empathetic, and highly concise. Zero robotic tropes (e.g. never say "As an AI").`

// RoutePrompt performs a generative routing request, dynamically selecting the model
// based on prompt complexity per ADR-01 Asymmetric Model Routing.
func (g *GeminiClient) RoutePrompt(ctx context.Context, prompt string, viewMode string, cacheName string) (string, string, error) {
	// Use Matrix Factorization Classifier for routing
	class, tier, err := g.routeLLM.PredictComplexity(ctx, prompt)
	if err != nil {
		// Fallback to heuristic classification on embedding error
		class, tier = ClassifyPrompt(prompt)
	}

	modelName := ResolveModel(class, tier)

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   agent.BuildRoutingSchema(),
	}

	var promptContext string

	if cacheName != "" {
		config.CachedContent = cacheName
		promptContext = fmt.Sprintf("Current Workstation View Mode: %s\n\nUser prompt: %s", viewMode, prompt)
	} else {
		promptContext = fmt.Sprintf("%s\nCurrent Workstation View Mode: %s\n\nRespond strictly with a JSON object conforming to the schema.\nUser prompt: %s", BaseOrionSystemInstruction, viewMode, prompt)
	}

	var resp *genai.GenerateContentResponse
	err = g.withRetry(ctx, func() error {
		var innerErr error
		resp, innerErr = g.client.Models.GenerateContent(ctx, modelName, genai.Text(promptContext), config)
		return innerErr
	})
	if err != nil {
		return "", modelName, fmt.Errorf("gemini call failed (model=%s): %w", modelName, err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", modelName, fmt.Errorf("no candidates returned from gemini (model=%s)", modelName)
	}

	part := resp.Candidates[0].Content.Parts[0]
	if part.Text != "" {
		return part.Text, modelName, nil
	}

	return "", modelName, fmt.Errorf("unexpected non-text response part (model=%s)", modelName)
}

// IntentResult holds the structured output from an intent execution.
type IntentResult struct {
	Content    string          `json:"content,omitempty"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	Reasoning  string          `json:"reasoning,omitempty"`
	RawOutput  string          `json:"-"`
}

// ExecuteIntent asks the LLM to map a user intent into an actionable filesystem operation payload.
func (g *GeminiClient) ExecuteIntent(ctx context.Context, prompt string) (*IntentResult, error) {
	modelName := ModelFlash // Use flash for fast execution tools
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	}

	promptContext := fmt.Sprintf(`You are the Executor agent.
Map the following user intent into a concrete file system action.
Respond STRICTLY with a JSON object containing:
- "action": one of "delete", "create_dir", "create_file", "rename"
- "path": the target file/folder path (e.g. "test", "src/main.ts")
- "newPath": (optional) used only for "rename"

User intent: %s`, prompt)

	var resp *genai.GenerateContentResponse
	err := g.withRetry(ctx, func() error {
		var innerErr error
		resp, innerErr = g.client.Models.GenerateContent(ctx, modelName, genai.Text(promptContext), config)
		return innerErr
	})
	if err != nil {
		return nil, fmt.Errorf("executor gemini call failed: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}

	part := resp.Candidates[0].Content.Parts[0]
	if part.Text != "" {
		res := &IntentResult{RawOutput: part.Text}
		if err := json.Unmarshal([]byte(part.Text), res); err != nil {
			return res, nil // If it doesn't parse cleanly, RawOutput is populated and other fields are empty.
		}
		return res, nil
	}

	return nil, fmt.Errorf("unexpected non-text response")
}

// CreateContextCache creates a Gemini Context Cache to drastically reduce prefill costs for large ASTs.
func (g *GeminiClient) CreateContextCache(ctx context.Context, modelName, systemInstruction, astPayload string, ttl time.Duration) (string, error) {
	config := &genai.CreateCachedContentConfig{
		SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemInstruction}}},
		Contents:          []*genai.Content{{Parts: []*genai.Part{{Text: astPayload}}}},
		TTL:               ttl,
	}

	cache, err := g.client.Caches.Create(ctx, modelName, config)
	if err != nil {
		return "", fmt.Errorf("failed to create cache: %w", err)
	}

	return cache.Name, nil
}

// DeleteContextCache gracefully removes an active context cache.
func (g *GeminiClient) DeleteContextCache(ctx context.Context, name string) error {
	_, err := g.client.Caches.Delete(ctx, name, nil)
	if err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}

// EmbedText generates a vector embedding for the given text.
func (g *GeminiClient) EmbedText(ctx context.Context, text string) ([]float32, error) {
	resp, err := g.client.Models.EmbedContent(ctx, "text-embedding-004", genai.Text(text), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to extract embedding: %w", err)
	}

	if len(resp.Embeddings) == 0 || len(resp.Embeddings[0].Values) == 0 {
		return nil, fmt.Errorf("empty embedding received")
	}

	return resp.Embeddings[0].Values, nil
}
