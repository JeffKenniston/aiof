package ingestion

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRepository(t *testing.T) {
	tempDir := t.TempDir()
	
	os.MkdirAll(filepath.Join(tempDir, ".git"), 0755)
	os.WriteFile(filepath.Join(tempDir, ".git", "config"), []byte("git config"), 0644)
	
	os.MkdirAll(filepath.Join(tempDir, "node_modules"), 0755)
	os.WriteFile(filepath.Join(tempDir, "node_modules", "package.json"), []byte("{}"), 0644)

	os.MkdirAll(filepath.Join(tempDir, "src"), 0755)
	validGoCode := `package main

import "fmt"

type TestStruct struct {}

func main() {
	fmt.Println("Hello")
}
`
	os.WriteFile(filepath.Join(tempDir, "src", "main.go"), []byte(validGoCode), 0644)

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	parser := NewASTParser(logger, nil)
	err := parser.ParseRepository(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	logOutput := buf.String()
	if !strings.Contains(logOutput, "Ingesting file via AST") {
		t.Errorf("expected to ingest valid Go file, got log: %s", logOutput)
	}
	
	if strings.Contains(logOutput, "node_modules") {
		t.Errorf("should ignore node_modules directory")
	}
}
