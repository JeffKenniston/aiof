package ingestion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// DualIndexMemory manages the Hybrid RAG architecture, utilizing both a Vector DB
// for semantic search and a Knowledge Graph for structural codebase relationships.
type DualIndexMemory struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
}

// NewDualIndexMemory initializes the database pool for dual-index vector/graph querying.
func NewDualIndexMemory(logger *slog.Logger, pool *pgxpool.Pool) *DualIndexMemory {
	return &DualIndexMemory{
		logger: logger,
		pool:   pool,
	}
}

// UpsertEntity inserts or updates an entity in both the Knowledge Graph and Vector index.
// It explicitly generates embeddings and stores relationships.
func (m *DualIndexMemory) UpsertEntity(ctx context.Context, id, content string, metadata map[string]string, embedding []float32, dependencies []string, subgraphType string) error {
	m.logger.Info("Upserting entity into dual-index memory", slog.String("id", id))
	
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil && rErr != pgx.ErrTxClosed {
			m.logger.Warn("rollback failed", "error", rErr)
		}
	}()

	// 1. Upsert into Vector DB table (pgvector)
	vec := pgvector.NewVector(embedding)
	_, err = tx.Exec(ctx, `
		INSERT INTO repository_embeddings (id, content, metadata, embedding) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET 
			content = EXCLUDED.content, 
			metadata = EXCLUDED.metadata, 
			embedding = EXCLUDED.embedding
	`, id, content, metadata, vec)
	if err != nil {
		return fmt.Errorf("vector upsert failed: %w", err)
	}

	// 2. Clear old graph relationships and upsert into Knowledge Graph table
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

// SemanticSearch queries the Vector DB using pgvector `<=>` operator (cosine distance)
// and enriches results by traversing the Knowledge Graph.
func (m *DualIndexMemory) SemanticSearch(ctx context.Context, queryEmbedding []float32, topK int) ([]string, error) {
	m.logger.Info("Executing hybrid semantic search")
	
	vec := pgvector.NewVector(queryEmbedding)
	
	// Complex Hybrid Query: 
	// 1. Get top-K semantic matches via Vector DB
	// 2. Join against Graph DB to extract adjacent context nodes
	query := `
		WITH semantic_matches AS (
			SELECT id, content, embedding <=> $1 AS distance
			FROM repository_embeddings
			ORDER BY distance ASC
			LIMIT $2
		)
		SELECT m.id, g.target_id 
		FROM semantic_matches m
		LEFT JOIN repository_graph g ON m.id = g.source_id
	`
	
	rows, err := m.pool.Query(ctx, query, vec, topK)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}
	defer rows.Close()

	var results []string
	var id string
	var targetId *string
	
	for rows.Next() {
		if err := rows.Scan(&id, &targetId); err != nil {
			return nil, err
		}
		res := id
		if targetId != nil {
			res += fmt.Sprintf(" (depends on: %s)", *targetId)
		}
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
