package project

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"aiof/internal/eventmesh"
	"aiof/internal/governance"
)

var (
	ErrProjectNotFound   = errors.New("project: project not found")
	ErrInvalidPath       = errors.New("project: invalid or non-existent project directory path")
	ErrProjectAlreadyExists = errors.New("project: project already exists at specified path")
)

// Engine is the central thread-safe orchestrator for multi-modal projects across AIOF.
type Engine struct {
	logger      *slog.Logger
	costEngine  *governance.CostGovernanceEngine
	eventMesh   *eventmesh.EventMesh
	registry    *ModalityHandlerRegistry

	mu            sync.RWMutex
	projects      map[string]*Project  // Keyed by Project ID
	projectsByPath map[string]string   // Keyed by Normalized Absolute Path -> Project ID
	activeID      string
	recentProjects []ProjectMetadata
}

// NewEngine initializes the Universal Project Engine.
func NewEngine(logger *slog.Logger, costEngine *governance.CostGovernanceEngine, mesh *eventmesh.EventMesh) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	if costEngine == nil {
		costEngine = governance.NewCostGovernanceEngine(nil)
	}

	return &Engine{
		logger:         logger,
		costEngine:     costEngine,
		eventMesh:      mesh,
		registry:       NewModalityHandlerRegistry(logger),
		projects:       make(map[string]*Project),
		projectsByPath: make(map[string]string),
		recentProjects: make([]ProjectMetadata, 0),
	}
}

// CreateProject provisions a new project with specified modality, initializes budget caps, and persists state.
func (e *Engine) CreateProject(ctx context.Context, name, parentPath string, modality Modality) (*Project, error) {
	if name == "" {
		return nil, errors.New("project: project name cannot be empty")
	}
	if err := ValidateModality(modality); err != nil {
		return nil, err
	}

	absParent, err := filepath.Abs(parentPath)
	if err != nil {
		return nil, fmt.Errorf("project: invalid parent path: %w", err)
	}
	fullPath := filepath.Join(absParent, name)

	e.mu.Lock()
	if existingID, exists := e.projectsByPath[fullPath]; exists {
		e.mu.Unlock()
		return nil, fmt.Errorf("%w (ID: %s)", ErrProjectAlreadyExists, existingID)
	}
	e.mu.Unlock()

	// Ensure physical project directory exists
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return nil, fmt.Errorf("project: failed to create project directory: %w", err)
	}

	// Initialize project isolated config dir
	configDir := GetProjectConfigDir(fullPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("project: failed to create .aiof project config directory: %w", err)
	}

	projectID := fmt.Sprintf("proj-%x", sha256.Sum256([]byte(fullPath)))[:16]
	defaultConfig := GetDefaultConfigForModality(modality)

	proj := &Project{
		Metadata: ProjectMetadata{
			ID:          projectID,
			Name:        name,
			Description: fmt.Sprintf("Universal %s project", modality),
			Modality:    modality,
			RootPath:    fullPath,
			AuthorID:    "default-user",
			OrgID:       "default-org",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		Config: defaultConfig,
		State: ProjectState{
			ActiveBranch: "main",
			HealthScore:  1.0,
		},
	}

	// Provision Budget Cap in CostGovernanceEngine
	_ = e.costEngine.SetBudgetCap(projectID, proj.Config.SoftBudgetUSD, proj.Config.HardBudgetUSD, proj.Config.ForceLocalFallback)
	if b, found := e.costEngine.GetBudgetCap(projectID); found {
		proj.Budget = b
	}

	// Initialize strategy handler
	handler, _ := e.registry.GetHandler(modality)
	if handler != nil {
		_ = handler.Initialize(ctx, proj)
	}

	// Persist project configuration
	if err := e.persistProject(proj); err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.projects[projectID] = proj
	e.projectsByPath[fullPath] = projectID
	e.activeID = projectID
	e.updateRecentProjectsLocked(proj.Metadata)
	e.mu.Unlock()

	// Broadcast EventMesh event
	e.publishEvent("PROJECT_CREATED", projectID, map[string]interface{}{
		"name":     name,
		"modality": modality,
		"path":     fullPath,
	})

	return proj, nil
}

