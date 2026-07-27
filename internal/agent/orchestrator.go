package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"go/ast"
	"go/parser"
	"go/token"

	"google.golang.org/genai"
)

// AgentHost provides the necessary context and callbacks to agents without circular dependencies.
type AgentHost interface {
	EmitLog(ctx context.Context, agentName, severity, message string)
	EmitState(ctx context.Context, docType string, id string, payload string)
	GetProjectDir() string
	GetLatestVLMFrame(ctx context.Context) string
}

// Agent defines the base interface for executable agent capabilities.
type Agent interface {
	Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error)
}

// ValidatorAgent implements Phase 2.4 Consensus Protocols.
// It is responsible for static analysis, dependency checks, and blast-radius estimation.
type ValidatorAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewValidatorAgent(logger *slog.Logger, client *genai.Client) *ValidatorAgent {
	return &ValidatorAgent{logger: logger, client: client}
}

func (v *ValidatorAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	proposedChanges := input
	host.EmitLog(ctx, "Validator", "info", "Commencing blast-radius estimation and static analysis consensus...")
	host.EmitState(ctx, "agent_graph_state", "validator", `{"id": "validator", "label": "Validator", "model": "Pro", "color": "#ffb86c", "status": "running"}`)
	
	if len(proposedChanges) == 0 {
		host.EmitLog(ctx, "Validator", "info", "No proposed changes to validate.")
		host.EmitState(ctx, "agent_graph_state", "validator", `{"id": "validator", "label": "Validator", "model": "Pro", "color": "#50fa7b", "status": "success"}`)
		return "APPROVE (no changes)", nil
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Role: "system",
			Parts: []*genai.Part{
				{Text: "You are the Validator Consensus Agent. Analyze the proposed changes for blast-radius impact and static dependency conflicts. Reply exactly with 'APPROVE' or 'REJECT: <reason>'."},
			},
		},
	}

	// Phase 2.5: Programmatic Static Analysis (AST-based)
	fileCount := strings.Count(proposedChanges, "diff --git")
	if fileCount == 0 {
		fileCount = strings.Count(proposedChanges, "--- a/")
	}
	if fileCount == 0 {
		fileCount = strings.Count(proposedChanges, "File ") // Fallback for our executor's format
	}
	if fileCount > 20 {
		host.EmitLog(ctx, "Validator", "error", "blast radius exceeds 20 files")
		return "", fmt.Errorf("blast radius exceeds 20 files")
	}

	var foundPatterns []string

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", proposedChanges, parser.AllErrors)
	if err == nil {
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok {
						pkgName := id.Name
						funcName := sel.Sel.Name
						if pkgName == "os" && funcName == "Exit" {
							foundPatterns = append(foundPatterns, "os.Exit")
						} else if pkgName == "exec" && funcName == "Command" {
							foundPatterns = append(foundPatterns, "exec.Command")
						} else if pkgName == "syscall" {
							foundPatterns = append(foundPatterns, "syscall."+funcName)
						}
					}
				}
			}
			return true
		})
	} else {
		host.EmitLog(ctx, "Validator", "warn", "Could not parse code as Go for strict static analysis; falling back to string matching")
		dangerousPatterns := []string{"os.Exit", "exec.Command", "syscall.", "unsafe.Pointer"}
		for _, pattern := range dangerousPatterns {
			if strings.Contains(proposedChanges, pattern) {
				foundPatterns = append(foundPatterns, pattern)
			}
		}
	}

	promptText := fmt.Sprintf("Proposed Changes:\n%s\n\nAnalyze and approve or reject.", proposedChanges)
	if len(foundPatterns) > 0 {
		promptText = fmt.Sprintf("Proposed Changes:\n%s\n\nWARNING: The following dangerous patterns were detected: %v. Please evaluate them carefully.\n\nAnalyze and approve or reject.", proposedChanges, foundPatterns)
	}

	if v.client == nil {
		return "", fmt.Errorf("validator client is nil. Prompt: %s", promptText)
	}

	resp, err := v.client.Models.GenerateContent(ctx, "gemini-3.5-flash", []*genai.Content{
		genai.NewContentFromText(promptText, ""),
	}, config)
	
	if err != nil {
		host.EmitLog(ctx, "Validator", "error", fmt.Sprintf("Consensus failure: %v", err))
		host.EmitState(ctx, "agent_graph_state", "validator", `{"id": "validator", "label": "Validator", "model": "Pro", "color": "#ff5555", "status": "error"}`)
		return "", err
	}

	host.EmitState(ctx, "agent_graph_state", "validator", `{"id": "validator", "label": "Validator", "model": "Pro", "color": "#50fa7b", "status": "success"}`)

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		decision := resp.Candidates[0].Content.Parts[0].Text
		host.EmitLog(ctx, "Validator", "info", fmt.Sprintf("Consensus Decision: %s", decision))
		if strings.HasPrefix(strings.TrimSpace(strings.ToUpper(decision)), "REJECT") {
			return decision, fmt.Errorf("validator rejected changes")
		}
		return decision, nil
	}

	return "APPROVE", nil
}

// SetValuedRouter implements Phase 2.3 Set-Valued Predictor Routing.
// It executes target nodes concurrently rather than relying on sequential intent classification.
type SetValuedRouter struct {
	logger   *slog.Logger
	mu       sync.RWMutex
	registry map[string]Agent
}

func NewSetValuedRouter(logger *slog.Logger) *SetValuedRouter {
	return &SetValuedRouter{
		logger:   logger,
		registry: make(map[string]Agent),
	}
}

// RegisterAgent maps a semantic node string to an executable Agent.
func (r *SetValuedRouter) RegisterAgent(name string, agent Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registry[name] = agent
}

// Dispatch consumes a CognitiveSequence and spawns all target nodes in parallel.
func (r *SetValuedRouter) Dispatch(ctx context.Context, host AgentHost, input string, sequence CognitiveSequence) (string, error) {
	r.logger.Info("Dispatching Set-Valued routing matrix concurrently", slog.Any("nodes", sequence.TargetNodes))

	var wg sync.WaitGroup
	errCh := make(chan error, len(sequence.TargetNodes))
	resCh := make(chan string, len(sequence.TargetNodes))

	for _, nodeName := range sequence.TargetNodes {
		r.mu.RLock()
		agent, ok := r.registry[nodeName]
		r.mu.RUnlock()
		if !ok {
			r.logger.Warn("Unknown target node requested by orchestrator", slog.String("node", nodeName))
			continue
		}

		wg.Add(1)
		go func(name string, a Agent) {
			defer wg.Done()
			r.logger.Debug("Spawning parallel agent thread", slog.String("agent", name))
			res, err := a.Execute(ctx, host, input, sequence.DynamicStateMutations)
			if err != nil {
				errCh <- fmt.Errorf("agent %s failed consensus: %w", name, err)
			}
			if res != "" {
				resCh <- res
			}
		}(nodeName, agent)
	}

	wg.Wait()
	close(errCh)
	close(resCh)

	// Collect any validation or execution consensus errors.
	for err := range errCh {
		if err != nil {
			return "", err
		}
	}

	var globalChanges string
	for res := range resCh {
		globalChanges += res
	}

	return globalChanges, nil
}
