package engineering

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExtractSymbolsAndASTDiff(t *testing.T) {
	preCode := `package sample

type Config struct {
	Port int
}

func Process(data string) (string, error) {
	return data, nil
}
`

	postCode := `package sample

import "context"

type Config struct {
	Port int
	Host string
}

type Engine interface {
	Run(ctx context.Context) error
}

func Process(data string) (string, error) {
	return data + "_updated", nil
}

func NewEngine() Engine {
	return nil
}
`

	diff, err := CompareAST(preCode, postCode)
	if err != nil {
		t.Fatalf("CompareAST failed: %v", err)
	}

	if diff.TotalChanges == 0 {
		t.Errorf("Expected AST diff changes > 0, got 0")
	}

	foundAddedFunc := false
	foundAddedInterface := false

	for _, sym := range diff.AddedSymbols {
		if sym.Name == "NewEngine" && sym.Kind == SymbolFunction {
			foundAddedFunc = true
		}
		if sym.Name == "Engine" && sym.Kind == SymbolInterface {
			foundAddedInterface = true
		}
	}

	if !foundAddedFunc {
		t.Errorf("Expected 'NewEngine' in added symbols")
	}
	if !foundAddedInterface {
		t.Errorf("Expected 'Engine' in added symbols")
	}
}

func TestSynthesizeUnitTests(t *testing.T) {
	code := `package calculator

func Add(a, b int) int {
	return a + b
}

type Service struct{}

func (s *Service) Execute(cmd string) error {
	return nil
}
`

	diff, err := CompareAST("", code)
	if err != nil {
		t.Fatalf("CompareAST failed: %v", err)
	}

	testCode, err := SynthesizeUnitTests("calculator", code, diff)
	if err != nil {
		t.Fatalf("SynthesizeUnitTests failed: %v", err)
	}

	if !strings.Contains(testCode, "package calculator") {
		t.Errorf("Expected package calculator in test code")
	}
	if !strings.Contains(testCode, "func TestAdd(") {
		t.Errorf("Expected TestAdd in synthesized test code")
	}
	if !strings.Contains(testCode, "func TestService_Execute(") {
		t.Errorf("Expected TestService_Execute in synthesized test code")
	}
}

func TestREPLSession(t *testing.T) {
	session := NewREPLSession("sess-123", "user-456", "")
	ctx := context.Background()

	cmd1, err := session.ExecuteSnippet(ctx, ":setfile main.go")
	_ = cmd1
	if err != nil {
		t.Fatalf("ExecuteSnippet setfile failed: %v", err)
	}
	if session.ActiveFile != "main.go" {
		t.Errorf("Expected active file main.go, got %s", session.ActiveFile)
	}

	codeSnippet := `package main
type Worker struct { ID string }
func (w *Worker) Start() {}
`
	cmd2, err := session.ExecuteSnippet(ctx, codeSnippet)
	if err != nil {
		t.Fatalf("ExecuteSnippet code snippet failed: %v", err)
	}
	if cmd2.Output == "" {
		t.Errorf("Expected output message in command record")
	}

	syms := session.GetSymbolState()
	if _, exists := syms["Worker"]; !exists {
		t.Errorf("Expected 'Worker' in REPL symbol state")
	}
	if _, exists := syms["*Worker.Start"]; !exists {
		t.Errorf("Expected '*Worker.Start' in REPL symbol state")
	}
}

func TestSelfHealingEngine(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sourceCode := `package sandbox

func GetValue(data []string) string {
	return data[0]
}
`

	testCode := `package sandbox

import "testing"

func TestGetValue(t *testing.T) {
	var empty []string
	_ = GetValue(empty)
}
`

	res, err := HealAndVerify(ctx, "", "sandbox.go", sourceCode, testCode, 2)
	if err != nil {
		t.Fatalf("HealAndVerify encountered system error: %v", err)
	}

	if res.Attempts == 0 {
		t.Errorf("Expected attempts > 0")
	}
}
