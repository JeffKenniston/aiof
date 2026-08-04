package mcp

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"aiof/internal/security"
)

// ============================================================================
// ADR-08 & ADR-11: Enterprise MCP Protocol Governance Gateway
// ============================================================================

var (
	ErrInvalidJSONRPC      = errors.New("mcp/gateway: invalid JSON-RPC 2.0 request")
	ErrMethodNotFound      = errors.New("mcp/gateway: requested method not found")
	ErrUnauthorized        = errors.New("mcp/gateway: unauthorized tool invocation")
	ErrSignatureFailed     = errors.New("mcp/gateway: ED25519 payload signature verification failed")
	ErrCapabilityExpired   = errors.New("mcp/gateway: capability token expired or invalid")
	ErrToolExecutionFailed = errors.New("mcp/gateway: tool execution failed in sandbox")
	ErrElicitationTimeout  = errors.New("mcp/gateway: elicitation challenge timed out")
	ErrElicitationRequired = errors.New("mcp/gateway: elicitation challenge required before tool execution")
)

// ElicitationChallenge represents an MCP 2026-07-28 stateless multi-round-trip elicitation challenge.
type ElicitationChallenge struct {
	ChallengeID string          `json:"challenge_id"`
	ToolName    string          `json:"tool_name"`
	Prompt      string          `json:"prompt"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	ExpiresAt   time.Time       `json:"expires_at"`
}

// ElicitationResponse represents the client response to an ElicitationChallenge.
type ElicitationResponse struct {
	ChallengeID string          `json:"challenge_id"`
	Response    json.RawMessage `json:"response"`
}


// JSONRPCRequest represents a JSON-RPC 2.0 request envelope.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response envelope.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id"`
}

// ToolCallParams represents params passed to tools/call method.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// CapabilityToken holds JWT & Macaroon token headers passed in MCP requests.
type CapabilityToken struct {
	JWTToken  string             `json:"jwt_token,omitempty"`
	Macaroon  *security.Macaroon `json:"macaroon,omitempty"`
	Signature string             `json:"signature,omitempty"`  // Hex ED25519 signature
	PublicKey string             `json:"public_key,omitempty"` // Hex ED25519 public key
}

// ToolExecutorFunc represents a sandbox execution callback function.
type ToolExecutorFunc func(ctx context.Context, spec *ToolSpec, sanitizedInput json.RawMessage) (any, error)

// MCPGateway provides in-memory governance, ED25519 signature verification,
// Macaroon caveat checks, and RBAC payload sanitization upstream of sandboxes.
type MCPGateway struct {
	registry *ToolRegistry
	auth     *security.EnclaveKeyAuthority
	executor ToolExecutorFunc
	logger   *slog.Logger

	mu         sync.RWMutex
	usedTokens map[string]time.Time         // Nonce / JTI single-use cache
	challenges map[string]*ElicitationChallenge // Stateless multi-round-trip elicitation cache
}

// NewMCPGateway initializes a new MCP Governance Gateway instance.
func NewMCPGateway(registry *ToolRegistry, auth *security.EnclaveKeyAuthority, executor ToolExecutorFunc, logger *slog.Logger) *MCPGateway {
	if registry == nil {
		registry = NewToolRegistry()
	}
	return &MCPGateway{
		registry:   registry,
		auth:       auth,
		executor:   executor,
		logger:     logger,
		usedTokens: make(map[string]time.Time),
		challenges: make(map[string]*ElicitationChallenge),
	}
}

// HandleJSONRPC processes an incoming JSON-RPC 2.0 payload through the governance pipeline.
func (g *MCPGateway) HandleJSONRPC(ctx context.Context, capToken *CapabilityToken, rawMsg []byte) (*JSONRPCResponse, error) {
	var req JSONRPCRequest
	if err := json.Unmarshal(rawMsg, &req); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32700,
				Message: fmt.Sprintf("Parse error: %v", err),
			},
			ID: nil,
		}, nil
	}

	if req.JSONRPC != "2.0" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32600,
				Message: "Invalid Request: jsonrpc version must be '2.0'",
			},
			ID: req.ID,
		}, nil
	}

	switch req.Method {
	case "tools/list":
		return g.handleToolsList(req.ID), nil

	case "tools/call":
		return g.handleToolsCall(ctx, req, capToken, rawMsg)

	case "tools/elicitation_challenge":
		return g.handleElicitationChallenge(req), nil

	case "tools/elicitation_respond":
		return g.handleElicitationRespond(req), nil

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
			ID: req.ID,
		}, nil
	}
}

func (g *MCPGateway) handleToolsList(reqID interface{}) *JSONRPCResponse {
	specs := g.registry.List()
	toolList := make([]map[string]any, 0, len(specs))

	for _, spec := range specs {
		toolList = append(toolList, map[string]any{
			"name":                       spec.Name,
			"description":                spec.Description,
			"category":                   spec.Category,
			"version":                    spec.Version,
			"side_effect_classification": spec.SideEffect,
			"input_schema":               spec.InputSchema,
			"output_schema":              spec.OutputSchema,
		})
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result: map[string]any{
			"tools": toolList,
		},
		ID: reqID,
	}
}

