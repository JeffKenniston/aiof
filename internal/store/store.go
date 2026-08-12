package store

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BiTemporalInfinity represents an unbounded valid-time end (max int64).
const BiTemporalInfinity int64 = 9223372036854775807

// Document represents a state mutation recorded in the WAL.
// ADR-02 / Phase 1.2: requires deterministic sorting (UpdatedAt) and _deleted flags.
type Document struct {
	ID             string
	DocumentType   string
	Payload        json.RawMessage
	UpdatedAt      int64
	IsDeleted      bool
	ValidTimeStart int64
	ValidTimeEnd   int64
	SystemTime     int64
	Provenance     json.RawMessage // K-semiring provenance
}

// Store handles PostgreSQL persistence and real-time pub/sub streaming.
type Store struct {
	pool        *pgxpool.Pool
	mu          sync.RWMutex
	subscribers map[chan Document]struct{}
}

// NewStore initializes a new PostgreSQL connection pool and returns the Store.
func NewStore(ctx context.Context, connString string) (*Store, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}
	config.MaxConns = 50
	config.MinConns = 10
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Store{
		pool:        pool,
		subscribers: make(map[chan Document]struct{}),
	}

	if err := s.ensureSchema(ctx); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	return s, nil
}

// ensureSchema runs the critical DDL statements on startup so tables exist.
// Non-critical failures (e.g. pgvector not installed) are logged as warnings.
func (s *Store) ensureSchema(ctx context.Context) error {
	// Core tables that do not require extensions.
	coreDDL := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			document_type TEXT NOT NULL,
			payload JSONB NOT NULL,
			updated_at BIGINT NOT NULL,
			_deleted BOOLEAN NOT NULL DEFAULT false,
			valid_time_start BIGINT NOT NULL DEFAULT 0,
			valid_time_end BIGINT NOT NULL DEFAULT %d,
			system_time BIGINT NOT NULL DEFAULT 0,
			provenance JSONB
		)`, BiTemporalInfinity),
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS valid_time_start BIGINT NOT NULL DEFAULT 0`,
		fmt.Sprintf(`ALTER TABLE documents ADD COLUMN IF NOT EXISTS valid_time_end BIGINT NOT NULL DEFAULT %d`, BiTemporalInfinity),
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS system_time BIGINT NOT NULL DEFAULT 0`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS provenance JSONB`,
		`CREATE INDEX IF NOT EXISTS idx_documents_updated_at ON documents(updated_at DESC)`,
		`CREATE TABLE IF NOT EXISTS documents_audit (
			audit_id SERIAL PRIMARY KEY,
			id TEXT NOT NULL,
			document_type TEXT NOT NULL,
			payload JSONB NOT NULL,
			updated_at BIGINT NOT NULL,
			_deleted BOOLEAN NOT NULL,
			valid_time_start BIGINT NOT NULL,
			valid_time_end BIGINT NOT NULL,
			system_time BIGINT NOT NULL,
			provenance JSONB,
			demoted_at BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS repository_graph (
			source_id TEXT NOT NULL,
			target_id TEXT NOT NULL,
			relation_type TEXT NOT NULL DEFAULT '',
			subgraph_type TEXT NOT NULL DEFAULT 'SEMANTIC',
			PRIMARY KEY (source_id, target_id, relation_type, subgraph_type)
		)`,
		`ALTER TABLE repository_graph ADD COLUMN IF NOT EXISTS subgraph_type TEXT NOT NULL DEFAULT 'SEMANTIC'`,
		`ALTER TABLE repository_graph ADD COLUMN IF NOT EXISTS valid_time_start BIGINT DEFAULT 0`,
		fmt.Sprintf(`ALTER TABLE repository_graph ADD COLUMN IF NOT EXISTS valid_time_end BIGINT DEFAULT %d`, BiTemporalInfinity),
		`ALTER TABLE repository_graph ADD COLUMN IF NOT EXISTS system_time BIGINT DEFAULT 0`,
	}

	for _, ddl := range coreDDL {
		if _, err := s.pool.Exec(ctx, ddl); err != nil {
			return fmt.Errorf("ensureSchema: failed to execute core DDL: %w", err)
		}
	}

	// repository_embeddings requires the pgvector extension; log and continue if unavailable.
	embeddingsDDL := `CREATE TABLE IF NOT EXISTS repository_embeddings (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		metadata JSONB,
		embedding VECTOR(768)
	)`
	if _, err := s.pool.Exec(ctx, embeddingsDDL); err != nil {
		return fmt.Errorf("ensureSchema: repository_embeddings table creation failed: %w", err)
	}

	return nil
}

