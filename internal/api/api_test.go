package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupTestServer() (*Server, *http.ServeMux) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer(logger, nil, nil)
	mux := http.NewServeMux()
	server.RegisterHandlers(mux)
	return server, mux
}

func TestRegisterHandlers(t *testing.T) {
	_, mux := setupTestServer()

	req := httptest.NewRequest("GET", "/api/health", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", rr.Code)
	}
}
