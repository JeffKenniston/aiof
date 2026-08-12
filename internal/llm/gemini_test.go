package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/genai"
)

func TestExecuteIntent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{"text": `{"action":"create_file","path":"test.txt"}`},
						},
					},
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

	gc := &GeminiClient{
		client: client,
	}

	res, err := gc.ExecuteIntent(ctx, "create a test.txt file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.RawOutput != `{"action":"create_file","path":"test.txt"}` {
		t.Errorf("unexpected RawOutput: %s", res.RawOutput)
	}
}
