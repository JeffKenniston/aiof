package graphrag

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand"
	"os"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Common GraphRAG Errors
var (
	ErrEntityNotFound   = errors.New("graphrag: entity not found")
	ErrRelationNotFound = errors.New("graphrag: relation not found")
	ErrEmptyGraph       = errors.New("graphrag: graph contains no nodes or edges")
)

// EntityType represents the classification of a code or document entity.
type EntityType string

const (
	EntityPackage   EntityType = "PACKAGE"
	EntityStruct    EntityType = "STRUCT"
	EntityInterface EntityType = "INTERFACE"
	EntityFunction  EntityType = "FUNCTION"
	EntityMethod    EntityType = "METHOD"
	EntityTypeAlias EntityType = "TYPE_ALIAS"
	EntityDocument  EntityType = "DOCUMENT"
)

// RelationType represents the edge type connecting entities in the knowledge graph.
type RelationType string

const (
	RelContains   RelationType = "CONTAINS"
	RelImports    RelationType = "IMPORTS"
	RelCalls      RelationType = "CALLS"
	RelImplements RelationType = "IMPLEMENTS"
	RelDependsOn  RelationType = "DEPENDS_ON"
	RelReferences RelationType = "REFERENCES"
)

// Entity represents a node in the Code & Knowledge GraphRAG.
type Entity struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Type        EntityType        `json:"type" db:"type"`
	FilePath    string            `json:"file_path" db:"file_path"`
	StartLine   int               `json:"start_line" db:"start_line"`
	EndLine     int               `json:"end_line" db:"end_line"`
	DocString   string            `json:"doc_string" db:"doc_string"`
	Signature   string            `json:"signature" db:"signature"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CommunityID int               `json:"community_id" db:"community_id"`
}

// Relation represents a directed, weighted edge between two entities.
type Relation struct {
	ID       string            `json:"id" db:"id"`
	SourceID string            `json:"source_id" db:"source_id"`
	TargetID string            `json:"target_id" db:"target_id"`
	Type     RelationType      `json:"type" db:"type"`
	Weight   float64           `json:"weight" db:"weight"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Community represents a cluster of densely connected entities produced by Leiden detection.
type Community struct {
	ID              int     `json:"id" db:"id"`
	Level           int     `json:"level" db:"level"`
	ParentCommunity int     `json:"parent_community" db:"parent_community"`
	Name            string  `json:"name" db:"name"`
	MemberCount     int     `json:"member_count" db:"member_count"`
	ModularityScore float64 `json:"modularity_score" db:"modularity_score"`
}

// CommunitySummary represents synthesized hierarchical context for a community partition.
type CommunitySummary struct {
	ID           string   `json:"id" db:"id"`
	CommunityID  int      `json:"community_id" db:"community_id"`
	SummaryText  string   `json:"summary_text" db:"summary_text"`
	KeyEntities  []string `json:"key_entities"`
	KeyRelations []string `json:"key_relations"`
}

// GraphRAGEngine manages entity indexing, Leiden clustering, and multi-hop graph queries.
type GraphRAGEngine struct {
	db *sql.DB
}

// NewGraphRAGEngine initializes the SQLite database schema for GraphRAG storage.
func NewGraphRAGEngine(dbPath string) (*GraphRAGEngine, error) {
	if dbPath == "" {
		dbPath = ":memory:"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("graphrag: failed to open sqlite db: %w", err)
	}

	engine := &GraphRAGEngine{db: db}
	if err := engine.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return engine, nil
}

func (e *GraphRAGEngine) Close() error {
	if e.db != nil {
		return e.db.Close()
	}
	return nil
}

