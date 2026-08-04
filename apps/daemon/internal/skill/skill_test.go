package skill

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSKILLMD(t *testing.T) {
	content := `---
name: code-refactor
description: Automated multi-file Go refactoring skill
version: 1.0.0
author: Jeff Kenniston
allowed_tools: [git, gofmt, govet]
tags: [go, refactor, ast]
---
# Code Refactoring Instructions
Execute automated AST refactoring following clean code guidelines.
`

	fm, body, err := ParseSKILLMD([]byte(content))
	if err != nil {
		t.Fatalf("ParseSKILLMD failed: %v", err)
	}

	if fm.Name != "code-refactor" {
		t.Errorf("expected name 'code-refactor', got '%s'", fm.Name)
	}
	if fm.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", fm.Version)
	}
	if len(fm.AllowedTools) != 3 {
		t.Errorf("expected 3 allowed tools, got %d", len(fm.AllowedTools))
	}
	if !containsStr(body, "Code Refactoring Instructions") {
		t.Errorf("expected body to contain heading, got '%s'", body)
	}
}

func Test3TierProgressiveDisclosure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "skill-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	skillDir := filepath.Join(tempDir, "test-skill")
	scriptsDir := filepath.Join(skillDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("failed to create skill dirs: %v", err)
	}

	skillMD := `---
name: test-skill
description: Test Progressive Disclosure
version: 2.0.0
---
# Instruction Body
This is Level 2 instruction content.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	scriptContent := "#!/bin/bash\necho 'running asset script'"
	if err := os.WriteFile(filepath.Join(scriptsDir, "deploy.sh"), []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write script asset: %v", err)
	}

	mgr := NewSkillManager(tempDir)

	// Level 1: Discovery
	discoveries, err := mgr.ScanDirectory(tempDir)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}
	if len(discoveries) != 1 {
		t.Fatalf("expected 1 discovery, got %d", len(discoveries))
	}
	disc, ok := mgr.GetDiscovery("test-skill")
	if !ok || disc.Name != "test-skill" {
		t.Fatalf("Level 1 GetDiscovery failed")
	}

	// Level 2: Instruction Body
	inst, err := mgr.LoadInstruction("test-skill")
	if err != nil {
		t.Fatalf("Level 2 LoadInstruction failed: %v", err)
	}
	if !containsStr(inst.Instruction, "Level 2 instruction content") {
		t.Errorf("unexpected Level 2 content: %s", inst.Instruction)
	}

	// Level 3: Execution Asset
	asset, err := mgr.LoadAsset("test-skill", "scripts/deploy.sh")
	if err != nil {
		t.Fatalf("Level 3 LoadAsset failed: %v", err)
	}
	if asset.Category != AssetCategoryScripts {
		t.Errorf("expected AssetCategoryScripts, got %s", asset.Category)
	}
	if !containsStr(string(asset.Content), "running asset script") {
		t.Errorf("unexpected asset content: %s", string(asset.Content))
	}
}

func TestSkillPackageExportImport(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "package-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	srcSkillDir := filepath.Join(tempDir, "src-skill")
	if err := os.MkdirAll(srcSkillDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	skillMD := `---
name: export-skill
description: Skill for export test
version: 1.5.0
---
# Instructions
Export test instruction body.
`
	if err := os.WriteFile(filepath.Join(srcSkillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}

	archivePath := filepath.Join(tempDir, "export-skill.skill")
	if err := ExportSkillPackage(srcSkillDir, priv, pub, archivePath); err != nil {
		t.Fatalf("ExportSkillPackage failed: %v", err)
	}

	importDir := filepath.Join(tempDir, "imported")
	disc, err := ImportSkillPackage(archivePath, pub, importDir)
	if err != nil {
		t.Fatalf("ImportSkillPackage failed: %v", err)
	}

	if disc.Name != "export-skill" {
		t.Errorf("expected imported name 'export-skill', got '%s'", disc.Name)
	}

	// Test invalid signature rejection
	badPub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, err = ImportSkillPackage(archivePath, badPub, importDir)
	if err == nil {
		t.Errorf("expected error when verifying with wrong public key, got nil")
	}
}

func containsStr(s, sub string) bool {
	return filepath.HasPrefix(s, sub) || len(s) >= len(sub) && (s == sub || searchSub(s, sub))
}

func searchSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
