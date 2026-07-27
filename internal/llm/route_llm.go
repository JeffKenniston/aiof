package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"google.golang.org/genai"
)

// RouteLLMClassifier implements a Matrix Factorization (MF) routing strategy 
// as defined in ADR-01 to predict query complexity non-autoregressively.
type RouteLLMWeights struct {
	FC1Weight [][]float32 `json:"fc1.weight"`
	FC1Bias   []float32   `json:"fc1.bias"`
	FC2Weight [][]float32 `json:"fc2.weight"`
	FC2Bias   []float32   `json:"fc2.bias"`
}

type RouteLLMClassifier struct {
	client    *genai.Client
	threshold float32
	weights   *RouteLLMWeights
}

const defaultEmbeddingDim = 768
const defaultHiddenDim = 128

// NewRouteLLMClassifier initializes the MF router. If no weights file is found,
// it now crashes (panic) as mandated by enterprise-grade determinism requirements (Phase 1 hardening).
func NewRouteLLMClassifier(client *genai.Client, weightsPath string) *RouteLLMClassifier {
	classifier := &RouteLLMClassifier{
		client:    client,
		threshold: 0.5,
	}

	data, err := os.ReadFile(weightsPath)
	if err != nil {
		slog.Error("RouteLLM: failed to load weights file. Enterprise strict determinism requires a valid matrix.", "path", weightsPath, "error", err)
		panic(fmt.Sprintf("FATAL: RouteLLM weights file missing or inaccessible: %v", err))
	}
	
	var w RouteLLMWeights
	if err := json.Unmarshal(data, &w); err != nil {
		slog.Error("RouteLLM: failed to parse weights JSON.", "path", weightsPath, "error", err)
		panic(fmt.Sprintf("FATAL: RouteLLM weights file is invalid JSON: %v", err))
	}
	
	classifier.weights = &w
	return classifier
}

func (r *RouteLLMClassifier) PredictComplexity(ctx context.Context, prompt string) (ModelClass, Tier, error) {
	resp, err := r.client.Models.EmbedContent(ctx, "text-embedding-004", genai.Text(prompt), nil)
	if err != nil {
		return ClassFlash, TierLow, fmt.Errorf("failed to extract embedding for RouteLLM: %w", err)
	}

	if len(resp.Embeddings) == 0 || len(resp.Embeddings[0].Values) == 0 {
		return ClassFlash, TierLow, fmt.Errorf("empty embedding received")
	}

	emb := resp.Embeddings[0].Values
	if len(emb) != defaultEmbeddingDim {
		return ClassFlash, TierLow, fmt.Errorf("dimension mismatch: got %d, expected %d", len(emb), defaultEmbeddingDim)
	}

	// Forward pass through 2-layer MLP (FC1 -> ReLU -> FC2)
	w := r.weights
	hidden := make([]float32, len(w.FC1Weight))
	for i := 0; i < len(w.FC1Weight); i++ {
		var sum float32 = w.FC1Bias[i]
		for j := 0; j < len(emb); j++ {
			sum += emb[j] * w.FC1Weight[i][j]
		}
		// ReLU
		if sum < 0 {
			sum = 0
		}
		hidden[i] = sum
	}

	var score float32 = w.FC2Bias[0]
	for j := 0; j < len(hidden); j++ {
		score += hidden[j] * w.FC2Weight[0][j]
	}

	if score > r.threshold {
		return ClassPro, TierHigh, nil
	}
	
	if score < 0.2 {
		return ClassFlashLite, TierLow, nil
	}
	
	return ClassFlash, TierMedium, nil
}
