package project

import (
	"context"
	"fmt"
)

// ModalityHandler defines the pluggable lifecycle strategy for a specific workload modality.
type ModalityHandler interface {
	// Modality returns the identifier handled by this strategy.
	Modality() Modality
	// Initialize provisions domain-specific resources and sets default skills/tools when loading a project.
	Initialize(ctx context.Context, p *Project) error
	// ProvisionSkills returns the set of active skills required for this modality.
	ProvisionSkills(ctx context.Context, p *Project) ([]string, error)
	// Teardown cleans up modality-specific background jobs or sandboxes before switching project state.
	Teardown(ctx context.Context, p *Project) error
}

// GetDefaultConfigForModality returns baseline skills, MCP tool specs, and budget defaults for a modality.
func GetDefaultConfigForModality(m Modality) ProjectConfig {
	switch m {
	case ModalitySoftwareEngineering:
		return ProjectConfig{
			ActiveSkills:       []string{"plann", "swebench", "ast-parser"},
			ActivePlugins:      []string{"git-integration", "sandbox-cow"},
			MCPTools:           []string{"code_edit", "run_terminal", "view_file", "grep_search"},
			SoftBudgetUSD:      100.0,
			HardBudgetUSD:      250.0,
			ForceLocalFallback: true,
		}
	case ModalityWebFullstack:
		return ProjectConfig{
			ActiveSkills:       []string{"plann", "web-developer", "vite-orchestration"},
			ActivePlugins:      []string{"dev-server-runner", "ui-wireframe-canvas"},
			MCPTools:           []string{"code_edit", "run_terminal", "read_url_content", "view_file"},
			SoftBudgetUSD:      100.0,
			HardBudgetUSD:      200.0,
			ForceLocalFallback: true,
		}
	case ModalityMediaCreative:
		return ProjectConfig{
			ActiveSkills:       []string{"media-processing", "visual-reasoning", "storyboarding"},
			ActivePlugins:      []string{"gemini-multimodal", "spatial-canvas-media"},
			MCPTools:           []string{"generate_image", "view_file", "write_to_file"},
			SoftBudgetUSD:      150.0,
			HardBudgetUSD:      300.0,
			ForceLocalFallback: false,
		}
	case ModalityAcademicResearch:
		return ProjectConfig{
			ActiveSkills:       []string{"plann", "academic-research", "latex-formatter", "graphrag-synthesis"},
			ActivePlugins:      []string{"bitemporal-toki", "graphrag-engine"},
			MCPTools:           []string{"search_web", "read_url_content", "view_file", "write_to_file"},
			SoftBudgetUSD:      80.0,
			HardBudgetUSD:      150.0,
			ForceLocalFallback: true,
		}
	case ModalityDataScienceML:
		return ProjectConfig{
			ActiveSkills:       []string{"python-repl", "data-analysis", "ml-evaluator"},
			ActivePlugins:      []string{"levc-vectors", "repl-sandbox"},
			MCPTools:           []string{"run_command", "view_file", "write_to_file", "grep_search"},
			SoftBudgetUSD:      200.0,
			HardBudgetUSD:      500.0,
			ForceLocalFallback: true,
		}
	case ModalityEnterpriseBusiness:
		return ProjectConfig{
			ActiveSkills:       []string{"plann", "financial-governance", "sla-tracker"},
			ActivePlugins:      []string{"mcp-gateway-rbac", "cost-ledger"},
			MCPTools:           []string{"search_web", "view_file", "write_to_file"},
			SoftBudgetUSD:      500.0,
			HardBudgetUSD:      1000.0,
			ForceLocalFallback: true,
		}
	case ModalityDevOpsIaC:
		return ProjectConfig{
			ActiveSkills:       []string{"iac-engine", "security-sbom", "ebpf-observability"},
			ActivePlugins:      []string{"terraform-runner", "attestation-verifier"},
			MCPTools:           []string{"run_command", "view_file", "grep_search"},
			SoftBudgetUSD:      150.0,
			HardBudgetUSD:      300.0,
			ForceLocalFallback: true,
		}
	case ModalityDocumentation:
		return ProjectConfig{
			ActiveSkills:       []string{"technical-writer", "dual-index-search"},
			ActivePlugins:      []string{"graphrag-engine", "rich-doc-canvas"},
			MCPTools:           []string{"view_file", "write_to_file", "grep_search"},
			SoftBudgetUSD:      50.0,
			HardBudgetUSD:      100.0,
			ForceLocalFallback: true,
		}
	case ModalityEducationLearning:
		return ProjectConfig{
			ActiveSkills:       []string{"hcs-memory-tutor", "interactive-walkthrough"},
			ActivePlugins:      []string{"hcs-vector-store", "canvas-walkthrough"},
			MCPTools:           []string{"view_file", "ask_question"},
			SoftBudgetUSD:      30.0,
			HardBudgetUSD:      75.0,
			ForceLocalFallback: true,
		}
	case ModalityComposableHybrid:
		fallthrough
	default:
		return ProjectConfig{
			ActiveSkills:       []string{"plann", "general-assistant"},
			ActivePlugins:      []string{"mcp-gateway", "bitemporal-toki"},
			MCPTools:           []string{"view_file", "write_to_file", "run_command", "grep_search", "search_web"},
			SoftBudgetUSD:      100.0,
			HardBudgetUSD:      250.0,
			ForceLocalFallback: true,
		}
	}
}

// ValidateModality returns an error if the modality string is invalid.
func ValidateModality(m Modality) error {
	for _, valid := range AllModalities {
		if m == valid {
			return nil
		}
	}
	return fmt.Errorf("project: unsupported modality '%s'", m)
}
