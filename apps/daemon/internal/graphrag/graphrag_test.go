package graphrag

import (
	"context"
	"strings"
	"testing"
)

const sampleGoCode = `
package orchestrator

import (
	"context"
	"fmt"
)

// PipelineConfig configures multi-agent pipeline parameters.
type PipelineConfig struct {
	MaxWorkers int
	DebugMode  bool
}

// AgentRunner executes agent tasks asynchronously.
type AgentRunner interface {
	RunTask(ctx context.Context, task string) error
}

// OrchestratorEngine manages multi-agent choreography.
type OrchestratorEngine struct {
	Config PipelineConfig
}

// ProcessJob executes a pipeline job and logs output.
func (o *OrchestratorEngine) ProcessJob(ctx context.Context, jobID string) error {
	fmt.Println("Processing job:", jobID)
	return o.RunTask(ctx, jobID)
}

// RunTask executes the runner implementation.
func (o *OrchestratorEngine) RunTask(ctx context.Context, task string) error {
	return nil
}
`

func TestGraphRAGExtractionAndTraversal(t *testing.T) {
	ctx := context.Background()
	engine, err := NewGraphRAGEngine(":memory:")
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") || strings.Contains(err.Error(), "requires cgo") {
			t.Skip("Skipping GraphRAG test: CGO is disabled on this platform/build environment")
		}
		t.Fatalf("failed to initialize GraphRAG engine: %v", err)
	}
	defer engine.Close()

	// 1. Index Sample Source
	err = engine.ExtractAndIndexSource(ctx, "orchestrator/engine.go", sampleGoCode)
	if err != nil {
		t.Fatalf("failed to extract and index source: %v", err)
	}

	// 2. Verify Entity Extraction
	entities, relations, err := engine.LoadGraphState(ctx)
	if err != nil {
		t.Fatalf("failed to load graph state: %v", err)
	}

	if len(entities) == 0 {
		t.Fatalf("expected extracted entities, got 0")
	}

	if len(relations) == 0 {
		t.Fatalf("expected extracted relations, got 0")
	}

	// Check Structs
	foundStruct := false
	for _, ent := range entities {
		if ent.Name == "OrchestratorEngine" && ent.Type == EntityStruct {
			foundStruct = true
			break
		}
	}
	if !foundStruct {
		t.Errorf("expected OrchestratorEngine struct entity")
	}

	// Check Interfaces
	foundIface := false
	for _, ent := range entities {
		if ent.Name == "AgentRunner" && ent.Type == EntityInterface {
			foundIface = true
			break
		}
	}
	if !foundIface {
		t.Errorf("expected AgentRunner interface entity")
	}

	// 3. Test Leiden Community Detection
	err = engine.RunLeidenClustering(ctx, 1.0)
	if err != nil {
		t.Fatalf("failed to run Leiden clustering: %v", err)
	}

	// Verify communities were populated
	var commCount int
	err = engine.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM graph_communities").Scan(&commCount)
	if err != nil {
		t.Fatalf("failed to query community count: %v", err)
	}
	if commCount == 0 {
		t.Errorf("expected graph communities > 0, got %d", commCount)
	}

	// 4. Test Multi-Hop Traversal
	res, err := engine.TraverseAndReason(ctx, "OrchestratorEngine", 2)
	if err != nil {
		t.Fatalf("failed to traverse graph: %v", err)
	}

	if len(res.SeedEntities) == 0 {
		t.Errorf("expected seed entities for 'OrchestratorEngine'")
	}

	if !strings.Contains(res.SynthesizedText, "OrchestratorEngine") {
		t.Errorf("expected synthesized context to contain 'OrchestratorEngine'")
	}
}
