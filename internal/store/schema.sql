-- Core orchestrator tables (no extensions required)
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    document_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    updated_at BIGINT NOT NULL,
    _deleted BOOLEAN NOT NULL DEFAULT false,
    valid_time_start BIGINT NOT NULL DEFAULT 0,
    valid_time_end BIGINT NOT NULL DEFAULT 9223372036854775807 /* BiTemporalInfinity */,
    system_time BIGINT NOT NULL DEFAULT 0,
    provenance JSONB
);

-- Index for deterministic sorting as per Phase 1.2 requirements
CREATE INDEX IF NOT EXISTS idx_documents_updated_at ON documents(updated_at DESC);

-- Audit table for TOKI Bitemporal contradiction resolution
CREATE TABLE IF NOT EXISTS documents_audit (
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
);

-- Knowledge Graph: structural codebase relationships (AST dependency edges)
CREATE TABLE IF NOT EXISTS repository_graph (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relation_type TEXT NOT NULL DEFAULT '',
    subgraph_type TEXT NOT NULL DEFAULT 'SEMANTIC',
    valid_time_start BIGINT DEFAULT 0,
    valid_time_end BIGINT DEFAULT 9223372036854775807 /* BiTemporalInfinity */,
    system_time BIGINT DEFAULT 0,
    PRIMARY KEY (source_id, target_id, relation_type, subgraph_type)
);

-- Ingestion subsystem tables (requires pgvector extension)
-- Only created when AIOF_ENABLE_INGESTION=true
CREATE EXTENSION IF NOT EXISTS vector;

-- Vector DB: semantic embeddings for hybrid RAG search (requires pgvector extension)
-- NOTE: This table is created conditionally at runtime; pgvector may not be installed.
CREATE TABLE IF NOT EXISTS repository_embeddings (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    metadata JSONB,
    embedding VECTOR(768)
);
