package canvas

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMultimodalCanvasEngine(t *testing.T) {
	engine := NewMultimodalCanvasEngine()

	subCh := engine.Subscribe("client-ui-1")
	defer engine.Unsubscribe("client-ui-1")

	data := json.RawMessage(`{"nodes":[{"id":"n1","label":"Service A"}]}`)
	art, err := engine.UpsertArtifact("art-001", "proj-1", "Architecture Diagram", ArtifactUMLDiagram, data, "agent-designer", nil)
	if err != nil {
		t.Fatalf("failed to upsert artifact: %v", err)
	}

	if art.Version != 1 {
		t.Errorf("expected version 1, got %d", art.Version)
	}

	// Verify subscriber notification
	select {
	case evt := <-subCh:
		if evt.Action != "CREATED" || evt.ArtifactID != "art-001" {
			t.Errorf("unexpected canvas event: %v", evt)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("timed out waiting for canvas reactive event")
	}

	// Update artifact -> version bump to 2
	data2 := json.RawMessage(`{"nodes":[{"id":"n1","label":"Service A"},{"id":"n2","label":"Service B"}]}`)
	art2, err := engine.UpsertArtifact("art-001", "proj-1", "Architecture Diagram", ArtifactUMLDiagram, data2, "agent-designer", nil)
	if err != nil {
		t.Fatalf("failed to update artifact: %v", err)
	}

	if art2.Version != 2 {
		t.Errorf("expected version 2, got %d", art2.Version)
	}
}
