package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// SGLangClient implements the decoupled bare-metal SGLang provider using RadixAttention.
// ADR-01 requires zero-copy memory transfers and shared prefix routing for agent graphs.
type SGLangClient struct {
	logger     *slog.Logger
	config     SGLangConfig
	httpClient *http.Client
}

type SGLangConfig struct {
	SocketPath          string
	EnablePrefillDecode bool
	MooncakeTransfer    bool
	LMCacheGlobalPool   string
	TurboQuantBits      int
}

// SGLangRequest matches the standard SGLang HTTP API payload structure.
type SGLangRequest struct {
	Text                 string         `json:"text"`
	SamplingParams       SamplingParams `json:"sampling_params"`
	Stream               bool           `json:"stream"`
	ReturnLogprob        bool           `json:"return_logprob"`
	LogprobStartLen      int            `json:"logprob_start_len"`
	TopLogprobsNum       int            `json:"top_logprobs_num"`
	ReturnTextInLogprobs bool           `json:"return_text_in_logprobs"`
	MooncakeRouting      bool           `json:"mooncake_routing,omitempty"`
	LMCacheEnabled       bool           `json:"lmcache_enabled,omitempty"`
	TurboQuantBits       int            `json:"turbo_quant_bits,omitempty"`
}

type SamplingParams struct {
	MaxNewTokens int      `json:"max_new_tokens"`
	Stop         []string `json:"stop,omitempty"`
	Temperature  float32  `json:"temperature"`
	TopP         float32  `json:"top_p"`
}

// SGLangResponse matches the generated output from the SGLang /generate endpoint.
type SGLangResponse struct {
	Text            string  `json:"text"`
	TurboQuantError float64 `json:"turbo_quant_error,omitempty"`
	// Radix cache stats would be attached here natively.
}

// NewSGLangClient initializes an enterprise-grade connection pool to the SGLang cluster using Unix Domain Sockets.
func NewSGLangClient(logger *slog.Logger, cfg SGLangConfig) *SGLangClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Bypass network stack, use Unix Domain Sockets
			return net.Dial("unix", cfg.SocketPath)
		},
	}

	return &SGLangClient{
		logger: logger,
		config: cfg,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// Generate routes the prompt to the local SGLang RadixAttention engine.
// Prefix caching is handled automatically at the VRAM layer by SGLang.
func (s *SGLangClient) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := SGLangRequest{
		Text: prompt,
		SamplingParams: SamplingParams{
			MaxNewTokens: 1024,
			Temperature:  0.0, // Greedy decoding for deterministic orchestration
			TopP:         0.95,
		},
		Stream:          false,
		MooncakeRouting: s.config.MooncakeTransfer,
		LMCacheEnabled:  s.config.LMCacheGlobalPool != "",
		TurboQuantBits:  s.config.TurboQuantBits,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to encode sglang request: %w", err)
	}

	// Address doesn't matter since the dialer forces the UDS socket path
	req, err := http.NewRequestWithContext(ctx, "POST", "http://unix/generate", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	s.logger.Debug("Routing inference to bare-metal SGLang engine", slog.Int("payload_bytes", len(payload)))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sglang execution failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("sglang: failed to read error response body: %w", err)
		}
		return "", fmt.Errorf("sglang returned non-200 status %d: %s", resp.StatusCode, string(body))
	}

	var sglResp SGLangResponse
	if err := json.NewDecoder(resp.Body).Decode(&sglResp); err != nil {
		return "", fmt.Errorf("failed to decode sglang response: %w", err)
	}

	// Phase 6: TurboQuant 1-bit QJL Error Checking
	if s.config.TurboQuantBits > 0 && sglResp.TurboQuantError > 0.05 {
		s.logger.Warn("TurboQuant QJL error exceeded threshold", slog.Float64("error", sglResp.TurboQuantError))
		return "", fmt.Errorf("sglang: quantization error exceeded threshold (%.4f)", sglResp.TurboQuantError)
	}

	return sglResp.Text, nil
}
