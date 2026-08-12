package agent

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
)

type mockHost struct{}

func (m *mockHost) EmitLog(ctx context.Context, agentName, sev, msg string) {}
func (m *mockHost) EmitState(ctx context.Context, docType, id, payload string) {}
func (m *mockHost) GetProjectDir() string { return "." }

func TestDispatchValidator_BlastRadius(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v := NewValidatorAgent(logger, nil)

	var changesBuilder strings.Builder
	for i := 0; i < 21; i++ {
		changesBuilder.WriteString("diff --git a/file")
		changesBuilder.WriteString(string(rune(i)))
		changesBuilder.WriteString(" b/file")
		changesBuilder.WriteString(string(rune(i)))
		changesBuilder.WriteString("\n")
	}

	ctx := context.Background()
	host := &mockHost{}
	decision, err := v.Execute(ctx, host, changesBuilder.String(), nil)
	if err == nil || !strings.Contains(err.Error(), "blast radius exceeds 20 files") {
		t.Errorf("Expected blast radius error, got decision %s and err %v", decision, err)
	}
}

func TestDispatchValidator_DangerousCode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v := NewValidatorAgent(logger, nil)

	ctx := context.Background()
	host := &mockHost{}

	proposedChanges := `
package main

import "os"

func main() {
	os.Exit(1)
}
`
	decision, err := v.Execute(ctx, host, proposedChanges, nil)
	if err == nil || !strings.Contains(err.Error(), "os.Exit") || !strings.Contains(err.Error(), "dangerous patterns were detected") {
		t.Errorf("Expected error to contain dangerous pattern warning, got decision %s, err %v", decision, err)
	}
}

func TestDispatchValidator_FallbackStringMatch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	v := NewValidatorAgent(logger, nil)

	ctx := context.Background()
	host := &mockHost{}

	proposedChanges := `
File main.go:
package main
import "os"
func main() {
	os.Exit(1)
`
	decision, err := v.Execute(ctx, host, proposedChanges, nil)
	if err == nil || !strings.Contains(err.Error(), "os.Exit") || !strings.Contains(err.Error(), "dangerous patterns were detected") {
		t.Errorf("Expected error to contain dangerous pattern warning from fallback, got decision %s, err %v", decision, err)
	}
}