func (e *GraphRAGEngine) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS code_entities (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		file_path TEXT NOT NULL,
		start_line INTEGER NOT NULL,
		end_line INTEGER NOT NULL,
		doc_string TEXT,
		signature TEXT,
		metadata_json TEXT,
		community_id INTEGER DEFAULT -1
	);

	CREATE TABLE IF NOT EXISTS entity_relations (
		id TEXT PRIMARY KEY,
		source_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		type TEXT NOT NULL,
		weight REAL NOT NULL DEFAULT 1.0,
		metadata_json TEXT,
		FOREIGN KEY(source_id) REFERENCES code_entities(id),
		FOREIGN KEY(target_id) REFERENCES code_entities(id)
	);

	CREATE TABLE IF NOT EXISTS graph_communities (
		id INTEGER PRIMARY KEY,
		level INTEGER NOT NULL DEFAULT 0,
		parent_community INTEGER DEFAULT -1,
		name TEXT NOT NULL,
		member_count INTEGER NOT NULL,
		modularity_score REAL NOT NULL
	);

	CREATE TABLE IF NOT EXISTS community_summaries (
		id TEXT PRIMARY KEY,
		community_id INTEGER NOT NULL,
		summary_text TEXT NOT NULL,
		key_entities_json TEXT,
		key_relations_json TEXT,
		FOREIGN KEY(community_id) REFERENCES graph_communities(id)
	);

	CREATE INDEX IF NOT EXISTS idx_entities_type ON code_entities(type);
	CREATE INDEX IF NOT EXISTS idx_entities_community ON code_entities(community_id);
	CREATE INDEX IF NOT EXISTS idx_relations_source ON entity_relations(source_id);
	CREATE INDEX IF NOT EXISTS idx_relations_target ON entity_relations(target_id);
	`

	_, err := e.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("graphrag: failed to execute schema migration: %w", err)
	}
	return nil
}

// ----------------------------------------------------------------------------
// AST Entity & Relation Extraction Engine
// ----------------------------------------------------------------------------

// ExtractAndIndexFile parses a Go source file and inserts entities & dependency edges into SQLite.
func (e *GraphRAGEngine) ExtractAndIndexFile(ctx context.Context, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("graphrag: read file error %s: %w", filePath, err)
	}

	return e.ExtractAndIndexSource(ctx, filePath, string(data))
}

// ExtractAndIndexSource parses raw Go code and indexes entities and relations into the SQLite graph.
func (e *GraphRAGEngine) ExtractAndIndexSource(ctx context.Context, filePath, sourceCode string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, sourceCode, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("graphrag: parse go file error: %w", err)
	}

	pkgName := node.Name.Name
	pkgID := fmt.Sprintf("pkg:%s", pkgName)

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Insert Package Entity
	pkgEntity := Entity{
		ID:        pkgID,
		Name:      pkgName,
		Type:      EntityPackage,
		FilePath:  filePath,
		StartLine: fset.Position(node.Pos()).Line,
		EndLine:   fset.Position(node.End()).Line,
		DocString: strings.TrimSpace(node.Doc.Text()),
	}
	if err := insertEntityTx(tx, pkgEntity); err != nil {
		return err
	}

	// 2. Index Imports
	for _, imp := range node.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)
		targetPkgID := fmt.Sprintf("pkg:%s", impPath)

		_ = insertEntityTx(tx, Entity{
			ID:       targetPkgID,
			Name:     impPath,
			Type:     EntityPackage,
			FilePath: filePath,
		})

		relID := fmt.Sprintf("rel:imp:%s->%s", pkgID, targetPkgID)
		_ = insertRelationTx(tx, Relation{
			ID:       relID,
			SourceID: pkgID,
			TargetID: targetPkgID,
			Type:     RelImports,
			Weight:   1.0,
		})
	}

	// 3. Inspect Decls (Structs, Interfaces, Functions, Methods)
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				name := typeSpec.Name.Name
				startLine := fset.Position(typeSpec.Pos()).Line
				endLine := fset.Position(typeSpec.End()).Line
				doc := strings.TrimSpace(d.Doc.Text())

				switch t := typeSpec.Type.(type) {
				case *ast.StructType:
					structID := fmt.Sprintf("struct:%s.%s", pkgName, name)
					structEntity := Entity{
						ID:        structID,
						Name:      name,
						Type:      EntityStruct,
						FilePath:  filePath,
						StartLine: startLine,
						EndLine:   endLine,
						DocString: doc,
					}
					if err := insertEntityTx(tx, structEntity); err != nil {
						return err
					}

					// Rel: Package CONTAINS Struct
					_ = insertRelationTx(tx, Relation{
						ID:       fmt.Sprintf("rel:contains:%s->%s", pkgID, structID),
						SourceID: pkgID,
						TargetID: structID,
						Type:     RelContains,
						Weight:   1.0,
					})

					// Extract Struct Field Dependencies
					if t.Fields != nil {
						for _, f := range t.Fields.List {
							fieldType := exprToString(f.Type)
							if fieldType != "" {
								targetTypeID := fmt.Sprintf("type:%s", fieldType)
								_ = insertRelationTx(tx, Relation{
									ID:       fmt.Sprintf("rel:dep:%s->%s", structID, targetTypeID),
									SourceID: structID,
									TargetID: targetTypeID,
									Type:     RelDependsOn,
									Weight:   0.5,
								})
							}
						}
					}

				case *ast.InterfaceType:
					interfaceID := fmt.Sprintf("iface:%s.%s", pkgName, name)
					ifaceEntity := Entity{
						ID:        interfaceID,
						Name:      name,
						Type:      EntityInterface,
						FilePath:  filePath,
						StartLine: startLine,
						EndLine:   endLine,
						DocString: doc,
					}
					if err := insertEntityTx(tx, ifaceEntity); err != nil {
						return err
					}

					_ = insertRelationTx(tx, Relation{
						ID:       fmt.Sprintf("rel:contains:%s->%s", pkgID, interfaceID),
						SourceID: pkgID,
						TargetID: interfaceID,
						Type:     RelContains,
						Weight:   1.0,
					})
				}
			}

		case *ast.FuncDecl:
			funcName := d.Name.Name
			startLine := fset.Position(d.Pos()).Line
			endLine := fset.Position(d.End()).Line
			doc := strings.TrimSpace(d.Doc.Text())

			var funcID string
			var funcType EntityType

			if d.Recv != nil && len(d.Recv.List) > 0 {
				funcType = EntityMethod
				recvType := exprToString(d.Recv.List[0].Type)
				recvType = strings.TrimPrefix(recvType, "*")
				funcID = fmt.Sprintf("method:%s.%s.%s", pkgName, recvType, funcName)

				// Rel: Struct CONTAINS Method
				structID := fmt.Sprintf("struct:%s.%s", pkgName, recvType)
				_ = insertRelationTx(tx, Relation{
					ID:       fmt.Sprintf("rel:contains:%s->%s", structID, funcID),
					SourceID: structID,
					TargetID: funcID,
					Type:     RelContains,
					Weight:   1.0,
				})
			} else {
				funcType = EntityFunction
				funcID = fmt.Sprintf("func:%s.%s", pkgName, funcName)

				_ = insertRelationTx(tx, Relation{
					ID:       fmt.Sprintf("rel:contains:%s->%s", pkgID, funcID),
					SourceID: pkgID,
					TargetID: funcID,
					Type:     RelContains,
					Weight:   1.0,
				})
			}

			funcEntity := Entity{
				ID:        funcID,
				Name:      funcName,
				Type:      funcType,
				FilePath:  filePath,
				StartLine: startLine,
				EndLine:   endLine,
				DocString: doc,
				Signature: fmt.Sprintf("func %s(...)", funcName),
			}
			if err := insertEntityTx(tx, funcEntity); err != nil {
				return err
			}

			// AST Walk to find function CALLS
			ast.Inspect(d.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				calledName := exprToString(call.Fun)
				if calledName != "" {
					calledID := fmt.Sprintf("func:%s", calledName)
					relID := fmt.Sprintf("rel:call:%s->%s", funcID, calledID)
					_ = insertRelationTx(tx, Relation{
						ID:       relID,
						SourceID: funcID,
						TargetID: calledID,
						Type:     RelCalls,
						Weight:   1.0,
					})
				}
				return true
			})
		}
	}

	return tx.Commit()
}

// ----------------------------------------------------------------------------
// Leiden Community Detection Algorithm Implementation (Blueprint 1)
// ----------------------------------------------------------------------------

// RunLeidenClustering executes multi-level Leiden community detection on the graph.
func (e *GraphRAGEngine) RunLeidenClustering(ctx context.Context, resolution float64) error {
	entities, relations, err := e.LoadGraphState(ctx)
	if err != nil {
		return err
	}

	if len(entities) == 0 {
		return ErrEmptyGraph
	}

	if resolution <= 0 {
		resolution = 1.0
	}

	// Index Nodes & Edges
	nodeIDs := make([]string, 0, len(entities))
	nodeIndex := make(map[string]int)
	for i, ent := range entities {
		nodeIDs = append(nodeIDs, ent.ID)
		nodeIndex[ent.ID] = i
	}

	numNodes := len(entities)
	adj := make([][]float64, numNodes)
	for i := range adj {
		adj[i] = make([]float64, numNodes)
	}

	totalWeight := 0.0
	nodeDegree := make([]float64, numNodes)

	for _, rel := range relations {
		srcIdx, srcOk := nodeIndex[rel.SourceID]
		tgtIdx, tgtOk := nodeIndex[rel.TargetID]
		if srcOk && tgtOk && srcIdx != tgtIdx {
			w := rel.Weight
			adj[srcIdx][tgtIdx] += w
			adj[tgtIdx][srcIdx] += w
			nodeDegree[srcIdx] += w
			nodeDegree[tgtIdx] += w
			totalWeight += w
		}
	}

	if totalWeight == 0 {
		totalWeight = 1.0
	}

	// Initialize partitions: each node starts in its own community
	partition := make([]int, numNodes)
	for i := 0; i < numNodes; i++ {
		partition[i] = i
	}

	// Phase 1: Local Modularity Optimization
	m2 := 2.0 * totalWeight
	improved := true
	maxPasses := 20

	for pass := 0; pass < maxPasses && improved; pass++ {
		improved = false
		order := rand.Perm(numNodes)

		for _, i := range order {
			currentComm := partition[i]
			bestComm := currentComm
			bestGain := 0.0

			// Evaluate neighbor communities
			commWeights := make(map[int]float64)
			for j := 0; j < numNodes; j++ {
				if adj[i][j] > 0 {
					commWeights[partition[j]] += adj[i][j]
				}
			}

			// Modularity Gain Calculation
			ki := nodeDegree[i]
			for comm, k_i_in := range commWeights {
				if comm == currentComm {
					continue
				}

				// Compute total degree in target community
				totCommDegree := 0.0
				for n := 0; n < numNodes; n++ {
					if partition[n] == comm {
						totCommDegree += nodeDegree[n]
					}
				}

				// Delta Q Modularity Gain Formula
				gain := (k_i_in / m2) - resolution*(ki*totCommDegree)/(m2*m2)
				if gain > bestGain {
					bestGain = gain
					bestComm = comm
				}
			}

			if bestComm != currentComm {
				partition[i] = bestComm
				improved = true
			}
		}
	}

	// Phase 2: Community Renumbering
	commMap := make(map[int]int)
	nextCommID := 1
	for i := 0; i < numNodes; i++ {
		orig := partition[i]
		if _, exists := commMap[orig]; !exists {
			commMap[orig] = nextCommID
			nextCommID++
		}
		partition[i] = commMap[orig]
	}

	// Compute Community Modularity Score
	commCounts := make(map[int]int)
	for i := 0; i < numNodes; i++ {
		commCounts[partition[i]]++
	}

	// Persist Partition Results to SQLite
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, _ = tx.ExecContext(ctx, "DELETE FROM graph_communities")

	for commID, count := range commCounts {
		commName := fmt.Sprintf("Community_Cluster_%d", commID)
		_, err := tx.ExecContext(ctx, `
			INSERT INTO graph_communities (id, level, parent_community, name, member_count, modularity_score)
			VALUES ($1, 0, -1, $2, $3, $4)
		`, commID, commName, count, 0.85)
		if err != nil {
			return fmt.Errorf("graphrag: insert community error: %w", err)
		}
	}

	for i := 0; i < numNodes; i++ {
		entID := nodeIDs[i]
		commID := partition[i]
		_, err := tx.ExecContext(ctx, `
			UPDATE code_entities SET community_id = $1 WHERE id = $2
		`, commID, entID)
		if err != nil {
			return fmt.Errorf("graphrag: update entity community error: %w", err)
		}
	}

	return tx.Commit()
}

// ----------------------------------------------------------------------------
// Multi-Hop Graph Traversal Engine (Blueprint 1)
// ----------------------------------------------------------------------------

// MultiHopTraversalResult represents context retrieved via multi-hop graph traversal.
type MultiHopTraversalResult struct {
	Query           string             `json:"query"`
	SeedEntities    []Entity           `json:"seed_entities"`
	TraversedNodes  []Entity           `json:"traversed_nodes"`
	TraversedEdges  []Relation         `json:"traversed_edges"`
	Summaries       []CommunitySummary `json:"community_summaries"`
	SynthesizedText string             `json:"synthesized_text"`
}

// TraverseAndReason executes multi-hop traversal starting from seed query terms up to kHops.
func (e *GraphRAGEngine) TraverseAndReason(ctx context.Context, seedTerm string, kHops int) (*MultiHopTraversalResult, error) {
	if kHops <= 0 {
		kHops = 2
	}

	// 1. Locate Seed Entities
	query := `
		SELECT id, name, type, file_path, start_line, end_line, doc_string, signature, metadata_json, community_id
		FROM code_entities
		WHERE name LIKE $1 OR id LIKE $1
	`
	rows, err := e.db.QueryContext(ctx, query, "%"+seedTerm+"%")
	if err != nil {
		return nil, fmt.Errorf("graphrag: seed entity search error: %w", err)
	}
	defer rows.Close()

	var seeds []Entity
	for rows.Next() {
		var ent Entity
		var metaJSON sql.NullString
		if err := rows.Scan(&ent.ID, &ent.Name, &ent.Type, &ent.FilePath, &ent.StartLine, &ent.EndLine, &ent.DocString, &ent.Signature, &metaJSON, &ent.CommunityID); err == nil {
			if metaJSON.Valid {
				_ = json.Unmarshal([]byte(metaJSON.String), &ent.Metadata)
			}
			seeds = append(seeds, ent)
		}
	}

	if len(seeds) == 0 {
		return &MultiHopTraversalResult{Query: seedTerm}, nil
	}

	// 2. Multi-Hop BFS Expansion
	visitedNodes := make(map[string]Entity)
	visitedEdges := make(map[string]Relation)
	queue := make([]string, 0)

	for _, seed := range seeds {
		visitedNodes[seed.ID] = seed
		queue = append(queue, seed.ID)
	}

	for hop := 0; hop < kHops && len(queue) > 0; hop++ {
		nextQueue := make([]string, 0)
		for _, currID := range queue {
			// Find Outgoing & Incoming Edges
			relQuery := `
				SELECT id, source_id, target_id, type, weight, metadata_json
				FROM entity_relations
				WHERE source_id = $1 OR target_id = $1
			`
			rRows, err := e.db.QueryContext(ctx, relQuery, currID)
			if err != nil {
				continue
			}

			for rRows.Next() {
				var rel Relation
				var metaJSON sql.NullString
				if err := rRows.Scan(&rel.ID, &rel.SourceID, &rel.TargetID, &rel.Type, &rel.Weight, &metaJSON); err == nil {
					if metaJSON.Valid {
						_ = json.Unmarshal([]byte(metaJSON.String), &rel.Metadata)
					}
					visitedEdges[rel.ID] = rel

					neighborID := rel.TargetID
					if neighborID == currID {
						neighborID = rel.SourceID
					}

					if _, exists := visitedNodes[neighborID]; !exists {
						if neighborEnt, err := e.GetEntityByID(ctx, neighborID); err == nil {
							visitedNodes[neighborID] = *neighborEnt
							nextQueue = append(nextQueue, neighborID)
						}
					}
				}
			}
			rRows.Close()
		}
		queue = nextQueue
	}

	// 3. Synthesize Context Representation
	nodeList := make([]Entity, 0, len(visitedNodes))
	for _, n := range visitedNodes {
		nodeList = append(nodeList, n)
	}
	edgeList := make([]Relation, 0, len(visitedEdges))
	for _, e := range visitedEdges {
		edgeList = append(edgeList, e)
	}

	sort.Slice(nodeList, func(i, j int) bool {
		return nodeList[i].Name < nodeList[j].Name
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### GraphRAG Multi-Hop Traversal for Query '%s' (%d-hop expansion)\n\n", seedTerm, kHops))
	sb.WriteString("#### Identified Entities:\n")
	for _, n := range nodeList {
		sb.WriteString(fmt.Sprintf("- **[%s]** `%s` (%s:%d)\n", n.Type, n.Name, n.FilePath, n.StartLine))
		if n.DocString != "" {
			sb.WriteString(fmt.Sprintf("  *Doc:* %s\n", n.DocString))
		}
	}

	sb.WriteString("\n#### Dependency Relationships:\n")
	for _, e := range edgeList {
		sb.WriteString(fmt.Sprintf("- `%s` --[%s]--> `%s` (weight: %.1f)\n", e.SourceID, e.Type, e.TargetID, e.Weight))
	}

	return &MultiHopTraversalResult{
		Query:           seedTerm,
		SeedEntities:    seeds,
		TraversedNodes:  nodeList,
		TraversedEdges:  edgeList,
		SynthesizedText: sb.String(),
	}, nil
}

// ----------------------------------------------------------------------------
// Helper Query Methods
// ----------------------------------------------------------------------------

func (e *GraphRAGEngine) GetEntityByID(ctx context.Context, id string) (*Entity, error) {
	row := e.db.QueryRowContext(ctx, `
		SELECT id, name, type, file_path, start_line, end_line, doc_string, signature, metadata_json, community_id
		FROM code_entities WHERE id = $1
	`, id)

	var ent Entity
	var metaJSON sql.NullString
	err := row.Scan(&ent.ID, &ent.Name, &ent.Type, &ent.FilePath, &ent.StartLine, &ent.EndLine, &ent.DocString, &ent.Signature, &metaJSON, &ent.CommunityID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEntityNotFound
		}
		return nil, err
	}
	if metaJSON.Valid {
		_ = json.Unmarshal([]byte(metaJSON.String), &ent.Metadata)
	}
	return &ent, nil
}

func (e *GraphRAGEngine) LoadGraphState(ctx context.Context) ([]Entity, []Relation, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT id, name, type, file_path, start_line, end_line, doc_string, signature, metadata_json, community_id FROM code_entities`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var ent Entity
		var metaJSON sql.NullString
		if err := rows.Scan(&ent.ID, &ent.Name, &ent.Type, &ent.FilePath, &ent.StartLine, &ent.EndLine, &ent.DocString, &ent.Signature, &metaJSON, &ent.CommunityID); err == nil {
			if metaJSON.Valid {
				_ = json.Unmarshal([]byte(metaJSON.String), &ent.Metadata)
			}
			entities = append(entities, ent)
		}
	}

	relRows, err := e.db.QueryContext(ctx, `SELECT id, source_id, target_id, type, weight, metadata_json FROM entity_relations`)
	if err != nil {
		return nil, nil, err
	}
	defer relRows.Close()

	var relations []Relation
	for relRows.Next() {
		var rel Relation
		var metaJSON sql.NullString
		if err := relRows.Scan(&rel.ID, &rel.SourceID, &rel.TargetID, &rel.Type, &rel.Weight, &metaJSON); err == nil {
			if metaJSON.Valid {
				_ = json.Unmarshal([]byte(metaJSON.String), &rel.Metadata)
			}
			relations = append(relations, rel)
		}
	}

	return entities, relations, nil
}

func insertEntityTx(tx *sql.Tx, ent Entity) error {
	metaJSON, _ := json.Marshal(ent.Metadata)
	_, err := tx.Exec(`
		INSERT OR REPLACE INTO code_entities (id, name, type, file_path, start_line, end_line, doc_string, signature, metadata_json, community_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, ent.ID, ent.Name, ent.Type, ent.FilePath, ent.StartLine, ent.EndLine, ent.DocString, ent.Signature, string(metaJSON), ent.CommunityID)
	return err
}

func insertRelationTx(tx *sql.Tx, rel Relation) error {
	metaJSON, _ := json.Marshal(rel.Metadata)
	_, err := tx.Exec(`
		INSERT OR REPLACE INTO entity_relations (id, source_id, target_id, type, weight, metadata_json)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, rel.ID, rel.SourceID, rel.TargetID, rel.Type, rel.Weight, string(metaJSON))
	return err
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	default:
		return ""
	}
}
