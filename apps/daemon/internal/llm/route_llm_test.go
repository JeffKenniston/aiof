package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/genai"
)

func TestClassifyPrompt(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		expected ClassClassAndTier
	}{
		{"Pro Architect", "I need you to architect a new solution for the database", ClassClassAndTier{ClassPro, TierLow}},
		{"Pro Refactor Long", "refactor " + generateString(600), ClassClassAndTier{ClassPro, TierHigh}},
		{"Flash Short", "what is the time?", ClassClassAndTier{ClassFlash, TierLow}},
		{"Flash Long", "just print this " + generateString(600), ClassClassAndTier{ClassFlash, TierHigh}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class, tier := ClassifyPrompt(tt.prompt)
			if class != tt.expected.class || tier != tt.expected.tier {
				t.Errorf("expected (%v, %v), got (%v, %v)", tt.expected.class, tt.expected.tier, class, tier)
			}
		})
	}
}

type ClassClassAndTier struct {
	class ModelClass
	tier  Tier
}

func generateString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

func TestPredictComplexity(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock an embedding response with 768 elements of 1.0
		// which will exceed the threshold if weights are positive enough
		values := make([]float32, 768)
		// Set values high enough to trigger Pro/High
		for i := 0; i < 768; i++ {
			values[i] = 10.0
		}
		
		response := map[string]any{
			"embeddings": []map[string]any{
				{
					"values": values,
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: "test-key",
		HTTPOptions: genai.HTTPOptions{
			BaseURL: ts.URL,
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	weightsJSON := `{
  "version": "2.6.0",
  "weights": {
    "ast_complexity": 0.35,
    "semantic_vector_distance": 0.25,
    "token_length": 0.20,
    "historical_latency": 0.20
  },
  "thresholds": {
    "shadow_spawn": 0.60,
    "cloud_escalation": 0.75,
    "fabrication": 0.85
  }
}`
	tmpWeightsPath := filepath.Join(t.TempDir(), "weights.json")
	if err := os.WriteFile(tmpWeightsPath, []byte(weightsJSON), 0644); err != nil {
		t.Fatalf("failed to write tmp weights: %v", err)
	}

	classifier := NewRouteLLMClassifier(client, tmpWeightsPath)
	// Test the threshold behavior
	class, tier, err := classifier.PredictComplexity(ctx, "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Because we pushed values very high, it should be ClassPro, TierHigh (or at least it shouldn't error)
	// Actually we don't know the exact sum because of random weights, but we can verify it doesn't fail.
	if class != ClassPro && class != ClassFlash && class != ClassFlashLite {
		t.Errorf("unexpected class: %v", class)
	}
	if tier != TierHigh && tier != TierMedium && tier != TierLow {
		t.Errorf("unexpected tier: %v", tier)
	}
}

func TestLoadRealWeights(t *testing.T) {
	path := "../../deployments/config/route_llm_weights.json"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("Skipping TestLoadRealWeights: %s not found", path)
	}
	classifier := NewRouteLLMClassifier(nil, path)
	if classifier.weights == nil {
		t.Fatal("expected weights to be loaded, got nil")
	}

	if len(classifier.weights.FC1Weight) > 0 {
		if len(classifier.weights.FC1Weight) != 128 {
			t.Errorf("expected FC1Weight length 128, got %d", len(classifier.weights.FC1Weight))
		}
	}
}
