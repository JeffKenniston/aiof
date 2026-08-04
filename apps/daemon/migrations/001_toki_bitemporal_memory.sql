-- ============================================================================
-- TOKI Bitemporal Memory Engine Schema Migration
-- Migration ID: 001_toki_bitemporal_memory.sql
-- Architecture Specification: ADR-04 (Bitemporal Memory) & ADR-13 (LEVC Clocks)
-- Target Database: PostgreSQL 14+
-- ============================================================================

BEGIN;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- ----------------------------------------------------------------------------
-- 1. Table: agent_memories_current (Current Row Table)
-- Represents active, currently valid knowledge assertions for agents.
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_memories_current (
    memory_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id                VARCHAR(255) NOT NULL,
    entity_id               VARCHAR(255) NOT NULL,
    attribute_key           VARCHAR(255) NOT NULL,
    attribute_value         JSONB NOT NULL,
    metadata                JSONB DEFAULT '{}'::jsonb,

    -- Bitemporal Dimensions
    -- Valid-time: Period during which the assertion is true in the real world
    valid_from              TIMESTAMPTZ NOT NULL,
    valid_to                TIMESTAMPTZ NOT NULL DEFAULT 'infinity'::timestamptz,

    -- System-time: Period during which the assertion was recorded in the database
    system_from             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    system_to               TIMESTAMPTZ NOT NULL DEFAULT 'infinity'::timestamptz,

    -- Logical Epoch Vector Clock (LEVC) Fields
    levc_epoch_id           BIGINT NOT NULL,
    levc_lsn                BIGINT NOT NULL,
    levc_transaction_id     BIGINT NOT NULL,

    -- K-Semiring Provenance & Cryptographic Attestation Columns
    provenance_polynomial   TEXT NOT NULL,
    confidence_score        DOUBLE PRECISION NOT NULL DEFAULT 1.0 CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    derivation_path         JSONB DEFAULT '[]'::jsonb,
    source_uri              VARCHAR(1024),
    payload_signature       BYTEA,

    -- Table Constraints
    CONSTRAINT chk_valid_time_interval CHECK (valid_to > valid_from),
    CONSTRAINT chk_system_time_interval CHECK (system_to > system_from),
    CONSTRAINT uq_current_agent_entity_attr UNIQUE (agent_id, entity_id, attribute_key)
);

-- ----------------------------------------------------------------------------
-- 2. Table: agent_memories_audit (Audit Row Table)
-- Preserves historic, superseded, or overwritten facts for auditability.
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_memories_audit (
    audit_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id               UUID NOT NULL,
    agent_id                VARCHAR(255) NOT NULL,
    entity_id               VARCHAR(255) NOT NULL,
    attribute_key           VARCHAR(255) NOT NULL,
    attribute_value         JSONB NOT NULL,
    metadata                JSONB DEFAULT '{}'::jsonb,

    -- Bitemporal Dimensions (Snapshot at time of audit demotion)
    valid_from              TIMESTAMPTZ NOT NULL,
    valid_to                TIMESTAMPTZ NOT NULL,
    system_from             TIMESTAMPTZ NOT NULL,
    system_to               TIMESTAMPTZ NOT NULL,

    -- Original LEVC Fields at time of creation
    levc_epoch_id           BIGINT NOT NULL,
    levc_lsn                BIGINT NOT NULL,
    levc_transaction_id     BIGINT NOT NULL,

    -- Demotion / Superseded LEVC Fields
    superseded_by_epoch_id  BIGINT NOT NULL,
    superseded_by_lsn       BIGINT NOT NULL,
    superseded_by_tx_id     BIGINT NOT NULL,

    -- K-Semiring Provenance & Cryptographic Attestation Columns
    provenance_polynomial   TEXT NOT NULL,
    confidence_score        DOUBLE PRECISION NOT NULL,
    derivation_path         JSONB DEFAULT '[]'::jsonb,
    source_uri              VARCHAR(1024),
    payload_signature       BYTEA,

    -- Audit Metadata
    demoted_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    demotion_reason         TEXT NOT NULL,
    replaced_by_memory_id   UUID,

    CONSTRAINT chk_audit_valid_time CHECK (valid_to >= valid_from),
    CONSTRAINT chk_audit_system_time CHECK (system_to >= system_from)
);

-- ----------------------------------------------------------------------------
-- 3. Indexes for High-Performance Bitemporal & Vector Clock Queries
-- ----------------------------------------------------------------------------

-- GiST Bitemporal Range Indexes for Current Table
CREATE INDEX IF NOT EXISTS idx_curr_valid_range 
    ON agent_memories_current USING gist (tstzrange(valid_from, valid_to, '[)'));

CREATE INDEX IF NOT EXISTS idx_curr_system_range 
    ON agent_memories_current USING gist (tstzrange(system_from, system_to, '[)'));

-- Composite LEVC Ordering Indexes
CREATE INDEX IF NOT EXISTS idx_curr_levc_clock 
    ON agent_memories_current (levc_epoch_id DESC, levc_lsn DESC, levc_transaction_id DESC);

-- Business Entity Lookup Indexes
CREATE INDEX IF NOT EXISTS idx_curr_agent_entity_key 
    ON agent_memories_current (agent_id, entity_id, attribute_key);

-- GiST Bitemporal Range Indexes for Audit Table
CREATE INDEX IF NOT EXISTS idx_audit_valid_range 
    ON agent_memories_audit USING gist (tstzrange(valid_from, valid_to, '[)'));

CREATE INDEX IF NOT EXISTS idx_audit_system_range 
    ON agent_memories_audit USING gist (tstzrange(system_from, system_to, '[)'));

CREATE INDEX IF NOT EXISTS idx_audit_memory_id 
    ON agent_memories_audit (memory_id);

CREATE INDEX IF NOT EXISTS idx_audit_agent_entity_key 
    ON agent_memories_audit (agent_id, entity_id, attribute_key);

-- ----------------------------------------------------------------------------
-- 4. Unified Bitemporal View
-- Combines current and audit rows into a seamless bitemporal timeline view.
-- ----------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_agent_memories_bitemporal AS
SELECT 
    memory_id,
    'CURRENT' AS row_type,
    agent_id,
    entity_id,
    attribute_key,
    attribute_value,
    metadata,
    valid_from,
    valid_to,
    system_from,
    system_to,
    levc_epoch_id,
    levc_lsn,
    levc_transaction_id,
    provenance_polynomial,
    confidence_score,
    derivation_path,
    source_uri,
    payload_signature,
    NULL::UUID AS replaced_by_memory_id,
    NULL::TEXT AS demotion_reason
FROM agent_memories_current
UNION ALL
SELECT 
    memory_id,
    'AUDIT' AS row_type,
    agent_id,
    entity_id,
    attribute_key,
    attribute_value,
    metadata,
    valid_from,
    valid_to,
    system_from,
    system_to,
    levc_epoch_id,
    levc_lsn,
    levc_transaction_id,
    provenance_polynomial,
    confidence_score,
    derivation_path,
    source_uri,
    payload_signature,
    replaced_by_memory_id,
    demotion_reason
FROM agent_memories_audit;

COMMIT;
