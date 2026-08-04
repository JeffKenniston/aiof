package mcp

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"aiof/internal/security"
)

func TestToolSpecParsingAndRegistry(t *testing.T) {
	rawSpec := []byte(`{
		"name": "file_reader",
		"description": "Reads file contents safely",
		"category": "filesystem",
		"version": "1.2.0",
		"side_effect_classification": "READ_ONLY",
		"input_schema": {
			"type": "object",
			"required": ["path"]
		},
		"sensitive_fields": ["path", "password"],
		"ebpf_lsm_boundaries": {
			"allowed_syscalls": ["read", "openat"],
			"allowed_paths": ["/workspace"]
		}
	}`)

	spec, err := ParseToolSpec(rawSpec)
	if err != nil {
		t.Fatalf("ParseToolSpec failed: %v", err)
	}

	if spec.Name != "file_reader" || spec.SideEffect != SideEffectReadOnly {
		t.Errorf("Unexpected parsed spec fields: name=%s, side_effect=%s", spec.Name, spec.SideEffect)
	}

	registry := NewToolRegistry()
	if err := registry.Register(spec); err != nil {
		t.Fatalf("Registry Register failed: %v", err)
	}

	// Duplicate registration should fail
	if err := registry.Register(spec); err == nil {
		t.Errorf("Expected duplicate registration error, got nil")
	}

	found, err := registry.Lookup("file_reader")
	if err != nil || found.Name != "file_reader" {
		t.Errorf("Registry Lookup failed: %v", err)
	}

	list := registry.List()
	if len(list) != 1 {
		t.Errorf("Expected 1 registered tool, got %d", len(list))
	}
}

func TestInputValidationAndSanitization(t *testing.T) {
	spec := &ToolSpec{
		Name:            "db_query",
		SideEffect:      SideEffectMutating,
		InputSchema:     json.RawMessage(`{"type":"object","required":["query"]}`),
		SensitiveFields: []string{"secret_token"},
	}

	// Invalid input missing required property
	invalidInput := json.RawMessage(`{"other": "value"}`)
	if err := spec.ValidateInput(invalidInput); err == nil {
		t.Errorf("Expected schema validation error for missing property, got nil")
	}

	// Valid input
	validInput := json.RawMessage(`{"query":"SELECT 1","secret_token":"my_secret","password":"123"}`)
	if err := spec.ValidateInput(validInput); err != nil {
		t.Errorf("Valid input failed validation: %v", err)
	}

	sanitized, err := spec.SanitizeInput(validInput)
	if err != nil {
		t.Fatalf("SanitizeInput failed: %v", err)
	}

	sanitizedStr := string(sanitized)
	if strings.Contains(sanitizedStr, "my_secret") || strings.Contains(sanitizedStr, "123") {
		t.Errorf("Sanitization failed to redact secrets: %s", sanitizedStr)
	}
	if !strings.Contains(sanitizedStr, "[REDACTED]") {
		t.Errorf("Expected [REDACTED] in sanitized output: %s", sanitizedStr)
	}
}

func TestMCPGatewayJSONRPCListAndCall(t *testing.T) {
	registry := NewToolRegistry()
	spec := &ToolSpec{
		Name:        "echo_tool",
		Description: "Echoes input back",
		SideEffect:  SideEffectReadOnly,
		InputSchema: json.RawMessage(`{"type":"object","required":["message"]}`),
	}
	_ = registry.Register(spec)

	executor := func(ctx context.Context, spec *ToolSpec, input json.RawMessage) (any, error) {
		var args struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(input, &args)
		return map[string]string{"echo": args.Message}, nil
	}

	gateway := NewMCPGateway(registry, nil, executor, nil)

	// Test tools/list
	listReq := []byte(`{"jsonrpc":"2.0","method":"tools/list","id":1}`)
	listResp, err := gateway.HandleJSONRPC(context.Background(), nil, listReq)
	if err != nil || listResp.Error != nil {
		t.Fatalf("tools/list failed: %v, error: %v", err, listResp.Error)
	}

	resMap, ok := listResp.Result.(map[string]any)
	if !ok || resMap["tools"] == nil {
		t.Fatalf("Invalid tools/list result structure: %v", listResp.Result)
	}

	// Test tools/call
	callReq := []byte(`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"echo_tool","arguments":{"message":"hello world"}},"id":2}`)
	callResp, err := gateway.HandleJSONRPC(context.Background(), nil, callReq)
	if err != nil || callResp.Error != nil {
		t.Fatalf("tools/call failed: %v, error: %v", err, callResp.Error)
	}

	callRes, ok := callResp.Result.(map[string]string)
	if !ok || callRes["echo"] != "hello world" {
		t.Errorf("Unexpected tools/call result: %v", callResp.Result)
	}
}

func TestMCPGatewayED25519SignatureAndMacaroonVerification(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ED25519 keys: %v", err)
	}

	pubHex := hex.EncodeToString(pub)

	auth, err := security.NewEnclaveKeyAuthority("enclave-01")
	if err != nil {
		t.Fatalf("Failed to create EnclaveKeyAuthority: %v", err)
	}

	mac, err := security.MintRootMacaroon(auth, "local", "token-01")
	if err != nil {
		t.Fatalf("Failed to mint Macaroon: %v", err)
	}

	mac, _ = mac.AddCaveat(security.Caveat{Predicate: "action = READ_ONLY"})

	registry := NewToolRegistry()
	_ = registry.Register(&ToolSpec{
		Name:       "secure_tool",
		SideEffect: SideEffectReadOnly,
	})

	gateway := NewMCPGateway(registry, auth, nil, nil)

	callReq := []byte(`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"secure_tool","arguments":{}},"id":10}`)
	sig := ed25519.Sign(priv, callReq)
	sigHex := hex.EncodeToString(sig)

	capToken := &CapabilityToken{
		Macaroon:  mac,
		Signature: sigHex,
		PublicKey: pubHex,
	}

	resp, err := gateway.HandleJSONRPC(context.Background(), capToken, callReq)
	if err != nil || resp.Error != nil {
		t.Fatalf("HandleJSONRPC with signature and macaroon failed: %v, error: %v", err, resp.Error)
	}
}

func TestMCPElicitationStateless(t *testing.T) {
	gateway := NewMCPGateway(nil, nil, nil, nil)

	// Step 1: Request elicitation challenge
	chalReq := []byte(`{"jsonrpc":"2.0","method":"tools/elicitation_challenge","params":{"tool_name":"git_commit","prompt":"Please confirm commit message","schema":{}},"id":1}`)
	resp, err := gateway.HandleJSONRPC(context.Background(), nil, chalReq)
	if err != nil || resp.Error != nil {
		t.Fatalf("elicitation_challenge failed: %v %v", err, resp.Error)
	}
	chal, ok := resp.Result.(*ElicitationChallenge)
	if !ok || chal.ChallengeID == "" {
		t.Fatalf("expected valid ElicitationChallenge, got %T", resp.Result)
	}

	// Step 2: Respond to challenge
	respReqPayload := fmt.Sprintf(`{"jsonrpc":"2.0","method":"tools/elicitation_respond","params":{"challenge_id":"%s","response":{"confirmed":true}},"id":2}`, chal.ChallengeID)
	resp2, err := gateway.HandleJSONRPC(context.Background(), nil, []byte(respReqPayload))
	if err != nil || resp2.Error != nil {
		t.Fatalf("elicitation_respond failed: %v %v", err, resp2.Error)
	}
	resMap, ok := resp2.Result.(map[string]any)
	if !ok || resMap["status"] != "verified" {
		t.Fatalf("expected status verified, got %v", resp2.Result)
	}
}

