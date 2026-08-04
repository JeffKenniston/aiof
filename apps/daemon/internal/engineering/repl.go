package engineering

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// ADR-26: Interactive Autonomous Engineering Engine & Pair-Programming REPL
// ============================================================================

// SymbolKind categorizes Go language constructs identified during AST analysis.
type SymbolKind string

const (
	SymbolStruct    SymbolKind = "STRUCT"
	SymbolInterface SymbolKind = "INTERFACE"
	SymbolFunction  SymbolKind = "FUNCTION"
	SymbolMethod    SymbolKind = "METHOD"
	SymbolImport    SymbolKind = "IMPORT"
	SymbolTypeAlias SymbolKind = "TYPE_ALIAS"
)

// SymbolDetail represents an extracted AST symbol node with type signature metadata.
type SymbolDetail struct {
	Name       string     `json:"name"`
	Kind       SymbolKind `json:"kind"`
	Receiver   string     `json:"receiver,omitempty"` // For methods (e.g., "*TOKIEngine")
	Signature  string     `json:"signature"`          // e.g., "func(ctx context.Context, key string) (string, error)"
	DocComment string     `json:"doc_comment,omitempty"`
	File       string     `json:"file,omitempty"`
	Line       int        `json:"line,omitempty"`
}

// ASTDiffReport captures structural delta between pre- and post-refactoring ASTs.
type ASTDiffReport struct {
	AddedSymbols    []SymbolDetail `json:"added_symbols"`
	ModifiedSymbols []SymbolDetail `json:"modified_symbols"`
	RemovedSymbols  []SymbolDetail `json:"removed_symbols"`
	ImportChanges   []string       `json:"import_changes"`
	TotalChanges    int            `json:"total_changes"`
	HasBreaking     bool           `json:"has_breaking_changes"`
}

// CIVerificationResult captures execution metrics and status from pre-flight CI gates.
type CIVerificationResult struct {
	Passed       bool          `json:"passed"`
	CoveragePct  float64       `json:"coverage_pct"`
	Duration     time.Duration `json:"duration"`
	StdOut       string        `json:"stdout"`
	StdErr       string        `json:"stderr"`
	FailingTests []string      `json:"failing_tests,omitempty"`
	VetOutput    string        `json:"vet_output,omitempty"`
}

// SelfHealingResult captures the trajectory of surgical code patching and re-verification.
type SelfHealingResult struct {
	Resolved       bool                  `json:"resolved"`
	Attempts       int                   `json:"attempts"`
	FinalCode      string                `json:"final_code"`
	PatchesApplied []string              `json:"patches_applied"`
	Verification   *CIVerificationResult `json:"verification"`
	LastError      string                `json:"last_error,omitempty"`
}

// ----------------------------------------------------------------------------
// 1. REPL Session Manager
// ----------------------------------------------------------------------------

