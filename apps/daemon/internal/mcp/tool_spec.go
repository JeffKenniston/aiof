package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ============================================================================
// ADR-22: Declarative Tool Specification (TOOL.json)
// ============================================================================

type SideEffectClass string

const (
	SideEffectReadOnly       SideEffectClass = "READ_ONLY"
	SideEffectMutating       SideEffectClass = "MUTATING"
	SideEffectSystemMutating SideEffectClass = "SYSTEM_MUTATING"
)

// EBPFBoundarySpec defines eBPF LSM kernel boundary parameters required per ADR-12 and ADR-22.
type EBPFBoundarySpec struct {
	AllowedSyscalls     []string `json:"allowed_syscalls"`
	AllowedPaths        []string `json:"allowed_paths"`
	AllowedNetworkHosts []string `json:"allowed_network_hosts"`
}

// ToolSpec represents a parsed declarative TOOL.json specification.
type ToolSpec struct {
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Category        string           `json:"category"`
	Version         string           `json:"version"`
	SideEffect      SideEffectClass  `json:"side_effect_classification"`
	InputSchema     json.RawMessage  `json:"input_schema"`
	OutputSchema    json.RawMessage  `json:"output_schema"`
	EBPFBoundaries  EBPFBoundarySpec `json:"ebpf_lsm_boundaries"`
	RequiredRole    string           `json:"required_role,omitempty"`
	SensitiveFields []string         `json:"sensitive_fields,omitempty"` // Fields requiring RBAC redaction
}

var (
	ErrInvalidToolSpec  = errors.New("mcp: invalid TOOL.json specification")
	ErrToolExists       = errors.New("mcp: tool already registered")
	ErrToolNotFound     = errors.New("mcp: requested tool not found in registry")
	ErrSchemaValidation = errors.New("mcp: tool input payload failed schema validation")
)

// ParseToolSpec parses a raw JSON byte slice into a ToolSpec struct and validates required fields.
func ParseToolSpec(data []byte) (*ToolSpec, error) {
	var spec ToolSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToolSpec, err)
	}

	if spec.Name == "" {
		return nil, fmt.Errorf("%w: tool name cannot be empty", ErrInvalidToolSpec)
	}
	if spec.Version == "" {
		spec.Version = "1.0.0"
	}
	if spec.SideEffect == "" {
		spec.SideEffect = SideEffectReadOnly
	}

	return &spec, nil
}

// ToolRegistry manages registered declarative tool specifications in memory.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*ToolSpec
}

// NewToolRegistry initializes a thread-safe ToolRegistry instance.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolSpec),
	}
}

// Register adds a new ToolSpec to the registry.
func (r *ToolRegistry) Register(spec *ToolSpec) error {
	if spec == nil || spec.Name == "" {
		return ErrInvalidToolSpec
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[spec.Name]; exists {
		return fmt.Errorf("%w: %s", ErrToolExists, spec.Name)
	}

	r.tools[spec.Name] = spec
	return nil
}

// Lookup retrieves a ToolSpec by tool name.
func (r *ToolRegistry) Lookup(name string) (*ToolSpec, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	spec, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	return spec, nil
}

// List returns a slice of all registered ToolSpec instances.
func (r *ToolRegistry) List() []*ToolSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ToolSpec, 0, len(r.tools))
	for _, spec := range r.tools {
		result = append(result, spec)
	}
	return result
}

// ValidateInput performs structural field validation against the registered tool input schema.
func (spec *ToolSpec) ValidateInput(rawInput json.RawMessage) error {
	if len(spec.InputSchema) == 0 || string(spec.InputSchema) == "{}" {
		return nil // No input schema constraints
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rawInput, &payload); err != nil {
		return fmt.Errorf("%w: input must be a JSON object: %v", ErrSchemaValidation, err)
	}

	// Basic schema required-field checking from InputSchema
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(spec.InputSchema, &schemaMap); err == nil {
		if reqFields, ok := schemaMap["required"].([]interface{}); ok {
			for _, rf := range reqFields {
				fieldName, isStr := rf.(string)
				if isStr {
					if _, present := payload[fieldName]; !present {
						return fmt.Errorf("%w: missing required property '%s'", ErrSchemaValidation, fieldName)
					}
				}
			}
		}
	}

	return nil
}

// SanitizeInput redacts sensitive fields defined in spec.SensitiveFields or common key patterns.
func (spec *ToolSpec) SanitizeInput(rawInput json.RawMessage) (json.RawMessage, error) {
	if len(rawInput) == 0 {
		return rawInput, nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rawInput, &payload); err != nil {
		return rawInput, nil // Pass-through if not object
	}

	sensitiveSet := make(map[string]bool)
	for _, sf := range spec.SensitiveFields {
		sensitiveSet[strings.ToLower(sf)] = true
	}

	var redactMap func(m map[string]interface{})
	redactMap = func(m map[string]interface{}) {
		for k, v := range m {
			lk := strings.ToLower(k)
			if sensitiveSet[lk] || isSensitiveKeyName(lk) {
				m[k] = "[REDACTED]"
			} else if childMap, ok := v.(map[string]interface{}); ok {
				redactMap(childMap)
			}
		}
	}

	redactMap(payload)
	return json.Marshal(payload)
}

func isSensitiveKeyName(key string) bool {
	return strings.Contains(key, "password") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "token") ||
		strings.Contains(key, "private_key") ||
		strings.Contains(key, "ssn") ||
		strings.Contains(key, "credit_card")
}
