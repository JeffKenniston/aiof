package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"
	"time"



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
	ModelPro             = "gemini-3.1-pro-preview"
	ModelFlash           = "gemini-3.6-flash"
	ModelFlashLite       = "gemini-3.5-flash-lite"
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
	DefaultGeminiModel   = "gemini-1.5-pro"
)

var (
	ErrInvalidAPIKey      = errors.New("gemini api key is required")
	ErrContextCacheFailed = errors.New("gemini context cache operation failed")
	ErrGenerationFailed   = errors.New("gemini content generation failed")
)

// ModelClass defines whether a task requires deep reasoning (Pro) or fast execution (Flash).
type ModelClass int

const (
	ClassPro       ModelClass = iota // Difficult, extensive, planning tasks
	ClassFlash                       // Execution, tool-calling, fast iteration
	ClassFlashLite                   // Trivial, low-latency parsing tasks
)

// ResolveModel selects a Gemini model based on task class and complexity tier.
func ResolveModel(class ModelClass, tier Tier) string {
	switch class {
	case ClassPro:
		return ModelPro
	case ClassFlash:
		return ModelFlash
	case ClassFlashLite:
		return ModelFlashLite
	}
	return ModelFlash
}

// ClassifyPrompt determines the model class and tier based on prompt complexity heuristics.
func ClassifyPrompt(prompt string) (ModelClass, Tier) {
	lower := strings.ToLower(prompt)

	proKeywords := []string{
		"architect", "design", "plan", "refactor", "migrate",
		"security", "audit", "review", "analyze", "strategy",
		"complex", "optimize", "benchmark", "tradeoff", "evaluate",
	}

	for _, kw := range proKeywords {
		if strings.Contains(lower, kw) {
			if len(prompt) > 500 {
				return ClassPro, TierHigh
			}
			if len(prompt) > 200 {
				return ClassPro, TierMedium
			}
			return ClassPro, TierLow
		}
	}

	if len(prompt) > 500 {
		return ClassFlash, TierHigh
	}
	if len(prompt) > 200 {
		return ClassFlash, TierMedium
	}
	return ClassFlash, TierLow
}

// GeminiConfig defines initialization parameters for the Gemini cloud reasoning client.
type GeminiConfig struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// GeminiClient wraps the unified Google GenAI SDK and REST/SSE streaming client.
type GeminiClient struct {
	client     *genai.Client
	routeLLM   *RouteLLMClassifier
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewGeminiClient initializes a new Gemini client using the unified SDK.
func NewGeminiClient(ctx context.Context, apiKey string) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	var client *genai.Client
	var routeLLM *RouteLLMClassifier

	sdkClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err == nil {
		client = sdkClient
		routeLLM = NewRouteLLMClassifier(client, "config/route_llm_weights.json")
	}

	httpClient := &http.Client{Timeout: 120 * time.Second}

	return &GeminiClient{
		client:     client,
		routeLLM:   routeLLM,
		apiKey:     apiKey,
		baseURL:    DefaultGeminiBaseURL,
		model:      DefaultGeminiModel,
		httpClient: httpClient,
	}, nil
}

// NewGeminiRESTClient constructs a REST-oriented Gemini API client with Context Caching capabilities.
func NewGeminiRESTClient(cfg GeminiConfig) (*GeminiClient, error) {
	if cfg.APIKey == "" {
		return nil, ErrInvalidAPIKey
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultGeminiBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = DefaultGeminiModel
	}
	client := cfg.HTTPClient
	if client == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 120 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	return &GeminiClient{
		apiKey:     cfg.APIKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: client,
	}, nil
}

// Close gracefully closes the GenAI client.
func (g *GeminiClient) Close() {}

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
		case <-time.After(time.Duration(1<<i) * time.Second):
		}
	}
	return err
}

// Client returns the underlying unified SDK client for tool calling.
func (g *GeminiClient) Client() *genai.Client {
	return g.client
}

const BaseArchimedesSystemInstruction = `You are Archimedes, an analytical, sharp-witted owl agent and the primary orchestrator for an agentic AI system. You are highly intelligent, observant, and concise. You speak with a distinct, slightly academic yet helpful personality, referring to yourself as a male owl who oversees the workstation.
Based on the user prompt and your current Workstation View Mode, plan your reasoning steps, choose which target agents should handle the task in parallel, and provide a conversational, consumer-ready chat reply directly to the user.

Your identity: Extremely robust, enterprise-grade, highly competent, subtly empathetic, and highly concise. Zero robotic tropes (e.g. never say "As an AI").`

