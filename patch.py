import re
import os

# 1. Patch store.go
with open('internal/store/store.go', 'r') as f:
    content = f.read()

# a) Define BiTemporalInfinity constant and imports
content = content.replace(
    '"github.com/jackc/pgx/v5/pgxpool"',
    '"github.com/jackc/pgx/v5"\n\t"github.com/jackc/pgx/v5/pgxpool"'
)
content = content.replace(
    'import (',
    '// BiTemporalInfinity represents an unbounded valid-time end (max int64).\nconst BiTemporalInfinity int64 = 9223372036854775807\n\nimport ('
)

# b) Replace hardcoded 9223372036854775807 with BiTemporalInfinity (or in strings using fmt.Sprintf)
content = content.replace('valid_time_end BIGINT NOT NULL DEFAULT 9223372036854775807', 'valid_time_end BIGINT NOT NULL DEFAULT %d`, BiTemporalInfinity)')
content = content.replace('`CREATE TABLE IF NOT EXISTS documents (', 'fmt.Sprintf(`CREATE TABLE IF NOT EXISTS documents (')
content = content.replace('`ALTER TABLE documents ADD COLUMN IF NOT EXISTS valid_time_end BIGINT NOT NULL DEFAULT %d`, BiTemporalInfinity)', 'fmt.Sprintf(`ALTER TABLE documents ADD COLUMN IF NOT EXISTS valid_time_end BIGINT NOT NULL DEFAULT %d`, BiTemporalInfinity)')

content = content.replace('valid_time_end BIGINT DEFAULT 9223372036854775807', 'valid_time_end BIGINT DEFAULT %d`, BiTemporalInfinity)')
content = content.replace('`ALTER TABLE repository_graph ADD COLUMN IF NOT EXISTS valid_time_end BIGINT DEFAULT %d`, BiTemporalInfinity)', 'fmt.Sprintf(`ALTER TABLE repository_graph ADD COLUMN IF NOT EXISTS valid_time_end BIGINT DEFAULT %d`, BiTemporalInfinity)')

content = content.replace('doc.ValidTimeEnd = 9223372036854775807', 'doc.ValidTimeEnd = BiTemporalInfinity')

# c) Propagate ensureSchema errors fatally
content = content.replace('s.ensureSchema(ctx)', 'if err := s.ensureSchema(ctx); err != nil {\n\t\treturn nil, fmt.Errorf("schema validation failed: %w", err)\n\t}')
content = content.replace('func (s *Store) ensureSchema(ctx context.Context) {', 'func (s *Store) ensureSchema(ctx context.Context) error {')
content = content.replace('slog.Warn("ensureSchema: failed to execute core DDL", slog.String("error", err.Error()))', 'return fmt.Errorf("ensureSchema: failed to execute core DDL: %w", err)')
content = content.replace('slog.Warn("ensureSchema: repository_embeddings table creation failed (pgvector may not be installed)",\n\t\t\tslog.String("error", err.Error()))', 'return fmt.Errorf("ensureSchema: repository_embeddings table creation failed: %w", err)')
content = content.replace('	if _, err := s.pool.Exec(ctx, embeddingsDDL); err != nil {\n\t\treturn fmt.Errorf("ensureSchema: repository_embeddings table creation failed: %w", err)\n\t}\n}', '	if _, err := s.pool.Exec(ctx, embeddingsDDL); err != nil {\n\t\treturn fmt.Errorf("ensureSchema: repository_embeddings table creation failed: %w", err)\n\t}\n\n\treturn nil\n}')

# d) Add backpressure logging
content = content.replace('default:\n\t\t}', 'default:\n\t\t\tslog.Warn("subscriber channel full, mutation dropped", "doc_id", doc.ID, "collection", doc.DocumentType)\n\t\t}')

# e) Log deferred tx.Rollback errors
content = content.replace('defer tx.Rollback(ctx)', 'defer func() {\n\t\tif err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {\n\t\t\tslog.Warn("rollback failed", "error", err)\n\t\t}\n\t}()')

# f) Default Provenance logging
content = content.replace('doc.Provenance = json.RawMessage(fmt.Sprintf(`{"source":"agent","confidence":1.0,"timestamp":%d}`, time.Now().UnixMilli()))\n\t}', 'doc.Provenance = json.RawMessage(fmt.Sprintf(`{"source":"agent","confidence":1.0,"timestamp":%d}`, time.Now().UnixMilli()))\n\t\tslog.Debug("provenance auto-generated", "doc_id", doc.ID)\n\t}')

with open('internal/store/store.go', 'w') as f:
    f.write(content)

# 2. Patch schema.sql
with open('internal/store/schema.sql', 'r') as f:
    content = f.read()

content = '-- Core orchestrator tables (no extensions required)\n' + content
content = content.replace('9223372036854775807', '9223372036854775807 /* BiTemporalInfinity */')

content = content.replace('-- Vector DB: semantic embeddings for hybrid RAG search (requires pgvector extension)\n-- NOTE: This table is created conditionally at runtime; pgvector may not be installed.\nCREATE TABLE IF NOT EXISTS repository_embeddings', '-- Ingestion subsystem tables (requires pgvector extension)\n-- Only created when AIOF_ENABLE_INGESTION=true\nCREATE EXTENSION IF NOT EXISTS vector;\n\n-- Vector DB: semantic embeddings for hybrid RAG search (requires pgvector extension)\n-- NOTE: This table is created conditionally at runtime; pgvector may not be installed.\nCREATE TABLE IF NOT EXISTS repository_embeddings')

with open('internal/store/schema.sql', 'w') as f:
    f.write(content)

# 3. Patch store_test.go
with open('internal/store/store_test.go', 'r') as f:
    content = f.read()

content = content.replace('"path/filepath"', '"path/filepath"\n\t"runtime"')
content = content.replace('''	schemaBytes, err := os.ReadFile("schema.sql")
	if err != nil {
		// Try relative to project root
		schemaBytes, err = os.ReadFile(filepath.Join("..", "..", "internal", "store", "schema.sql"))
		require.NoError(t, err)
	}''', '''	_, filename, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(filename), "schema.sql")
	schemaBytes, err := os.ReadFile(schemaPath)
	require.NoError(t, err)''')

with open('internal/store/store_test.go', 'w') as f:
    f.write(content)

# 4. Patch ebpf.go
with open('internal/telemetry/ebpf.go', 'r') as f:
    content = f.read()

content = content.replace('reader     *perf.Reader', 'reader     *perf.Reader\n\tattached   bool')
content = content.replace('e.logger.Info("Successfully attached tracepoints for syscall boundary monitoring")\n\treturn nil', 'e.attached = true\n\te.logger.Info("Successfully attached tracepoints for syscall boundary monitoring")\n\treturn nil')
content = content.replace('func (e *EBPFHook) Correlate(ctx context.Context) error {\n\te.logger.Info("Starting eBPF to OTEL telemetry correlation pipeline")', 'func (e *EBPFHook) Correlate(ctx context.Context) error {\n\tif e.reader == nil {\n\t\treturn fmt.Errorf("correlate: perf reader is nil, call Attach() before Correlate()")\n\t}\n\tif !e.attached {\n\t\treturn fmt.Errorf("correlate: ebpf not attached")\n\t}\n\n\te.logger.Info("Starting eBPF to OTEL telemetry correlation pipeline")')

with open('internal/telemetry/ebpf.go', 'w') as f:
    f.write(content)

