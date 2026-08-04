package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// ADR-21: Fabricator Agent Authoring Pipeline for Ephemeral Swarms
// ============================================================================

type FabricatorPipeline struct {
	defaultPrimaryTier string
	defaultTTLSec      int
}

func NewFabricatorPipeline(defaultPrimaryTier string, defaultTTLSec int) *FabricatorPipeline {
	if defaultPrimaryTier == "" {
		defaultPrimaryTier = "frontier"
	}
	if defaultTTLSec <= 0 {
		defaultTTLSec = 3600 // 1 hour default TTL for ephemeral agents
	}
	return &FabricatorPipeline{
		defaultPrimaryTier: defaultPrimaryTier,
		defaultTTLSec:      defaultTTLSec,
	}
}

// FabricateEphemeralAgent authors a specialized, ephemeral AgentManifest and tailored system prompt
// for complex tasks outside static catalog taxonomies per ADR-21.
func (f *FabricatorPipeline) FabricateEphemeralAgent(
	ctx context.Context,
	taskDescription string,
	allowedTools []string,
) (*AgentManifest, string, error) {
	if strings.TrimSpace(taskDescription) == "" {
		return nil, "", fmt.Errorf("agent/fabricator: task description cannot be empty")
	}

	now := time.Now().UTC()
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(taskDescription+now.String())))[:12]
	agentName := fmt.Sprintf("ephemeral-fabricator-%s", hash)

	// Determine domain and role from task keywords
	role := "Ephemeral Task Specialist"
	cognitiveStyle := "Surgical execution with strict capability sandboxing"
	domain := "general"

	lowerTask := strings.ToLower(taskDescription)
	switch {
	case strings.Contains(lowerTask, "code") || strings.Contains(lowerTask, "refactor") || strings.Contains(lowerTask, "ast"):
		role = "Software Engineering Specialist"
		cognitiveStyle = "Formal AST analysis, test-driven refactoring, and invariant verification"
		domain = "software-engineering"
	case strings.Contains(lowerTask, "security") || strings.Contains(lowerTask, "cve") || strings.Contains(lowerTask, "sbom"):
		role = "Security Audit Specialist"
		cognitiveStyle = "Zero-trust verification, vulnerability scanning, and cryptographic attestation"
		domain = "cybersecurity"
	case strings.Contains(lowerTask, "finance") || strings.Contains(lowerTask, "cost") || strings.Contains(lowerTask, "token"):
		role = "Financial Governance Specialist"
		cognitiveStyle = "Cost optimization, SLA tracking, and transaction auditing"
		domain = "finance"
	}

	// Build Macaroon caveats to enforce restricted capability scope
	caveats := []string{
		fmt.Sprintf("time < %s", now.Add(time.Duration(f.defaultTTLSec)*time.Second).Format(time.RFC3339)),
		"max_depth = 1",
		fmt.Sprintf("enclave_domain = %s", domain),
	}

	manifest := &AgentManifest{
		Metadata: AgentMetadata{
			Name:        agentName,
			Version:     "1.0.0-ephemeral",
			Description: fmt.Sprintf("Dynamically fabricated ephemeral agent for: %s", taskDescription),
			Author:      "FabricatorPipeline/aiof",
			Domain:      domain,
			CreatedAt:   now,
		},
		Identity: AgentIdentity{
			Role:           role,
			Persona:        fmt.Sprintf("Specialized ephemeral agent dynamically instantiated to solve: %s", taskDescription),
			CognitiveStyle: cognitiveStyle,
			Tone:           "Objective, concise, and structured",
		},
		Capabilities: AgentCapabilities{
			AllowedTools:       allowedTools,
			SystemCalls:        []string{"read", "exec_sandboxed"},
			NetworkEndpoints:   []string{},
			MaxDelegationDepth: 1, // Restrict multi-hop delegation for ephemeral agents
		},
		ModelPreferences: ModelPreferences{
			PrimaryTier:         f.defaultPrimaryTier,
			FallbackTier:        "local-7b",
			Temperature:         0.2,
			TopP:                0.95,
			ContextWindowTokens: 128000,
		},
		MemoryIsolation: MemoryIsolation{
			Boundary:    "session", // Ephemeral session boundary isolates state
			ReadScopes:  []string{"session_read"},
			WriteScopes: []string{"session_write"},
		},
		MacaroonCaveats: MacaroonCaveats{
			PreBoundCaveats: caveats,
		},
		EphemeralTTLSec: f.defaultTTLSec,
	}

	systemPrompt := f.BuildTailoredSystemPrompt(manifest, taskDescription)
	return manifest, systemPrompt, nil
}

// BuildTailoredSystemPrompt constructs an explicit system prompt embedding identity, allowed capabilities, and scope rules.
func (f *FabricatorPipeline) BuildTailoredSystemPrompt(manifest *AgentManifest, taskDescription string) string {
	toolsList := "None"
	if len(manifest.Capabilities.AllowedTools) > 0 {
		toolsList = strings.Join(manifest.Capabilities.AllowedTools, ", ")
	}

	caveatsList := "None"
	if len(manifest.MacaroonCaveats.PreBoundCaveats) > 0 {
		caveatsList = strings.Join(manifest.MacaroonCaveats.PreBoundCaveats, "; ")
	}

	return fmt.Sprintf(`You are %s (%s), an ephemeral agent dynamically fabricated by aiof for a specialized task.

=== IDENTITY & PERSONA ===
- Role: %s
- Persona: %s
- Cognitive Style: %s
- Domain: %s

=== ASSIGNED TASK ===
%s

=== CAPABILITY BOUNDARIES & RESTRICTIONS ===
- Allowed Tools: %s
- Memory Isolation Boundary: %s
- Max Delegation Depth: %d
- Macaroon Capability Caveats: %s
- Ephemeral TTL: %d seconds

=== OPERATIONAL INSTRUCTIONS ===
1. Focus strictly on completing the assigned task within your capability boundaries.
2. Do not attempt tool calls outside your allowed tools list.
3. Maintain zero-trust security and provide structured, verifiable outputs.
`,
		manifest.Metadata.Name,
		manifest.Metadata.Version,
		manifest.Identity.Role,
		manifest.Identity.Persona,
		manifest.Identity.CognitiveStyle,
		manifest.Metadata.Domain,
		taskDescription,
		toolsList,
		manifest.MemoryIsolation.Boundary,
		manifest.Capabilities.MaxDelegationDepth,
		caveatsList,
		manifest.EphemeralTTLSec,
	)
}