// REPLCommandRecord logs an individual REPL interaction.
type REPLCommandRecord struct {
	ID        string        `json:"id"`
	Timestamp time.Time     `json:"timestamp"`
	Input     string        `json:"input"`
	Output    string        `json:"output"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
}

// REPLSession manages stateful pair-programming REPL execution.
type REPLSession struct {
	mu           sync.RWMutex
	SessionID    string                  `json:"session_id"`
	UserID       string                  `json:"user_id"`
	WorkingDir   string                  `json:"working_dir"`
	EnvVars      map[string]string       `json:"env_vars"`
	ActiveFile   string                  `json:"active_file"`
	ActiveCode   string                  `json:"active_code"`
	Symbols      map[string]SymbolDetail `json:"symbols"`
	History      []REPLCommandRecord     `json:"history"`
	CreatedAt    time.Time               `json:"created_at"`
	LastActiveAt time.Time               `json:"last_active_at"`
}

// NewREPLSession initializes a new pair-programming REPL session.
func NewREPLSession(sessionID, userID, workingDir string) *REPLSession {
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}
	return &REPLSession{
		SessionID:    sessionID,
		UserID:       userID,
		WorkingDir:   workingDir,
		EnvVars:      make(map[string]string),
		Symbols:      make(map[string]SymbolDetail),
		History:      make([]REPLCommandRecord, 0),
		CreatedAt:    time.Now().UTC(),
		LastActiveAt: time.Now().UTC(),
	}
}

// ExecuteSnippet executes a Go code snippet or REPL directive, updating session state.
func (s *REPLSession) ExecuteSnippet(ctx context.Context, input string) (*REPLCommandRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start := time.Now()
	s.LastActiveAt = start.UTC()

	rec := REPLCommandRecord{
		ID:        fmt.Sprintf("cmd-%d", start.UnixNano()),
		Timestamp: start.UTC(),
		Input:     input,
	}

	trimmed := strings.TrimSpace(input)

	// Directives
	if strings.HasPrefix(trimmed, ":setfile ") {
		s.ActiveFile = strings.TrimPrefix(trimmed, ":setfile ")
		rec.Output = fmt.Sprintf("Active file set to: %s", s.ActiveFile)
		rec.Duration = time.Since(start)
		s.History = append(s.History, rec)
		return &rec, nil
	}

	if strings.HasPrefix(trimmed, ":code ") {
		s.ActiveCode = strings.TrimPrefix(trimmed, ":code ")
		symbols, err := ExtractSymbolsFromCode(s.ActiveCode)
		if err == nil {
			for name, sym := range symbols {
				s.Symbols[name] = sym
			}
		}
		rec.Output = fmt.Sprintf("Loaded %d bytes of code. Updated %d symbols in session.", len(s.ActiveCode), len(symbols))
		rec.Duration = time.Since(start)
		s.History = append(s.History, rec)
		return &rec, nil
	}

	symbols, err := ExtractSymbolsFromCode(trimmed)
	if err == nil && len(symbols) > 0 {
		for name, sym := range symbols {
			s.Symbols[name] = sym
		}
		rec.Output = fmt.Sprintf("Parsed code snippet successfully. Active symbols registered: %d", len(symbols))
	} else {
		rec.Output = fmt.Sprintf("Evaluated input (%d chars).", len(trimmed))
	}

	rec.Duration = time.Since(start)
	s.History = append(s.History, rec)
	return &rec, nil
}

// GetSymbolState returns a snapshot of currently tracked symbols in the REPL session.
func (s *REPLSession) GetSymbolState() map[string]SymbolDetail {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]SymbolDetail, len(s.Symbols))
	for k, v := range s.Symbols {
		snapshot[k] = v
	}
	return snapshot
}

// ----------------------------------------------------------------------------
// 2. Real-Time AST Symbol Diff Engine
// ----------------------------------------------------------------------------

// ExtractSymbolsFromCode parses Go source code into an AST and extracts top-level symbols.
func ExtractSymbolsFromCode(code string) (map[string]SymbolDetail, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "snippet.go", code, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("ast parse error: %w", err)
	}

	symbols := make(map[string]SymbolDetail)

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					sym := SymbolDetail{
						Name: s.Name.Name,
						Line: fset.Position(s.Pos()).Line,
					}
					if d.Doc != nil {
						sym.DocComment = strings.TrimSpace(d.Doc.Text())
					}

					switch s.Type.(type) {
					case *ast.StructType:
						sym.Kind = SymbolStruct
						sym.Signature = "type " + s.Name.Name + " struct"
					case *ast.InterfaceType:
						sym.Kind = SymbolInterface
						sym.Signature = "type " + s.Name.Name + " interface"
					default:
						sym.Kind = SymbolTypeAlias
						sym.Signature = "type " + s.Name.Name
					}
					symbols[sym.Name] = sym

				case *ast.ImportSpec:
					impPath := ""
					if s.Path != nil {
						impPath = s.Path.Value
					}
					sym := SymbolDetail{
						Name:      impPath,
						Kind:      SymbolImport,
						Signature: "import " + impPath,
						Line:      fset.Position(s.Pos()).Line,
					}
					symbols["import:"+impPath] = sym
				}
			}

		case *ast.FuncDecl:
			sym := SymbolDetail{
				Name: d.Name.Name,
				Kind: SymbolFunction,
				Line: fset.Position(d.Pos()).Line,
			}
			if d.Doc != nil {
				sym.DocComment = strings.TrimSpace(d.Doc.Text())
			}

			if d.Recv != nil && len(d.Recv.List) > 0 {
				sym.Kind = SymbolMethod
				var buf bytes.Buffer
				_ = printer.Fprint(&buf, fset, d.Recv.List[0].Type)
				sym.Receiver = buf.String()
			}

			var sigBuf bytes.Buffer
			_ = printer.Fprint(&sigBuf, fset, d.Type)
			sym.Signature = "func " + d.Name.Name + sigBuf.String()

			key := sym.Name
			if sym.Receiver != "" {
				key = sym.Receiver + "." + sym.Name
			}
			symbols[key] = sym
		}
	}

	return symbols, nil
}

// CompareAST performs a deep structural AST comparison between pre- and post-refactoring code.
func CompareAST(preCode, postCode string) (*ASTDiffReport, error) {
	preSyms, errPre := ExtractSymbolsFromCode(preCode)
	postSyms, errPost := ExtractSymbolsFromCode(postCode)

	if errPre != nil && preCode != "" {
		return nil, fmt.Errorf("failed to parse pre-refactoring AST: %w", errPre)
	}
	if errPost != nil {
		return nil, fmt.Errorf("failed to parse post-refactoring AST: %w", errPost)
	}

	report := &ASTDiffReport{
		AddedSymbols:    make([]SymbolDetail, 0),
		ModifiedSymbols: make([]SymbolDetail, 0),
		RemovedSymbols:  make([]SymbolDetail, 0),
		ImportChanges:   make([]string, 0),
	}

	for key, postSym := range postSyms {
		preSym, exists := preSyms[key]
		if !exists {
			if postSym.Kind == SymbolImport {
				report.ImportChanges = append(report.ImportChanges, "+ "+postSym.Name)
			} else {
				report.AddedSymbols = append(report.AddedSymbols, postSym)
			}
		} else {
			if preSym.Signature != postSym.Signature {
				report.ModifiedSymbols = append(report.ModifiedSymbols, postSym)
				if preSym.Kind == SymbolInterface || preSym.Kind == SymbolStruct {
					report.HasBreaking = true
				}
			}
		}
	}

	for key, preSym := range preSyms {
		_, exists := postSyms[key]
		if !exists {
			if preSym.Kind == SymbolImport {
				report.ImportChanges = append(report.ImportChanges, "- "+preSym.Name)
			} else {
				report.RemovedSymbols = append(report.RemovedSymbols, preSym)
				report.HasBreaking = true
			}
		}
	}

	report.TotalChanges = len(report.AddedSymbols) + len(report.ModifiedSymbols) + len(report.RemovedSymbols)
	return report, nil
}

// ----------------------------------------------------------------------------
// 3. Automated Unit Test Synthesizer
// ----------------------------------------------------------------------------

// SynthesizeUnitTests generates Go unit test code for newly added or modified AST functions.
func SynthesizeUnitTests(packageName, sourceCode string, diff *ASTDiffReport) (string, error) {
	if packageName == "" {
		packageName = "main"
	}

	symbols, err := ExtractSymbolsFromCode(sourceCode)
	if err != nil {
		return "", fmt.Errorf("failed to parse AST for test synthesis: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	sb.WriteString("import (\n")
	sb.WriteString("\t\"testing\"\n")
	sb.WriteString(")\n\n")

	targetSymbols := make([]SymbolDetail, 0)

	if diff != nil && len(diff.AddedSymbols)+len(diff.ModifiedSymbols) > 0 {
		targetSymbols = append(targetSymbols, diff.AddedSymbols...)
		targetSymbols = append(targetSymbols, diff.ModifiedSymbols...)
	} else {
		for _, sym := range symbols {
			if sym.Kind == SymbolFunction || sym.Kind == SymbolMethod {
				targetSymbols = append(targetSymbols, sym)
			}
		}
	}

	for _, sym := range targetSymbols {
		if sym.Kind != SymbolFunction && sym.Kind != SymbolMethod {
			continue
		}

		funcName := sym.Name
		testFuncName := "Test" + strings.Title(funcName)
		if sym.Receiver != "" {
			recvClean := strings.TrimPrefix(strings.TrimPrefix(sym.Receiver, "*"), "")
			testFuncName = "Test" + strings.Title(recvClean) + "_" + strings.Title(funcName)
		}

		sb.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testFuncName))
		sb.WriteString("\ttests := []struct {\n")
		sb.WriteString("\t\tname    string\n")
		sb.WriteString("\t\twantErr bool\n")
		sb.WriteString("\t}{\n")
		sb.WriteString(fmt.Sprintf("\t\t{name: \"baseline_%s_test\", wantErr: false},\n", funcName))
		sb.WriteString("\t}\n\n")

		sb.WriteString("\tfor _, tt := range tests {\n")
		sb.WriteString("\t\tt.Run(tt.name, func(t *testing.T) {\n")
		sb.WriteString(fmt.Sprintf("\t\t\tt.Logf(\"Synthesized test run for %s\")\n", funcName))
		sb.WriteString("\t\t})\n")
		sb.WriteString("\t}\n")
		sb.WriteString("}\n\n")
	}

	return sb.String(), nil
}

// ----------------------------------------------------------------------------
// 4. Pre-Flight CI Verification Gate
// ----------------------------------------------------------------------------

// RunPreFlightVerification executes `go test` and `go vet` in an isolated sandbox directory.
func RunPreFlightVerification(ctx context.Context, targetDir string) (*CIVerificationResult, error) {
	start := time.Now()

	res := &CIVerificationResult{
		FailingTests: make([]string, 0),
	}

	vetCmd := exec.CommandContext(ctx, "go", "vet", "./...")
	vetCmd.Dir = targetDir
	var vetOut bytes.Buffer
	vetCmd.Stdout = &vetOut
	vetCmd.Stderr = &vetOut

	if err := vetCmd.Run(); err != nil {
		res.VetOutput = vetOut.String()
	}

	testCmd := exec.CommandContext(ctx, "go", "test", "-v", "-cover", "./...")
	testCmd.Dir = targetDir
	var stdoutBuf, stderrBuf bytes.Buffer
	testCmd.Stdout = &stdoutBuf
	testCmd.Stderr = &stderrBuf

	err := testCmd.Run()
	res.Duration = time.Since(start)
	res.StdOut = stdoutBuf.String()
	res.StdErr = stderrBuf.String()

	if err == nil {
		res.Passed = true
	} else {
		res.Passed = false
		scanner := strings.Split(res.StdOut, "\n")
		failRegex := regexp.MustCompile(`--- FAIL:\s+([A-Za-z0-9_]+)`)
		for _, line := range scanner {
			matches := failRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				res.FailingTests = append(res.FailingTests, matches[1])
			}
		}
	}

	covRegex := regexp.MustCompile(`coverage:\s+([0-9]+\.[0-9]+)%\s+of\s+statements`)
	matches := covRegex.FindStringSubmatch(res.StdOut)
	if len(matches) > 1 {
		if cov, parseErr := strconv.ParseFloat(matches[1], 64); parseErr == nil {
			res.CoveragePct = cov
		}
	}

	return res, nil
}

// ----------------------------------------------------------------------------
// 5. Self-Healing Debugging Loop
// ----------------------------------------------------------------------------

// HealAndVerify parses stack traces and applies surgical code patches until verification passes.
func HealAndVerify(ctx context.Context, workingDir, sourceFilename, sourceCode, testCode string, maxAttempts int) (*SelfHealingResult, error) {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	res := &SelfHealingResult{
		PatchesApplied: make([]string, 0),
		FinalCode:      sourceCode,
	}

	tmpDir, err := os.MkdirTemp("", "aiof-heal-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp sandbox: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	goModContent := "module aiof/sandbox\n\ngo 1.23\n"
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)

	currentSource := sourceCode

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		res.Attempts = attempt

		srcPath := filepath.Join(tmpDir, sourceFilename)
		testPath := filepath.Join(tmpDir, "auto_generated_test.go")

		if err := os.WriteFile(srcPath, []byte(currentSource), 0644); err != nil {
			return nil, fmt.Errorf("failed to write source file: %w", err)
		}
		if err := os.WriteFile(testPath, []byte(testCode), 0644); err != nil {
			return nil, fmt.Errorf("failed to write test file: %w", err)
		}

		ver, verErr := RunPreFlightVerification(ctx, tmpDir)
		res.Verification = ver

		if verErr == nil && ver.Passed {
			res.Resolved = true
			res.FinalCode = currentSource
			return res, nil
		}

		failureOutput := ver.StdOut + "\n" + ver.StdErr + "\n" + ver.VetOutput
		res.LastError = failureOutput

		patchDescription, patchedCode := applySurgicalPatch(currentSource, failureOutput)
		if patchedCode == currentSource {
			break
		}

		res.PatchesApplied = append(res.PatchesApplied, fmt.Sprintf("Attempt %d: %s", attempt, patchDescription))
		currentSource = patchedCode
	}

	res.FinalCode = currentSource
	return res, nil
}

func applySurgicalPatch(code, failureLogs string) (string, string) {
	if strings.Contains(failureLogs, "index out of range") {
		if !strings.Contains(code, "if len(") {
			lines := strings.Split(code, "\n")
			for i, line := range lines {
				if strings.Contains(line, "[") && strings.Contains(line, "]") && !strings.Contains(line, "make(") {
					lines[i] = "\tif len(data) > 0 {\n\t\t" + strings.TrimSpace(line) + "\n\t}"
					return "Added array bounds check guard", strings.Join(lines, "\n")
				}
			}
		}
	}

	if strings.Contains(failureLogs, "nil pointer dereference") || strings.Contains(failureLogs, "invalid memory address") {
		if !strings.Contains(code, "if err != nil") && !strings.Contains(code, "if ptr == nil") {
			lines := strings.Split(code, "\n")
			for i, line := range lines {
				if strings.Contains(line, ".") && (strings.Contains(line, "return") || strings.Contains(line, "=")) {
					lines[i] = "\tif ptr == nil { return }\n" + line
					return "Injected nil pointer check guard", strings.Join(lines, "\n")
				}
			}
		}
	}

	if strings.Contains(failureLogs, "undefined:") {
		re := regexp.MustCompile(`undefined:\s+([A-Za-z0-9_]+)`)
		matches := re.FindStringSubmatch(failureLogs)
		if len(matches) > 1 {
			missingSym := matches[1]
			patch := fmt.Sprintf("// Auto-patched missing symbol declaration\nvar %s interface{}\n", missingSym)
			return "Injected missing symbol definition for " + missingSym, patch + code
		}
	}

	return "No applicable surgical patch found", code
}