// Close gracefully closes the connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// Pool exposes the underlying connection pool for advanced queries.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// WriteMutation records a state mutation to the Write-Ahead Log table (documents).
func (s *Store) WriteMutation(ctx context.Context, doc Document) error {
	if doc.Provenance == nil {
		doc.Provenance = json.RawMessage(fmt.Sprintf(`{"source":"agent","confidence":1.0,"timestamp":%d}`, time.Now().UnixMilli()))
		slog.Debug("provenance auto-generated", "doc_id", doc.ID)
	}
	now := time.Now().UnixNano()
	if doc.UpdatedAt == 0 {
		doc.UpdatedAt = now
	}
	if doc.SystemTime == 0 {
		doc.SystemTime = now
	}
	if doc.ValidTimeStart == 0 {
		doc.ValidTimeStart = now
	}
	if doc.ValidTimeEnd == 0 {
		doc.ValidTimeEnd = BiTemporalInfinity
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			slog.Warn("rollback failed", "error", err)
		}
	}()

	auditQuery := `
		INSERT INTO documents_audit (id, document_type, payload, updated_at, _deleted, valid_time_start, valid_time_end, system_time, provenance, demoted_at)
		SELECT id, document_type, payload, updated_at, _deleted, valid_time_start, $1, system_time, provenance, $2
		FROM documents WHERE id = $3;
	`
	_, err = tx.Exec(ctx, auditQuery, doc.ValidTimeStart, now, doc.ID)
	if err != nil {
		return fmt.Errorf("failed to demote to audit row: %w", err)
	}

	query := `
		INSERT INTO documents (id, document_type, payload, updated_at, _deleted, valid_time_start, valid_time_end, system_time, provenance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			document_type = EXCLUDED.document_type,
			payload = EXCLUDED.payload,
			updated_at = EXCLUDED.updated_at,
			_deleted = EXCLUDED._deleted,
			valid_time_start = EXCLUDED.valid_time_start,
			valid_time_end = EXCLUDED.valid_time_end,
			system_time = EXCLUDED.system_time,
			provenance = EXCLUDED.provenance;
	`
	
	_, err = tx.Exec(ctx, query, doc.ID, doc.DocumentType, doc.Payload, doc.UpdatedAt, doc.IsDeleted, doc.ValidTimeStart, doc.ValidTimeEnd, doc.SystemTime, doc.Provenance)
	if err != nil {
		return fmt.Errorf("failed to write mutation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	s.mu.RLock()
	for ch := range s.subscribers {
		select {
		case ch <- doc:
		default:
			slog.Warn("subscriber channel full, mutation dropped", "doc_id", doc.ID, "collection", doc.DocumentType)
		}
	}
	s.mu.RUnlock()

	return nil
}

// Subscribe returns a channel that receives newly written documents.
func (s *Store) Subscribe() (<-chan Document, func()) {
	ch := make(chan Document, 100)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		delete(s.subscribers, ch)
		close(ch)
		s.mu.Unlock()
	}
}

// StreamMutations returns a Go 1.23 pull-based iterator for real-time document mutations.
// It yields the UpdatedAt timestamp and the Document itself.
func (s *Store) StreamMutations(ctx context.Context) iter.Seq2[int64, Document] {
	return func(yield func(int64, Document) bool) {
		sub, unsubscribe := s.Subscribe()
		defer unsubscribe()

		for {
			select {
			case <-ctx.Done():
				return
			case doc, ok := <-sub:
				if !ok {
					return
				}
				if !yield(doc.UpdatedAt, doc) {
					return
				}
			}
		}
	}
}

// ReadMutations fetches a range of documents, enforcing deterministic sorting by UpdatedAt.
func (s *Store) ReadMutations(ctx context.Context, since int64, limit int) ([]Document, error) {
	query := `
		SELECT id, document_type, payload, updated_at, _deleted, valid_time_start, valid_time_end, system_time, provenance
		FROM documents
		WHERE updated_at > $1
		ORDER BY updated_at ASC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read mutations: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.DocumentType, &doc.Payload, &doc.UpdatedAt, &doc.IsDeleted, &doc.ValidTimeStart, &doc.ValidTimeEnd, &doc.SystemTime, &doc.Provenance); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return docs, nil
}