// LoadProject loads a project by path or ID, activating its modality profile.
func (e *Engine) LoadProject(ctx context.Context, pathOrID string) (*Project, error) {
	absPath, err := filepath.Abs(pathOrID)
	if err != nil {
		absPath = pathOrID
	}

	e.mu.Lock()
	// Check by ID first
	if proj, exists := e.projects[pathOrID]; exists {
		e.activeID = proj.Metadata.ID
		e.mu.Unlock()
		return proj, nil
	}
	// Check by Path next
	if id, exists := e.projectsByPath[absPath]; exists {
		proj := e.projects[id]
		e.activeID = id
		e.mu.Unlock()
		return proj, nil
	}
	e.mu.Unlock()

	// Verify directory exists on disk
	stat, err := os.Stat(absPath)
	if err != nil || !stat.IsDir() {
		return nil, fmt.Errorf("%w: '%s'", ErrInvalidPath, pathOrID)
	}

	// Attempt reading project.json from config directory
	configDir := GetProjectConfigDir(absPath)
	targetPath := filepath.Join(configDir, "project.json")
	data, err := os.ReadFile(targetPath)

	var proj Project
	if err == nil {
		if err := json.Unmarshal(data, &proj); err != nil {
			return nil, fmt.Errorf("project: failed to parse project.json: %w", err)
		}
	} else {
		// Auto-discover/create default Software project if project.json does not exist
		name := filepath.Base(absPath)
		projectID := fmt.Sprintf("proj-%x", sha256.Sum256([]byte(absPath)))[:16]
		proj = Project{
			Metadata: ProjectMetadata{
				ID:        projectID,
				Name:      name,
				Modality:  ModalitySoftwareEngineering,
				RootPath:  absPath,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			Config: GetDefaultConfigForModality(ModalitySoftwareEngineering),
			State: ProjectState{
				ActiveBranch: "main",
				HealthScore:  1.0,
			},
		}
		_ = os.MkdirAll(configDir, 0755)
		_ = e.persistProject(&proj)
	}

	// Attach / Sync budget cap
	_ = e.costEngine.SetBudgetCap(proj.Metadata.ID, proj.Config.SoftBudgetUSD, proj.Config.HardBudgetUSD, proj.Config.ForceLocalFallback)
	if b, found := e.costEngine.GetBudgetCap(proj.Metadata.ID); found {
		proj.Budget = b
	}

	// Initialize Modality Handler
	handler, _ := e.registry.GetHandler(proj.Metadata.Modality)
	if handler != nil {
		_ = handler.Initialize(ctx, &proj)
	}

	e.mu.Lock()
	e.projects[proj.Metadata.ID] = &proj
	e.projectsByPath[absPath] = proj.Metadata.ID
	e.activeID = proj.Metadata.ID
	e.updateRecentProjectsLocked(proj.Metadata)
	e.mu.Unlock()

	e.publishEvent("PROJECT_LOADED", proj.Metadata.ID, map[string]interface{}{
		"name":     proj.Metadata.Name,
		"modality": proj.Metadata.Modality,
		"path":     proj.Metadata.RootPath,
	})

	return &proj, nil
}

// GetActiveProject returns the currently active project.
func (e *Engine) GetActiveProject() (*Project, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.activeID == "" {
		return nil, ErrProjectNotFound
	}
	proj, exists := e.projects[e.activeID]
	if !exists {
		return nil, ErrProjectNotFound
	}
	return proj, nil
}

// SwitchModality mutates a project's operational modality profile on the fly.
func (e *Engine) SwitchModality(ctx context.Context, projectID string, newModality Modality) (*Project, error) {
	if err := ValidateModality(newModality); err != nil {
		return nil, err
	}

	e.mu.Lock()
	proj, exists := e.projects[projectID]
	if !exists {
		e.mu.Unlock()
		return nil, ErrProjectNotFound
	}
	oldModality := proj.Metadata.Modality
	e.mu.Unlock()

	if oldModality == newModality {
		return proj, nil
	}

	// Teardown old modality handler
	if oldHandler, err := e.registry.GetHandler(oldModality); err == nil {
		_ = oldHandler.Teardown(ctx, proj)
	}

	// Update metadata and default skills
	proj.Metadata.Modality = newModality
	proj.Metadata.UpdatedAt = time.Now().UTC()
	proj.Config = GetDefaultConfigForModality(newModality)

	// Initialize new modality handler
	if newHandler, err := e.registry.GetHandler(newModality); err == nil {
		_ = newHandler.Initialize(ctx, proj)
	}

	// Persist changes
	if err := e.persistProject(proj); err != nil {
		return nil, err
	}

	e.publishEvent("PROJECT_MODALITY_CHANGED", projectID, map[string]interface{}{
		"old_modality": oldModality,
		"new_modality": newModality,
	})

	return proj, nil
}

// GetRecentProjects returns the list of recent projects.
func (e *Engine) GetRecentProjects() []ProjectMetadata {
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make([]ProjectMetadata, len(e.recentProjects))
	copy(res, e.recentProjects)
	return res
}

func (e *Engine) persistProject(p *Project) error {
	configDir := GetProjectConfigDir(p.Metadata.RootPath)
	targetPath := filepath.Join(configDir, "project.json")
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("project: failed to encode project JSON: %w", err)
	}
	tmpFile := targetPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("project: failed to write temp project config: %w", err)
	}
	return os.Rename(tmpFile, targetPath)
}

func (e *Engine) updateRecentProjectsLocked(m ProjectMetadata) {
	filtered := make([]ProjectMetadata, 0, len(e.recentProjects)+1)
	filtered = append(filtered, m)
	for _, item := range e.recentProjects {
		if item.ID != m.ID && item.RootPath != m.RootPath {
			filtered = append(filtered, item)
		}
		if len(filtered) >= 10 {
			break
		}
	}
	e.recentProjects = filtered
}

func (e *Engine) publishEvent(eventType, projectID string, payload map[string]interface{}) {
	if e.eventMesh != nil {
		event := &eventmesh.Event{
			ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			Topic:     eventType,
			Source:    projectID,
			Priority:  eventmesh.PriorityDefault,
			Payload:   payload,
			Timestamp: time.Now().UTC(),
		}
		_ = e.eventMesh.Publish(event)
	}
}
