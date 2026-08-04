package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ============================================================================
// ADR-23: Portable Skill Framework with 3-Tier Progressive Disclosure
// ============================================================================

var (
	ErrInvalidFrontmatter = errors.New("skill: invalid or missing YAML frontmatter in SKILL.md")
	ErrSkillNotFound      = errors.New("skill: requested skill not found")
	ErrAssetNotFound      = errors.New("skill: requested asset file not found")
)

// SkillFrontmatter represents parsed metadata from SKILL.md header.
type SkillFrontmatter struct {
	Name         string   `yaml:"name" json:"name"`
	Description  string   `yaml:"description" json:"description"`
	Version      string   `yaml:"version" json:"version"`
	Author       string   `yaml:"author,omitempty" json:"author,omitempty"`
	AllowedTools []string `yaml:"allowed_tools,omitempty" json:"allowed_tools,omitempty"`
	Tags         []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Level 1: Discovery metadata (Loaded in-memory for zero-overhead routing).
type SkillDiscovery struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Version      string   `json:"version"`
	Author       string   `json:"author,omitempty"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Directory    string   `json:"directory"`
}

// Level 2: Instruction body (Loaded on demand when skill is selected).
type SkillInstruction struct {
	Discovery   SkillDiscovery   `json:"discovery"`
	Frontmatter SkillFrontmatter `json:"frontmatter"`
	Instruction string           `json:"instruction"`
}

// AssetCategory represents execution asset types under Level 3.
type AssetCategory string

const (
	AssetCategoryScripts    AssetCategory = "scripts"
	AssetCategoryTemplates  AssetCategory = "templates"
	AssetCategoryReferences AssetCategory = "references"
	AssetCategoryAssets     AssetCategory = "assets"
)

// Level 3: Execution Asset (Loaded dynamically on demand).
type SkillAsset struct {
	SkillName    string        `json:"skill_name"`
	RelativePath string        `json:"relative_path"`
	Category     AssetCategory `json:"category"`
	Content      []byte        `json:"-"`
	IsExecutable bool          `json:"is_executable"`
	Size         int64         `json:"size"`
}

// SkillManager implements the 3-Tier Progressive Disclosure runtime.
type SkillManager struct {
	mu          sync.RWMutex
	baseDir     string
	discoveries map[string]*SkillDiscovery
}

// NewSkillManager initializes a new skill runtime manager.
func NewSkillManager(baseDir string) *SkillManager {
	return &SkillManager{
		baseDir:     baseDir,
		discoveries: make(map[string]*SkillDiscovery),
	}
}

// ParseSKILLMD parses a SKILL.md content into frontmatter and body.
func ParseSKILLMD(data []byte) (*SkillFrontmatter, string, error) {
	str := string(data)
	trimmed := strings.TrimSpace(str)

	if !strings.HasPrefix(trimmed, "---") {
		return nil, "", ErrInvalidFrontmatter
	}

	parts := strings.SplitN(trimmed[3:], "---", 2)
	if len(parts) < 2 {
		return nil, "", ErrInvalidFrontmatter
	}

	rawYaml := strings.TrimSpace(parts[0])
	body := strings.TrimSpace(parts[1])

	frontmatter := &SkillFrontmatter{}
	lines := strings.Split(rawYaml, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])
		val = strings.Trim(val, "\"")

		switch key {
		case "name":
			frontmatter.Name = val
		case "description":
			frontmatter.Description = val
		case "version":
			frontmatter.Version = val
		case "author":
			frontmatter.Author = val
		case "allowed_tools":
			tools := strings.Split(strings.Trim(val, "[]"), ",")
			for _, t := range tools {
				if t := strings.TrimSpace(t); t != "" {
					frontmatter.AllowedTools = append(frontmatter.AllowedTools, t)
				}
			}
		case "tags":
			tags := strings.Split(strings.Trim(val, "[]"), ",")
			for _, t := range tags {
				if t := strings.TrimSpace(t); t != "" {
					frontmatter.Tags = append(frontmatter.Tags, t)
				}
			}
		}
	}

	if frontmatter.Name == "" {
		return nil, "", fmt.Errorf("%w: missing 'name' field", ErrInvalidFrontmatter)
	}

	return frontmatter, body, nil
}

// Level 1 (Discovery): Scans directory and populates in-memory discovery registry.
func (m *SkillManager) ScanDirectory(dir string) ([]SkillDiscovery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if dir != "" {
		m.baseDir = dir
	}

	if m.baseDir == "" {
		return nil, errors.New("skill: base directory not configured")
	}

	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("skill: failed to read skill base dir: %w", err)
	}

	var results []SkillDiscovery

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(m.baseDir, entry.Name())
		skillPath := filepath.Join(skillDir, "SKILL.md")

		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue // Skip directories missing SKILL.md
		}

		frontmatter, _, err := ParseSKILLMD(data)
		if err != nil {
			continue
		}

		disc := SkillDiscovery{
			Name:         frontmatter.Name,
			Description:  frontmatter.Description,
			Version:      frontmatter.Version,
			Author:       frontmatter.Author,
			AllowedTools: frontmatter.AllowedTools,
			Tags:         frontmatter.Tags,
			Directory:    skillDir,
		}

		m.discoveries[frontmatter.Name] = &disc
		results = append(results, disc)
	}

	return results, nil
}

// Level 1: GetDiscovery returns in-memory light metadata for zero-overhead routing.
func (m *SkillManager) GetDiscovery(name string) (*SkillDiscovery, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	disc, exists := m.discoveries[name]
	if !exists {
		return nil, false
	}
	copyDisc := *disc
	return &copyDisc, true
}

// Level 2 (Instruction): Loads full SKILL.md instructions on demand when selected.
func (m *SkillManager) LoadInstruction(name string) (*SkillInstruction, error) {
	disc, exists := m.GetDiscovery(name)
	if !exists {
		return nil, ErrSkillNotFound
	}

	skillPath := filepath.Join(disc.Directory, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("skill: failed to read SKILL.md for %s: %w", name, err)
	}

	frontmatter, body, err := ParseSKILLMD(data)
	if err != nil {
		return nil, err
	}

	return &SkillInstruction{
		Discovery:   *disc,
		Frontmatter: *frontmatter,
		Instruction: body,
	}, nil
}

// Level 3 (Execution Assets): Loads supporting assets (scripts, templates, references, assets) dynamically.
func (m *SkillManager) LoadAsset(skillName, relPath string) (*SkillAsset, error) {
	disc, exists := m.GetDiscovery(skillName)
	if !exists {
		return nil, ErrSkillNotFound
	}

	cleanRel := filepath.Clean(relPath)
	if strings.HasPrefix(cleanRel, "..") {
		return nil, errors.New("skill: asset path traversal attempted")
	}

	fullPath := filepath.Join(disc.Directory, cleanRel)
	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		return nil, ErrAssetNotFound
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("skill: failed to read asset file: %w", err)
	}

	category := AssetCategoryAssets
	firstPart := strings.Split(cleanRel, string(filepath.Separator))[0]
	switch firstPart {
	case "scripts":
		category = AssetCategoryScripts
	case "templates":
		category = AssetCategoryTemplates
	case "references":
		category = AssetCategoryReferences
	}

	isExec := (info.Mode() & 0111) != 0

	return &SkillAsset{
		SkillName:    skillName,
		RelativePath: cleanRel,
		Category:     category,
		Content:      data,
		IsExecutable: isExec,
		Size:         info.Size(),
	}, nil
}

// ListAssets returns relative paths of all available Level 3 execution assets for a skill.
func (m *SkillManager) ListAssets(skillName string) ([]string, error) {
	disc, exists := m.GetDiscovery(skillName)
	if !exists {
		return nil, ErrSkillNotFound
	}

	var assets []string
	subdirs := []string{"scripts", "templates", "references", "assets"}

	for _, sub := range subdirs {
		targetDir := filepath.Join(disc.Directory, sub)
		_ = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(disc.Directory, path)
			if relErr == nil {
				assets = append(assets, rel)
			}
			return nil
		})
	}

	return assets, nil
}