func (g *MCPGateway) handleToolsCall(ctx context.Context, req JSONRPCRequest, capToken *CapabilityToken, rawMsg []byte) (*JSONRPCResponse, error) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32602,
				Message: fmt.Sprintf("Invalid params: %v", err),
			},
			ID: req.ID,
		}, nil
	}

	if params.Name == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: tool name required",
			},
			ID: req.ID,
		}, nil
	}

	// 1. Lookup Tool Specification (ADR-22)
	spec, err := g.registry.Lookup(params.Name)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Tool not found: %s", params.Name),
			},
			ID: req.ID,
		}, nil
	}

	// 2. Verify ED25519 Cryptographic Session Signature (ADR-11)
	if capToken != nil && capToken.Signature != "" && capToken.PublicKey != "" {
		pubBytes, err1 := hex.DecodeString(capToken.PublicKey)
		sigBytes, err2 := hex.DecodeString(capToken.Signature)

		if err1 == nil && err2 == nil && len(pubBytes) == ed25519.PublicKeySize && len(sigBytes) == ed25519.SignatureSize {
			if !ed25519.Verify(ed25519.PublicKey(pubBytes), rawMsg, sigBytes) {
				return &JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &JSONRPCError{
						Code:    -32001,
						Message: "Cryptographic signature verification failed",
					},
					ID: req.ID,
				}, nil
			}
		}
	}

	// 3. Evaluate Macaroon Capability Caveats (ADR-11, ADR-17)
	if capToken != nil && capToken.Macaroon != nil && g.auth != nil {
		opCtx := security.OperationContext{
			Action:          string(spec.SideEffect),
			Resource:        spec.Name,
			Role:            spec.RequiredRole,
			CurrentTime:     time.Now().UTC(),
			DelegationDepth: 1,
		}

		valid, err := capToken.Macaroon.Verify(g.auth, opCtx)
		if !valid || err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &JSONRPCError{
					Code:    -32002,
					Message: fmt.Sprintf("Capability Macaroon caveat check failed: %v", err),
				},
				ID: req.ID,
			}, nil
		}
	}

	// 4. Validate Input Schema (ADR-22)
	if err := spec.ValidateInput(params.Arguments); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32602,
				Message: fmt.Sprintf("Schema validation error: %v", err),
			},
			ID: req.ID,
		}, nil
	}

	// 5. RBAC Payload Sanitization & Field Redaction (ADR-08)
	sanitizedArgs, err := spec.SanitizeInput(params.Arguments)
	if err != nil {
		sanitizedArgs = params.Arguments
	}

	// 6. Forward to Sandbox Execution Callback
	if g.executor == nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Result: map[string]any{
				"status":          "validated",
				"sanitized_input": json.RawMessage(sanitizedArgs),
				"tool":           spec.Name,
			},
			ID: req.ID,
		}, nil
	}

	result, err := g.executor(ctx, spec, sanitizedArgs)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32000,
				Message: fmt.Sprintf("Tool execution error: %v", err),
			},
			ID: req.ID,
		}, nil
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}, nil
}

// IssueElicitationChallenge creates a stateless multi-round-trip challenge without locking transport sessions.
func (g *MCPGateway) IssueElicitationChallenge(toolName, prompt string, schema json.RawMessage, ttl time.Duration) *ElicitationChallenge {
	g.mu.Lock()
	defer g.mu.Unlock()

	id := fmt.Sprintf("chal_%d", time.Now().UnixNano())
	chal := &ElicitationChallenge{
		ChallengeID: id,
		ToolName:    toolName,
		Prompt:      prompt,
		Schema:      schema,
		ExpiresAt:   time.Now().Add(ttl),
	}
	g.challenges[id] = chal
	return chal
}

// VerifyElicitationResponse validates and consumes a stateless elicitation response.
func (g *MCPGateway) VerifyElicitationResponse(resp *ElicitationResponse) (*ElicitationChallenge, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	chal, exists := g.challenges[resp.ChallengeID]
	if !exists {
		return nil, ErrElicitationRequired
	}
	if time.Now().After(chal.ExpiresAt) {
		delete(g.challenges, resp.ChallengeID)
		return nil, ErrElicitationTimeout
	}
	delete(g.challenges, resp.ChallengeID)
	return chal, nil
}

func (g *MCPGateway) handleElicitationChallenge(req JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		ToolName string          `json:"tool_name"`
		Prompt   string          `json:"prompt"`
		Schema   json.RawMessage `json:"schema"`
		TTLSec   int             `json:"ttl_sec"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil || params.ToolName == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &JSONRPCError{Code: -32602, Message: "invalid elicitation_challenge params"},
			ID:      req.ID,
		}
	}
	ttl := 300 * time.Second
	if params.TTLSec > 0 {
		ttl = time.Duration(params.TTLSec) * time.Second
	}
	chal := g.IssueElicitationChallenge(params.ToolName, params.Prompt, params.Schema, ttl)
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  chal,
		ID:      req.ID,
	}
}

func (g *MCPGateway) handleElicitationRespond(req JSONRPCRequest) *JSONRPCResponse {
	var resp ElicitationResponse
	if err := json.Unmarshal(req.Params, &resp); err != nil || resp.ChallengeID == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &JSONRPCError{Code: -32602, Message: "invalid elicitation_respond params"},
			ID:      req.ID,
		}
	}
	chal, err := g.VerifyElicitationResponse(&resp)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &JSONRPCError{Code: -32001, Message: err.Error()},
			ID:      req.ID,
		}
	}
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result: map[string]any{
			"status":       "verified",
			"challenge_id": chal.ChallengeID,
			"tool_name":    chal.ToolName,
			"response":     resp.Response,
		},
		ID: req.ID,
	}
}

