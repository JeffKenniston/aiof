package ast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTSDGCompactionEngine(t *testing.T) {
	// Create temporary directory with dummy Go package files
	tmpDir, err := os.MkdirTemp("", "tsdg_test_*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write sample Go source — struct tags omitted to avoid backtick nesting issues.
	sampleCode := "package sample\n\nimport (\n\t\"context\"\n\t\"fmt\"\n)\n\n// UserStore defines database operations.\ntype UserStore interface {\n\tGetUser(ctx context.Context, id string) (*User, error)\n\tSaveUser(ctx context.Context, u *User) error\n}\n\n// User represents a system entity.\ntype User struct {\n\tID       string\n\tUsername string\n\tEmail    string\n}\n\n// Service provides user business logic.\ntype Service struct {\n\tstore UserStore\n}\n\nfunc NewService(store UserStore) *Service {\n\treturn &Service{store: store}\n}\n\nfunc (s *Service) ProcessUser(ctx context.Context, id string) (string, error) {\n\tu, err := s.store.GetUser(ctx, id)\n\tif err != nil {\n\t\treturn \"\", fmt.Errorf(\"process user failed: %w\", err)\n\t}\n\treturn u.Username, nil\n}\n"

	sampleFile := filepath.Join(tmpDir, "sample.go")
	if err := os.WriteFile(sampleFile, []byte(sampleCode), 0644); err != nil {
		t.Fatalf("failed to write sample.go: %v", err)
	}

	compactor := NewTSDGCompactor()

	// Case 1: Build TSDG without modifying ProcessUser (should compact ProcessUser body)
	modified := map[string]bool{
		"NewService": true, // Only NewService is modified
	}

	graph, err := compactor.BuildGraph(tmpDir, modified)
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	if graph == nil {
		t.Fatal("expected non-nil TSDGGraph")
	}

	// Verify Phase 1: Symbol Table & Import DAG
	if len(graph.SymbolTable) == 0 {
		t.Errorf("expected symbol table entries, got 0")
	}

	userStoreSym, exists := graph.SymbolTable["sample.UserStore"]
	if !exists || userStoreSym.Kind != SymbolInterface {
		t.Errorf("expected sample.UserStore interface symbol, got: %+v", userStoreSym)
	}

	if len(userStoreSym.Methods) != 2 {
		t.Errorf("expected 2 methods on UserStore interface, got %d", len(userStoreSym.Methods))
	}

	userSym, exists := graph.SymbolTable["sample.User"]
	if !exists || userSym.Kind != SymbolStruct {
		t.Errorf("expected sample.User struct symbol, got: %+v", userSym)
	}

	if len(userSym.Fields) != 3 {
		t.Errorf("expected 3 fields on User struct, got %d", len(userSym.Fields))
	}

	// Verify Phase 2: Body Compaction
	compactedCode, ok := graph.CompactCode["sample/sample.go"]
	if !ok {
		t.Fatalf("compacted code missing for sample/sample.go")
	}

	// ProcessUser should be summarized because it was not in modified map
	if !strings.Contains(compactedCode, "// [summarized body:") {
		t.Errorf("expected summarized body comment in compacted code, got:\n%s", compactedCode)
	}

	// NewService should NOT be summarized because it was marked modified
	if strings.Contains(compactedCode, "func NewService") && strings.Contains(compactedCode, "// [summarized body:") {
		// Verify NewService has full implementation
		if !strings.Contains(compactedCode, "return &Service{store: store}") {
			t.Errorf("expected NewService full body to be preserved, got:\n%s", compactedCode)
		}
	}

	// Verify KV-cache Friendly Serialization
	jsonStr, err := graph.ToCompactJSON()
	if err != nil {
		t.Fatalf("ToCompactJSON failed: %v", err)
	}

	if len(jsonStr) == 0 {
		t.Errorf("expected non-empty JSON string")
	}

	restoredGraph, err := FromCompactJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromCompactJSON failed: %v", err)
	}

	if restoredGraph.ModuleName != graph.ModuleName {
		t.Errorf("expected module name %s, got %s", graph.ModuleName, restoredGraph.ModuleName)
	}
}