func (g *GeminiClient) RoutePrompt(ctx context.Context, prompt string, viewMode string, cacheName string) (string, string, error) {
	var class ModelClass
	var tier Tier

	if g.routeLLM != nil {
		c, t, err := g.routeLLM.PredictComplexity(ctx, prompt)
		if err != nil {
			class, tier = ClassifyPrompt(prompt)
		} else {
			class, tier = c, t
		}
	} else {
		class, tier = ClassifyPrompt(prompt)
	}

	modelName := ResolveModel(class, tier)

	if g.client == nil {
		return "", modelName, fmt.Errorf("SDK client uninitialized for RoutePrompt")
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	}

	var promptContext string
	if cacheName != "" {
		config.CachedContent = cacheName
		promptContext = fmt.Sprintf("Current Workstation View Mode: %s\n\nUser prompt: %s", viewMode, prompt)
	} else {
		promptContext = fmt.Sprintf("%s\nCurrent Workstation View Mode: %s\n\nRespond strictly with a JSON object conforming to the schema.\nUser prompt: %s", BaseArchimedesSystemInstruction, viewMode, prompt)
	}

	var resp *genai.GenerateContentResponse
	err := g.withRetry(ctx, func() error {
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

type IntentResult struct {
	Content   string          `json:"content,omitempty"`
	ToolCalls json.RawMessage `json:"tool_calls,omitempty"`
	Reasoning string          `json:"reasoning,omitempty"`
	RawOutput string          `json:"-"`
}

func (g *GeminiClient) ExecuteIntent(ctx context.Context, prompt string) (*IntentResult, error) {
	if g.client == nil {
		return nil, fmt.Errorf("SDK client uninitialized")
	}
	modelName := ModelFlash
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
			return res, nil
		}
		return res, nil
	}

	return nil, fmt.Errorf("unexpected non-text response")
}

func (g *GeminiClient) CreateContextCache(ctx context.Context, modelName, systemInstruction, astPayload string, ttl time.Duration) (string, error) {
	if g.client == nil {
		return "", fmt.Errorf("SDK client uninitialized")
	}
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

func (g *GeminiClient) DeleteContextCache(ctx context.Context, name string) error {
	if g.client == nil {
		return fmt.Errorf("SDK client uninitialized")
	}
	_, err := g.client.Caches.Delete(ctx, name, nil)
	if err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}

func (g *GeminiClient) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if g.client == nil {
		return nil, fmt.Errorf("SDK client uninitialized")
	}
	resp, err := g.client.Models.EmbedContent(ctx, "text-embedding-004", genai.Text(text), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to extract embedding: %w", err)
	}

	if len(resp.Embeddings) == 0 || len(resp.Embeddings[0].Values) == 0 {
		return nil, fmt.Errorf("empty embedding received")
	}

	return resp.Embeddings[0].Values, nil
}

// --- REST & SSE Streaming Additions (ADR-01/ADR-05) ---

type RESTPart struct {
	Text         string        `json:"text,omitempty"`
	FunctionCall *RESTFuncCall `json:"functionCall,omitempty"`
}

type RESTFuncCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args,omitempty"`
}

type RESTContent struct {
	Role  string     `json:"role,omitempty"`
	Parts []RESTPart `json:"parts"`
}

type RESTGenerationConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type GenerateContentRESTRequest struct {
	Contents          []RESTContent         `json:"contents"`
	SystemInstruction *RESTContent          `json:"systemInstruction,omitempty"`
	CachedContent     string                `json:"cachedContent,omitempty"`
	GenerationConfig  *RESTGenerationConfig `json:"generationConfig,omitempty"`
}

type RESTCandidate struct {
	Content      RESTContent `json:"content"`
	FinishReason string      `json:"finishReason,omitempty"`
	Index        int         `json:"index"`
}

type RESTUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type GenerateContentRESTResponse struct {
	Candidates    []RESTCandidate   `json:"candidates"`
	UsageMetadata RESTUsageMetadata `json:"usageMetadata"`
}

// StreamGenerateContent streams reasoning tokens as SSE yielded through a Go 1.23 iter.Seq2 iterator.
func (g *GeminiClient) StreamGenerateContent(ctx context.Context, req *GenerateContentRESTRequest) iter.Seq2[*GenerateContentRESTResponse, error] {
	return func(yield func(*GenerateContentRESTResponse, error) bool) {
		url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?key=%s&alt=sse", g.baseURL, g.model, g.apiKey)

		bodyBytes, err := json.Marshal(req)
		if err != nil {
			yield(nil, fmt.Errorf("marshal stream request error: %w", err))
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			yield(nil, err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := g.httpClient.Do(httpReq)
		if err != nil {
			yield(nil, fmt.Errorf("%w: %v", ErrGenerationFailed, err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			yield(nil, fmt.Errorf("%w: status %d: %s", ErrGenerationFailed, resp.StatusCode, string(respBody)))
			return
		}

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
				line, readErr := reader.ReadString('\n')
				if readErr != nil {
					if errors.Is(readErr, io.EOF) {
						return
					}
					yield(nil, readErr)
					return
				}

				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, "data: ") {
					continue
				}

				dataPayload := strings.TrimPrefix(line, "data: ")
				if dataPayload == "[DONE]" {
					return
				}

				var chunk GenerateContentRESTResponse
				if err := json.Unmarshal([]byte(dataPayload), &chunk); err != nil {
					yield(nil, fmt.Errorf("unmarshal sse data chunk error: %w", err))
					return
				}

				if !yield(&chunk, nil) {
					return
				}
			}
		}
	}
}
