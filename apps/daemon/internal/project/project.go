package project

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"time"

	"aiof/internal/config"
	"aiof/internal/governance"
)

// Modality identifies the primary operational profile and workload type of a project.
type Modality string

const (
	ModalitySoftwareEngineering Modality = "software_engineering"
	ModalityWebFullstack        Modality = "web_fullstack"
	ModalityMediaCreative       Modality = "media_creative"
	ModalityAcademicResearch    Modality = "academic_research"
	ModalityDataScienceML       Modality = "datascience_ml"
	ModalityEnterpriseBusiness  Modality = "enterprise_business"
	ModalityDevOpsIaC           Modality = "devops_iac"
	ModalityDocumentation       Modality = "documentation_writing"
	ModalityEducationLearning   Modality = "education_learning"
	ModalityComposableHybrid    Modality = "composable_hybrid"
)

// AllModalities lists all 10 supported project modalities ensuring 100% workload coverage.
var AllModalities = []Modality{
	ModalitySoftwareEngineering,
	ModalityWebFullstack,
	ModalityMediaCreative,
	ModalityAcademicResearch,
	ModalityDataScienceML,
	ModalityEnterpriseBusiness,
	ModalityDevOpsIaC,
	ModalityDocumentation,
	ModalityEducationLearning,
	ModalityComposableHybrid,
}

// ProjectMetadata encapsulates static identity, ownership, and modality metadata.
type ProjectMetadata struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Modality    Modality  `json:"modality"`
	RootPath    string    `json:"root_path"`
	AuthorID    string    `json:"author_id,omitempty"`
	OrgID       string    `json:"org_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectConfig holds declarative operational settings, skill packages, MCP tools, and financial budget caps.
type ProjectConfig struct {
	ActiveSkills       []string               `json:"active_skills"`
	ActivePlugins      []string               `json:"active_plugins"`
	MCPTools           []string               `json:"mcp_tools"`
	SoftBudgetUSD      float64                `json:"soft_budget_usd"`
	HardBudgetUSD      float64                `json:"hard_budget_usd"`
	ForceLocalFallback bool                   `json:"force_local_fallback"`
	CustomSettings     map[string]interface{} `json:"custom_settings,omitempty"`
}

// ProjectState tracks active dynamic execution context, snapshots, and health telemetry.
type ProjectState struct {
	ActiveBranch         string    `json:"active_branch"`
	BitemporalSnapshotID string    `json:"bitemporal_snapshot_id,omitempty"`
	ActiveSandboxID      string    `json:"active_sandbox_id,omitempty"`
	CanvasIDs            []string  `json:"canvas_ids,omitempty"`
	HealthScore          float64   `json:"health_score"`
	LastIndexedAt        time.Time `json:"last_indexed_at,omitempty"`
}

// Project is the universal domain model representing any workload in AIOF.
type Project struct {
	Metadata ProjectMetadata       `json:"metadata"`
	Config   ProjectConfig         `json:"config"`
	State    ProjectState          `json:"state"`
	Budget   *governance.BudgetCap `json:"budget,omitempty"`
}

// GetProjectConfigDir computes the isolated hashed global configuration directory for a project path.
func GetProjectConfigDir(projectDir string) string {
	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		absPath = projectDir
	}
	hash := sha256.Sum256([]byte(absPath))
	return filepath.Join(config.GetConfigDir(), "projects", fmt.Sprintf("%x", hash))
}
