package skill

import (
	"archive/tar"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ============================================================================
// ADR-23: Portable .skill Package Export, Import, & Ed25519 Attestation
// ============================================================================

var (
	ErrSignatureMissing    = errors.New("package: .skill package missing signature.sig")
	ErrManifestMissing     = errors.New("package: .skill package missing manifest.json")
	ErrSignatureInvalid    = errors.New("package: .skill package signature verification failed")
	ErrChecksumMismatch    = errors.New("package: .skill package file checksum mismatch")
	ErrInvalidSkillArchive = errors.New("package: invalid .skill archive format")
)

// PackageManifest records file checksums and export metadata.
type PackageManifest struct {
	SkillName  string            `json:"skill_name"`
	Version    string            `json:"version"`
	ExportedAt time.Time         `json:"exported_at"`
	PublicKey  string            `json:"public_key"` // Hex-encoded Ed25519 public key
	Checksums  map[string]string `json:"checksums"`  // rel_path -> SHA256 hex
}

// ExportSkillPackage packages a skill directory into a signed .skill (.tar.gz) archive.
func ExportSkillPackage(skillDir string, privKey ed25519.PrivateKey, pubKey ed25519.PublicKey, outputPath string) error {
	skillPath := filepath.Join(skillDir, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return fmt.Errorf("package: failed to read SKILL.md: %w", err)
	}

	frontmatter, _, err := ParseSKILLMD(data)
	if err != nil {
		return fmt.Errorf("package: invalid SKILL.md: %w", err)
	}

	checksums := make(map[string]string)
	err = filepath.Walk(skillDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(skillDir, path)
		if relErr != nil || rel == "manifest.json" || rel == "signature.sig" {
			return nil
		}

		content, rErr := os.ReadFile(path)
		if rErr != nil {
			return rErr
		}

		hash := sha256.Sum256(content)
		checksums[rel] = hex.EncodeToString(hash[:])
		return nil
	})
	if err != nil {
		return fmt.Errorf("package: failed to compute checksums: %w", err)
	}

	manifest := PackageManifest{
		SkillName:  frontmatter.Name,
		Version:    frontmatter.Version,
		ExportedAt: time.Now().UTC(),
		PublicKey:  hex.EncodeToString(pubKey),
		Checksums:  checksums,
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("package: failed to marshal manifest: %w", err)
	}

	signature := ed25519.Sign(privKey, manifestBytes)

	outDir := filepath.Dir(outputPath)
	if outDir != "" {
		_ = os.MkdirAll(outDir, 0755)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("package: failed to create output archive file: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// 1. Write SKILL.md & Asset Files
	for relPath := range checksums {
		fullPath := filepath.Join(skillDir, relPath)
		content, rErr := os.ReadFile(fullPath)
		if rErr != nil {
			return rErr
		}
		info, _ := os.Stat(fullPath)

		header := &tar.Header{
			Name:    relPath,
			Mode:    int64(info.Mode()),
			Size:    int64(len(content)),
			ModTime: info.ModTime(),
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tw.Write(content); err != nil {
			return err
		}
	}

	// 2. Write manifest.json
	mHeader := &tar.Header{
		Name:    "manifest.json",
		Mode:    0644,
		Size:    int64(len(manifestBytes)),
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(mHeader); err != nil {
		return err
	}
	if _, err := tw.Write(manifestBytes); err != nil {
		return err
	}

	// 3. Write signature.sig
	sHeader := &tar.Header{
		Name:    "signature.sig",
		Mode:    0644,
		Size:    int64(len(signature)),
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(sHeader); err != nil {
		return err
	}
	if _, err := tw.Write(signature); err != nil {
		return err
	}

	return nil
}

// ImportSkillPackage verifies Ed25519 signature and checksums, then unpacks a .skill package.
func ImportSkillPackage(archivePath string, trustedPubKey ed25519.PublicKey, targetDir string) (*SkillDiscovery, error) {
	inFile, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("package: failed to open package archive: %w", err)
	}
	defer inFile.Close()

	gr, err := gzip.NewReader(inFile)
	if err != nil {
		return nil, fmt.Errorf("package: invalid gzip format: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	files := make(map[string][]byte)
	var manifestBytes []byte
	var sigBytes []byte

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("package: error reading tar archive: %w", err)
		}

		if header.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}

			cleanedName := filepath.Clean(header.Name)
			if strings.HasPrefix(cleanedName, "..") {
				return nil, errors.New("package: archive path traversal attempt detected")
			}

			switch cleanedName {
			case "manifest.json":
				manifestBytes = content
			case "signature.sig":
				sigBytes = content
			default:
				files[cleanedName] = content
			}
		}
	}

	if len(manifestBytes) == 0 {
		return nil, ErrManifestMissing
	}
	if len(sigBytes) == 0 {
		return nil, ErrSignatureMissing
	}

	// Cryptographic Signature Verification
	if len(trustedPubKey) == ed25519.PublicKeySize {
		if !ed25519.Verify(trustedPubKey, manifestBytes, sigBytes) {
			return nil, ErrSignatureInvalid
		}
	} else if len(sigBytes) != ed25519.SignatureSize {
		return nil, ErrSignatureInvalid
	}

	var manifest PackageManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("package: invalid manifest JSON: %w", err)
	}

	// Verify Checksums
	for relPath, expectedHash := range manifest.Checksums {
		content, exists := files[relPath]
		if !exists {
			return nil, fmt.Errorf("%w: missing file %s", ErrChecksumMismatch, relPath)
		}
		hash := sha256.Sum256(content)
		actualHash := hex.EncodeToString(hash[:])
		if actualHash != expectedHash {
			return nil, fmt.Errorf("%w: checksum mismatch on %s (got %s, expected %s)", ErrChecksumMismatch, relPath, actualHash, expectedHash)
		}
	}

	// Unpack to target directory
	destSkillDir := filepath.Join(targetDir, manifest.SkillName)
	if err := os.MkdirAll(destSkillDir, 0755); err != nil {
		return nil, fmt.Errorf("package: failed to create destination dir: %w", err)
	}

	for relPath, content := range files {
		destPath := filepath.Join(destSkillDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return nil, err
		}
	}

	// Read imported SKILL.md for discovery metadata
	importedSkillMD := filepath.Join(destSkillDir, "SKILL.md")
	mdData, err := os.ReadFile(importedSkillMD)
	if err != nil {
		return nil, fmt.Errorf("package: failed to read imported SKILL.md: %w", err)
	}

	frontmatter, _, err := ParseSKILLMD(mdData)
	if err != nil {
		return nil, err
	}

	return &SkillDiscovery{
		Name:         frontmatter.Name,
		Description:  frontmatter.Description,
		Version:      frontmatter.Version,
		Author:       frontmatter.Author,
		AllowedTools: frontmatter.AllowedTools,
		Tags:         frontmatter.Tags,
		Directory:    destSkillDir,
	}, nil
}
