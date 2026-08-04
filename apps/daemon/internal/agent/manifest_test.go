package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
)

func TestParseAgentManifest(t *testing.T) {
	jsonManifest := `{
		"metadata": {
			"name": "code-refactor-agent",
			"version": "1.0.0",
			"description": "Specialized multi-file code refactoring agent",
			"author": "AIOF Team"
		},
		"identity": {
			"role": "Software Architecture Refactorer",
			"persona": "Meticulous Go engineer focused on AST safety"
		},
		"capabilities": {
			"allowed_tools": ["ast_parse", "diff_apply", "run_tests"],
			"max_delegation_depth": 2
		},
		"model_preferences": {
			"primary_tier": "frontier",
			"temperature": 0.1,
			"top_p": 0.9
		},
		"memory_isolation": {
			"boundary": "session",
			"read_scopes": ["code_read"],
			"write_scopes": ["code_write"]
		}
	}`

	manifest, err := ParseAgentManifest([]byte(jsonManifest))
	if err != nil {
		t.Fatalf("Failed to parse valid manifest: %v", err)
	}

	if manifest.Metadata.Name != "code-refactor-agent" {
		t.Errorf("Expected name 'code-refactor-agent', got '%s'", manifest.Metadata.Name)
	}
	if manifest.Identity.Role != "Software Architecture Refactorer" {
		t.Errorf("Expected role 'Software Architecture Refactorer', got '%s'", manifest.Identity.Role)
	}
	if manifest.MemoryIsolation.Boundary != "session" {
		t.Errorf("Expected boundary 'session', got '%s'", manifest.MemoryIsolation.Boundary)
	}
}

func TestManifestValidationErrors(t *testing.T) {
	invalidJSON := `{"metadata": {"name": ""}, "identity": {"role": "Tester"}}`
	_, err := ParseAgentManifest([]byte(invalidJSON))
	if err == nil {
		t.Errorf("Expected error for missing name, got nil")
	}

	invalidBoundary := `{
		"metadata": {"name": "test", "version": "1.0"},
		"identity": {"role": "Tester"},
		"memory_isolation": {"boundary": "invalid_boundary_name"}
	}`
	_, err = ParseAgentManifest([]byte(invalidBoundary))
	if err == nil {
		t.Errorf("Expected error for invalid boundary, got nil")
	}
}

func TestAgentPackageExportImport(t *testing.T) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 keys: %v", err)
	}

	manifest := &AgentManifest{
		Metadata: AgentMetadata{
			Name:        "security-scanner",
			Version:     "2.6.0",
			Description: "Automated vulnerability and SBOM scanner",
		},
		Identity: AgentIdentity{
			Role: "Cybersecurity Auditor",
		},
		Capabilities: AgentCapabilities{
			AllowedTools:       []string{"sbom_scan", "cve_lookup"},
			MaxDelegationDepth: 1,
		},
		ModelPreferences: ModelPreferences{
			PrimaryTier: "local-7b",
			Temperature: 0.1,
		},
		MemoryIsolation: MemoryIsolation{
			Boundary: "enclave",
		},
	}

	pkgBytes, err := ExportAgentPackage(manifest, privKey, pubKey)
	if err != nil {
		t.Fatalf("ExportAgentPackage failed: %v", err)
	}

	if len(pkgBytes) == 0 {
		t.Fatalf("Exported package bytes is empty")
	}

	// Verify Import with trusted public key
	imported, err := ImportAgentPackage(pkgBytes, pubKey)
	if err != nil {
		t.Fatalf("ImportAgentPackage failed: %v", err)
	}

	if imported.Metadata.Name != "security-scanner" {
		t.Errorf("Expected imported name 'security-scanner', got '%s'", imported.Metadata.Name)
	}
	if imported.MemoryIsolation.Boundary != "enclave" {
		t.Errorf("Expected imported boundary 'enclave', got '%s'", imported.MemoryIsolation.Boundary)
	}

	// Test signature verification failure with wrong key
	_, wrongPriv, err := ed25519.GenerateKey(rand.Reader)
	wrongPub := wrongPriv.Public().(ed25519.PublicKey)

	_, err = ImportAgentPackage(pkgBytes, wrongPub)
	if err == nil {
		t.Errorf("Expected signature verification failure with wrong public key, got nil error")
	}
}

func TestFabricatorPipeline(t *testing.T) {
	pipe := NewFabricatorPipeline("frontier", 1800)
	ctx := context.Background()

	task := "Perform AST analysis and refactor Golang HTTP middleware for error handling"
	tools := []string{"ast_parse", "code_edit"}

	manifest, prompt, err := pipe.FabricateEphemeralAgent(ctx, task, tools)
	if err != nil {
		t.Fatalf("FabricateEphemeralAgent failed: %v", err)
	}

	if !strings.HasPrefix(manifest.Metadata.Name, "ephemeral-fabricator-") {
		t.Errorf("Expected name prefix 'ephemeral-fabricator-', got '%s'", manifest.Metadata.Name)
	}
	if manifest.Identity.Role != "Software Engineering Specialist" {
		t.Errorf("Expected role 'Software Engineering Specialist', got '%s'", manifest.Identity.Role)
	}
	if manifest.EphemeralTTLSec != 1800 {
		t.Errorf("Expected TTL 1800, got %d", manifest.EphemeralTTLSec)
	}

	if !strings.Contains(prompt, "ASSIGNED TASK") || !strings.Contains(prompt, task) {
		t.Errorf("Tailored prompt does not contain assigned task description")
	}
}
