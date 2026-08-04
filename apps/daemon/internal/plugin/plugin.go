package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DriverType defines the plugin execution driver mechanism per ADR-24.
type DriverType string

const (
	DriverTypeWasm       DriverType = "wasm"
	DriverTypeConnectRPC DriverType = "connect_rpc"
)

var (
	ErrInvalidManifest     = errors.New("plugin: invalid manifest format")
	ErrMissingID           = errors.New("plugin: missing required plugin id")
	ErrMissingEntrypoint   = errors.New("plugin: missing required entrypoint path")
	ErrUnsupportedDriver   = errors.New("plugin: unsupported driver type")
	ErrCapabilityViolation = errors.New("plugin: action violates capability grants")
)

// CapabilityGrants specifies the security isolation policy for a plugin.
type CapabilityGrants struct {
	Filesystem FilesystemGrants `yaml:"filesystem" json:"filesystem"`
	Network    NetworkGrants    `yaml:"network" json:"network"`
	IPC        IPCGrants        `yaml:"ipc" json:"ipc"`
}

type FilesystemGrants struct {
	AllowedReadPaths  []string `yaml:"allowed_read_paths" json:"allowed_read_paths"`
	AllowedWritePaths []string `yaml:"allowed_write_paths" json:"allowed_write_paths"`
}

type NetworkGrants struct {
	AllowedHosts []string `yaml:"allowed_hosts" json:"allowed_hosts"`
	AllowedPorts []int    `yaml:"allowed_ports" json:"allowed_ports"`
}

type IPCGrants struct {
	AllowedChannels []string `yaml:"allowed_channels" json:"allowed_channels"`
}

// PluginManifest represents a parsed PLUGIN.yaml manifest.
type PluginManifest struct {
	ID             string           `yaml:"id" json:"id"`
	Name           string           `yaml:"name" json:"name"`
	Version        string           `yaml:"version" json:"version"`
	Description    string           `yaml:"description" json:"description"`
	DriverType     DriverType       `yaml:"driver_type" json:"driver_type"`
	Entrypoint     string           `yaml:"entrypoint" json:"entrypoint"`
	Capabilities   CapabilityGrants `yaml:"capabilities" json:"capabilities"`
	TimeSliceMs    int              `yaml:"time_slice_ms" json:"time_slice_ms"`
	MaxMemoryBytes int64            `yaml:"max_memory_bytes" json:"max_memory_bytes"`
}

// ParsePluginManifest parses raw YAML bytes into a PluginManifest struct.
func ParsePluginManifest(data []byte) (*PluginManifest, error) {
	var m PluginManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		// Fallback to JSON
		if jErr := json.Unmarshal(data, &m); jErr != nil {
			return nil, fmt.Errorf("%w: YAML err: %v, JSON err: %v", ErrInvalidManifest, err, jErr)
		}
	}

	if err := ValidateManifest(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

// LoadPluginManifestFile reads and validates a PLUGIN.yaml file from disk.
func LoadPluginManifestFile(path string) (*PluginManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("plugin: failed to read manifest file: %w", err)
	}
	m, err := ParsePluginManifest(data)
	if err != nil {
		return nil, err
	}

	// Resolve entrypoint path relative to manifest if needed
	if !filepath.IsAbs(m.Entrypoint) {
		dir := filepath.Dir(path)
		m.Entrypoint = filepath.Join(dir, m.Entrypoint)
	}

	return m, nil
}

// ValidateManifest checks manifest fields for security and schema requirements.
func ValidateManifest(m *PluginManifest) error {
	if m == nil {
		return ErrInvalidManifest
	}

	m.ID = strings.TrimSpace(m.ID)
	if m.ID == "" {
		return ErrMissingID
	}

	m.Entrypoint = strings.TrimSpace(m.Entrypoint)
	if m.Entrypoint == "" {
		return ErrMissingEntrypoint
	}

	m.DriverType = DriverType(strings.ToLower(strings.TrimSpace(string(m.DriverType))))
	switch m.DriverType {
	case DriverTypeWasm, DriverTypeConnectRPC:
		// Valid driver
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedDriver, m.DriverType)
	}

	if m.TimeSliceMs <= 0 {
		m.TimeSliceMs = 1000 // Default 1 second time slice
	}

	if m.MaxMemoryBytes <= 0 {
		m.MaxMemoryBytes = 64 * 1024 * 1024 // Default 64MB memory limit
	}

	return nil
}

// ValidateFilesystemAccess checks if a path is permitted under the plugin capability grants.
func (m *PluginManifest) ValidateFilesystemAccess(path string, write bool) error {
	cleaned := filepath.Clean(path)

	allowedList := m.Capabilities.Filesystem.AllowedReadPaths
	if write {
		allowedList = m.Capabilities.Filesystem.AllowedWritePaths
	}

	for _, allowed := range allowedList {
		if allowed == "*" {
			return nil
		}
		allowedClean := filepath.Clean(allowed)
		if strings.HasPrefix(cleaned, allowedClean) {
			return nil
		}
	}

	return fmt.Errorf("%w: access to path %s (write=%v) denied", ErrCapabilityViolation, path, write)
}

// ValidateNetworkAccess checks if a host:port combination is permitted.
func (m *PluginManifest) ValidateNetworkAccess(host string, port int) error {
	hostAllowed := false
	for _, allowedHost := range m.Capabilities.Network.AllowedHosts {
		if allowedHost == "*" || strings.EqualFold(allowedHost, host) {
			hostAllowed = true
			break
		}
	}

	if !hostAllowed {
		return fmt.Errorf("%w: network host %s denied", ErrCapabilityViolation, host)
	}

	if len(m.Capabilities.Network.AllowedPorts) == 0 {
		return nil // All ports on allowed host permitted
	}

	for _, allowedPort := range m.Capabilities.Network.AllowedPorts {
		if allowedPort == 0 || allowedPort == port {
			return nil
		}
	}

	return fmt.Errorf("%w: network port %d on host %s denied", ErrCapabilityViolation, port, host)
}
