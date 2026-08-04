package project

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAll10Modalities(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aiof_project_modality_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewEngine(nil, nil, nil)
	ctx := context.Background()

	for idx, modality := range AllModalities {
		projName := filepath.Join(tmpDir, string(modality))
		proj, err := engine.CreateProject(ctx, string(modality), tmpDir, modality)
		if err != nil {
			t.Fatalf("[%d] Failed to create project for modality %s: %v", idx, modality, err)
		}

		if proj.Metadata.Modality != modality {
			t.Errorf("[%d] Expected modality %s, got %s", idx, modality, proj.Metadata.Modality)
		}

		if proj.Budget == nil {
			t.Errorf("[%d] Expected non-nil budget cap for modality %s", idx, modality)
		}

		// Verify re-loading project
		loaded, err := engine.LoadProject(ctx, projName)
		if err != nil {
			t.Fatalf("[%d] Failed to re-load project at %s: %v", idx, projName, err)
		}

		if loaded.Metadata.ID != proj.Metadata.ID {
			t.Errorf("[%d] Expected project ID %s, got %s", idx, proj.Metadata.ID, loaded.Metadata.ID)
		}
	}
}

func TestSwitchModality(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aiof_project_switch_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewEngine(nil, nil, nil)
	ctx := context.Background()

	proj, err := engine.CreateProject(ctx, "switch_test", tmpDir, ModalitySoftwareEngineering)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Switch to Media Creative
	switched, err := engine.SwitchModality(ctx, proj.Metadata.ID, ModalityMediaCreative)
	if err != nil {
		t.Fatalf("Failed to switch modality: %v", err)
	}

	if switched.Metadata.Modality != ModalityMediaCreative {
		t.Errorf("Expected modality %s, got %s", ModalityMediaCreative, switched.Metadata.Modality)
	}
}

func TestConcurrentProjectAccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aiof_project_concurrent_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewEngine(nil, nil, nil)
	ctx := context.Background()

	proj, err := engine.CreateProject(ctx, "concurrent_test", tmpDir, ModalityWebFullstack)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if id%2 == 0 {
				_, _ = engine.GetActiveProject()
			} else {
				_, _ = engine.LoadProject(ctx, proj.Metadata.RootPath)
			}
		}(i)
	}
	wg.Wait()
}
