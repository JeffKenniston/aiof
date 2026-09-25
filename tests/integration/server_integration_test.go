package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aiof/internal/api"
)

func TestServerHealthIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := api.NewServer(logger, nil, nil)

	mux := http.NewServeMux()
	server.RegisterHandlers(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/health", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

func TestServerFileSystemIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := api.NewServer(logger, nil, nil)

	mux := http.NewServeMux()
	server.RegisterHandlers(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/files", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
