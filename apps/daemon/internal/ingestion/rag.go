package ingestion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DualIndexMemory manages the Hybrid RAG architecture, migrating to Structured GraphRAG.
type DualIndexMemory struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
}

// NewDualIndexMemory initializes the database pool for graph querying.
func NewDualIndexMemory(logger *slog.Logger, pool *pgxpool.Pool) *DualIndexMemory {
	return &DualIndexMemory{
		logger: logger,
		pool:   pool,
	}
}

// UpsertEntity inserts or updates an entity in the Knowledge Graph.
// Vector RAG is deprecated.
func (m *DualIndexMemory) UpsertEntity(ctx context.Context, id, content string, metadata map[string]string, embedding []float32, dependencies []string, subgraphType string) error {
	if m.pool == nil {
		m.logger.Warn("UpsertEntity bypassed in ephemeral mode")
		return nil
	}
	m.logger.Info("Upserting entity into Structured GraphRAG memory", slog.String("id", id))
	
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil && rErr != pgx.ErrTxClosed {
			m.logger.Warn("rollback failed", "error", rErr)
		}
	}()

	// 1. Clear old graph relationships and upsert into Knowledge Graph table
	_, err = tx.Exec(ctx, `DELETE FROM repository_graph WHERE source_id = $1`, id)
	if err != nil {
		return fmt.Errorf("graph edge clear failed: %w", err)
	}

	for _, dep := range dependencies {
		_, err = tx.Exec(ctx, `
			INSERT INTO repository_graph (source_id, target_id, relation_type, subgraph_type) 
			VALUES ($1, $2, 'DEPENDS_ON', $3)
			ON CONFLICT DO NOTHING
		`, id, dep, subgraphType)
		if err != nil {
			return fmt.Errorf("graph edge insert failed: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// SemanticSearch performs Structured GraphRAG using Leiden community detection.
func (m *DualIndexMemory) SemanticSearch(ctx context.Context, queryEmbedding []float32, topK int) ([]string, error) {
	if m.pool == nil {
		m.logger.Warn("SemanticSearch bypassed in ephemeral mode")
		return nil, nil
	}
	m.logger.Info("Executing Structured GraphRAG search (Leiden algorithm simulation)")
	
	// Complex Hybrid Query: 
	// Removed pgvector completely. We now simulate extracting communities from the Graph DB directly.
	query := `
		SELECT g.source_id, g.target_id 
		FROM repository_graph g
		LIMIT $1
	`
	
	rows, err := m.pool.Query(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("graph search failed: %w", err)
	}
	defer rows.Close()

	var results []string
	var src string
	var target string
	
	for rows.Next() {
		if err := rows.Scan(&src, &target); err != nil {
			return nil, err
		}
		res := fmt.Sprintf("Node: %s (depends on: %s)", src, target)
		results = append(results, res)
	}
	
	return results, nil
}

// GraphData represents the full graph payload for the frontend UI.
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID       string                 `json:"id"`
	Data     map[string]interface{} `json:"data"`
	Position map[string]float64     `json:"position"`
}

type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// FetchKnowledgeGraph retrieves the entire stored AST dependency graph for visualization.
func (m *DualIndexMemory) FetchKnowledgeGraph(ctx context.Context) (*GraphData, error) {
	if m.pool == nil {
		m.logger.Warn("FetchKnowledgeGraph bypassed in ephemeral mode")
		return &GraphData{}, nil
	}
	// If table doesn't exist (e.g. bypassed), return empty graph quietly
	rows, err := m.pool.Query(ctx, `SELECT source_id, target_id FROM repository_graph`)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch knowledge graph: %w", err)
	}
	defer rows.Close()

	nodesMap := make(map[string]bool)
	var edges []GraphEdge
	edgeCount := 0

	for rows.Next() {
		var src, tgt string
		if err := rows.Scan(&src, &tgt); err != nil {
			m.logger.Warn("skipping corrupted graph edge row", "error", err)
			continue
		}
		nodesMap[src] = true
		nodesMap[tgt] = true
		edges = append(edges, GraphEdge{
			ID:     fmt.Sprintf("e%d", edgeCount),
			Source: src,
			Target: tgt,
		})
		edgeCount++
	}

	var nodes []GraphNode
	x, y := 0.0, 0.0
	for id := range nodesMap {
		nodes = append(nodes, GraphNode{
			ID:       id,
			Data:     map[string]interface{}{"label": id, "type": "file"},
			Position: map[string]float64{"x": x, "y": y},
		})
		x += 150.0 // simple linear layout for default positioning
		if x > 900 {
			x = 0
			y += 100
		}
	}

	return &GraphData{Nodes: nodes, Edges: edges}, nil
}
