package agent

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// ============================================================================
// ADR-21: Declarative Autonomous Agent Framework & Ephemeral Swarm Lifecycle
// ============================================================================

var (
	ErrInvalidManifest      = errors.New("agent/manifest: invalid AGENT.yaml manifest structure")
	ErrMissingRequiredField = errors.New("agent/manifest: missing required manifest field")
	ErrInvalidBoundary      = errors.New("agent/manifest: invalid memory isolation boundary")
	ErrCoractedPackage      = errors.New("agent/manifest: invalid or corrupted .agent package archive")
	ErrSignatureMismatch    = errors.New("agent/manifest: ed25519 signature verification failed for .agent package")
)

type AgentMetadata struct {
	Name        string    `json:"name" yaml:"name"`
	Version     string    `json:"version" yaml:"version"`
	Description string    `json:"description" yaml:"description"`
	Author      string    `json:"author,omitempty" yaml:"author,omitempty"`
	Domain      string    `json:"domain,omitempty" yaml:"domain,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty" yaml:"created_at,omitempty"`
}

type AgentIdentity struct {
	Role           string `json:"role" yaml:"role"`
	Persona        string `json:"persona" yaml:"persona"`
	CognitiveStyle string `json:"cognitive_style,omitempty" yaml:"cognitive_style,omitempty"`
	Tone           string `json:"tone,omitempty" yaml:"tone,omitempty"`
}

type AgentCapabilities struct {
	AllowedTools       []string `json:"allowed_tools" yaml:"allowed_tools"`
	SystemCalls        []string `json:"system_calls,omitempty" yaml:"system_calls,omitempty"`
	NetworkEndpoints   []string `json:"network_endpoints,omitempty" yaml:"network_endpoints,omitempty"`
	MaxDelegationDepth int      `json:"max_delegation_depth" yaml:"max_delegation_depth"`
}

type ModelPreferences struct {
	PrimaryTier         string  `json:"primary_tier" yaml:"primary_tier"` // "frontier", "local-7b", "local-14b", "npu-0.5b"
	FallbackTier        string  `json:"fallback_tier,omitempty" yaml:"fallback_tier,omitempty"`
	Temperature         float64 `json:"temperature" yaml:"temperature"`
	TopP                float64 `json:"top_p" yaml:"top_p"`
	ContextWindowTokens int     `json:"context_window_tokens,omitempty" yaml:"context_window_tokens,omitempty"`
}

type MemoryIsolation struct {
	Boundary    string   `json:"boundary" yaml:"boundary"` // "session", "agent", "shared", "enclave"
	ReadScopes  []string `json:"read_scopes,omitempty" yaml:"read_scopes,omitempty"`
	WriteScopes []string `json:"write_scopes,omitempty" yaml:"write_scopes,omitempty"`
}

type MacaroonCaveats struct {
	PreBoundCaveats []string `json:"pre_bound_caveats,omitempty" yaml:"pre_bound_caveats,omitempty"`
}

// AgentManifest represents a complete AGENT.yaml specification per ADR-21.
type AgentManifest struct {
	Metadata         AgentMetadata     `json:"metadata" yaml:"metadata"`
	Identity         AgentIdentity     `json:"identity" yaml:"identity"`
	Capabilities     AgentCapabilities `json:"capabilities" yaml:"capabilities"`
	ModelPreferences ModelPreferences  `json:"model_preferences" yaml:"model_preferences"`
	MemoryIsolation  MemoryIsolation   `json:"memory_isolation" yaml:"memory_isolation"`
	MacaroonCaveats  MacaroonCaveats   `json:"macaroon_caveats,omitempty" yaml:"macaroon_caveats,omitempty"`
	EphemeralTTLSec  int               `json:"ephemeral_ttl_sec,omitempty" yaml:"ephemeral_ttl_sec,omitempty"`
}

// ParseAgentManifest parses JSON/YAML content and validates required fields per ADR-21.
func ParseAgentManifest(data []byte) (*AgentManifest, error) {
	if len(data) == 0 {
		return nil, ErrInvalidManifest
	}

	var manifest AgentManifest
	// Standard JSON unmarshal supports JSON formatted manifests.
	// For YAML, if JSON unmarshal fails, we convert simple key-value pairs or rely on JSON unmarshaling.
	err := json.Unmarshal(data, &manifest)
	if err != nil {
		// Attempt lightweight key-value mapping if JSON unmarshal fails
		if parseErr := parseYamlLikeBytes(data, &manifest); parseErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
	}

	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// Validate checks manifest compliance against ADR-21 specifications.
func (m *AgentManifest) Validate() error {
	if m.Metadata.Name == "" {
		return fmt.Errorf("%w: metadata.name is required", ErrMissingRequiredField)
	}
	if m.Metadata.Version == "" {
		return fmt.Errorf("%w: metadata.version is required", ErrMissingRequiredField)
	}
	if m.Identity.Role == "" {
		return fmt.Errorf("%w: identity.role is required", ErrMissingRequiredField)
	}

	b := strings.ToLower(m.MemoryIsolation.Boundary)
	if b != "session" && b != "agent" && b != "shared" && b != "enclave" {
		return fmt.Errorf("%w: invalid boundary '%s', expected 'session'|'agent'|'shared'|'enclave'", ErrInvalidBoundary, m.MemoryIsolation.Boundary)
	}

	if m.Capabilities.MaxDelegationDepth < 0 {
		m.Capabilities.MaxDelegationDepth = 3 // Default depth limit
	}

	return nil
}

// ExportAgentPackage packages AGENT.yaml into a signed .agent tar.gz archive.
func ExportAgentPackage(manifest *AgentManifest, privKey ed25519.PrivateKey, pubKey ed25519.PublicKey) ([]byte, error) {
	if manifest == nil {
		return nil, ErrInvalidManifest
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	if len(privKey) != ed25519.PrivateKeySize {
		return nil, errors.New("agent/manifest: invalid ed25519 private key size")
	}

	yamlBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("agent/manifest: failed to serialize manifest: %w", err)
	}

	signature := ed25519.Sign(privKey, yamlBytes)
	sigHex := hex.EncodeToString(signature)
	pubHex := hex.EncodeToString(pubKey)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Write AGENT.yaml
	if err := writeTarFile(tw, "AGENT.yaml", yamlBytes); err != nil {
		return nil, err
	}

	// Write signature.sig
	if err := writeTarFile(tw, "signature.sig", []byte(sigHex)); err != nil {
		return nil, err
	}

	// Write public_key.pub
	if err := writeTarFile(tw, "public_key.pub", []byte(pubHex)); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ImportAgentPackage unpacks a .agent tar.gz package and verifies the Ed25519 signature.
func ImportAgentPackage(pkgBytes []byte, trustedPubKey ed25519.PublicKey) (*AgentManifest, error) {
	if len(pkgBytes) == 0 {
		return nil, ErrCoractedPackage
	}

	gr, err := gzip.NewReader(bytes.NewReader(pkgBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: gzip reader error: %v", ErrCoractedPackage, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	var manifestBytes []byte
	var sigBytes []byte
	var pubKeyBytes []byte

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: tar read error: %v", ErrCoractedPackage, err)
		}

		content, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("%w: file read error: %v", ErrCoractedPackage, err)
		}

		switch header.Name {
		case "AGENT.yaml", "AGENT.json":
			manifestBytes = content
		case "signature.sig":
			sigBytes = content
		case "public_key.pub":
			pubKeyBytes = content
		}
	}

	if len(manifestBytes) == 0 || len(sigBytes) == 0 {
		return nil, fmt.Errorf("%w: missing AGENT.yaml or signature.sig", ErrCoractedPackage)
	}

	// Decode Signature
	sig, err := hex.DecodeString(strings.TrimSpace(string(sigBytes)))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return nil, fmt.Errorf("%w: malformed signature format", ErrSignatureMismatch)
	}

	// Determine Public Key
	var pubKey ed25519.PublicKey
	if len(trustedPubKey) == ed25519.PublicKeySize {
		pubKey = trustedPubKey
	} else if len(pubKeyBytes) > 0 {
		keyBytes, err := hex.DecodeString(strings.TrimSpace(string(pubKeyBytes)))
		if err == nil && len(keyBytes) == ed25519.PublicKeySize {
			pubKey = keyBytes
		}
	}

	if len(pubKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: valid ed25519 public key not found for verification", ErrSignatureMismatch)
	}

	// Verify Ed25519 Signature over manifest content
	if !ed25519.Verify(pubKey, manifestBytes, sig) {
		return nil, ErrSignatureMismatch
	}

	return ParseAgentManifest(manifestBytes)
}

func writeTarFile(tw *tar.Writer, name string, content []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Mode:    0644,
		Size:    int64(len(content)),
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(content)
	return err
}

func parseYamlLikeBytes(data []byte, manifest *AgentManifest) error {
	// Basic line-by-line fallback parser for non-nested YAML manifests
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			switch key {
			case "name":
				manifest.Metadata.Name = val
			case "version":
				manifest.Metadata.Version = val
			case "role":
				manifest.Identity.Role = val
			case "boundary":
				manifest.MemoryIsolation.Boundary = val
			}
		}
	}
	if manifest.Metadata.Name == "" || manifest.Identity.Role == "" {
		return errors.New("fallback yaml parsing incomplete")
	}
	if manifest.MemoryIsolation.Boundary == "" {
		manifest.MemoryIsolation.Boundary = "session"
	}
	return nil
}

func SecureStringCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
