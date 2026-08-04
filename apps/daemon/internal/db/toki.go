package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Common TOKI Engine errors.
var (
	ErrClockRegression         = errors.New("toki: LEVC clock regression detected - update violates WAL monotonicity fence")
	ErrMemoryNotFound          = errors.New("toki: memory record not found")
	ErrInvalidBitemporalRange  = errors.New("toki: invalid bitemporal range (valid_to must be after valid_from)")
	ErrContradictionUnresolved = errors.New("toki: semantic contradiction could not be resolved")
)

// ContradictionStrategy defines how TOKI handles semantic contradiction resolutions per ADR-04.
type ContradictionStrategy int

const (
	// StrategyOverwriteAudit demotes old belief to Audit table and replaces Current with new assertion.
	StrategyOverwriteAudit ContradictionStrategy = iota
	// StrategySemiringFusion merges provenance polynomials via K-semiring addition (+) and weight scaling (*).
	StrategySemiringFusion
	// StrategyBranchVersion preserves both assertions in Current with distinct attribute sub-keys.
	StrategyBranchVersion
)

// AgentMemory represents an active assertion in the Current table.
type AgentMemory struct {
	MemoryID             string          `json:"memory_id" db:"memory_id"`
	AgentID              string          `json:"agent_id" db:"agent_id"`
	EntityID             string          `json:"entity_id" db:"entity_id"`
	AttributeKey         string          `json:"attribute_key" db:"attribute_key"`
	AttributeValue       json.RawMessage `json:"attribute_value" db:"attribute_value"`
	Metadata             json.RawMessage `json:"metadata" db:"metadata"`
	ValidFrom            time.Time       `json:"valid_from" db:"valid_from"`
	ValidTo              time.Time       `json:"valid_to" db:"valid_to"`
	SystemFrom           time.Time       `json:"system_from" db:"system_from"`
	SystemTo             time.Time       `json:"system_to" db:"system_to"`
	Clock                LEVC            `json:"clock"`
	ProvenancePolynomial string          `json:"provenance_polynomial" db:"provenance_polynomial"`
	ConfidenceScore      float64         `json:"confidence_score" db:"confidence_score"`
	DerivationPath       []string        `json:"derivation_path" db:"derivation_path"`
	SourceURI            string          `json:"source_uri" db:"source_uri"`
	PayloadSignature     []byte          `json:"payload_signature" db:"payload_signature"`
}

// AuditRecord represents a historic or superseded assertion in the Audit table.
type AuditRecord struct {
	AuditID             string      `json:"audit_id" db:"audit_id"`
	Memory              AgentMemory `json:"memory"`
	SupersededByClock   LEVC        `json:"superseded_by_clock"`
	DemotedAt           time.Time   `json:"demoted_at" db:"demoted_at"`
	DemotionReason      string      `json:"demotion_reason" db:"demotion_reason"`
	ReplacedByMemoryID *string      `json:"replaced_by_memory_id,omitempty" db:"replaced_by_memory_id"`
}

// BitemporalFilter options for point-in-time and range queries.
type BitemporalFilter struct {
	AgentID        string    `json:"agent_id"`
	EntityID       string    `json:"entity_id"`
	AttributeKey   string    `json:"attribute_key"`
	AsOfValid      time.Time `json:"as_of_valid"`
	AsOfSystem     time.Time `json:"as_of_system"`
	MinConfidence  float64   `json:"min_confidence"`
	IncludeAudit   bool      `json:"include_audit"`
}

// MonotonicWALFence thread-safely enforces strict LEVC monotonicity during WAL replay or updates per ADR-13.
type MonotonicWALFence struct {
	mu        sync.RWMutex
	lastClock LEVC
}

// NewMonotonicWALFence creates a new fence initialized with a baseline clock.
func NewMonotonicWALFence(initialClock LEVC) *MonotonicWALFence {
	return &MonotonicWALFence{
		lastClock: initialClock,
	}
}

// ValidateAndAdvance checks if incoming LEVC is strictly monotonic relative to lastClock.
// If valid, updates lastClock and returns nil. Otherwise returns ErrClockRegression.
func (f *MonotonicWALFence) ValidateAndAdvance(incoming LEVC) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !incoming.IsMonotonicAfter(f.lastClock) {
		return fmt.Errorf("%w: incoming clock %s <= fence clock %s", ErrClockRegression, incoming, f.lastClock)
	}

	f.lastClock = incoming
	return nil
}

// GetLastClock returns a copy of the fence's current highest LEVC.
func (f *MonotonicWALFence) GetLastClock() LEVC {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.lastClock
}

// TOKIEngine manages PostgreSQL bitemporal memory storage, LEVC monotonicity fences, and K-semiring provenance.
type TOKIEngine struct {
	db    *sql.DB
	fence *MonotonicWALFence
}

// NewTOKIEngine constructs a TOKIEngine instance wrapping a PostgreSQL connection pool.
func NewTOKIEngine(db *sql.DB, initialClock LEVC) *TOKIEngine {
	return &TOKIEngine{
		db:    db,
		fence: NewMonotonicWALFence(initialClock),
	}
}

// GetFence returns the engine's active MonotonicWALFence.
func (e *TOKIEngine) GetFence() *MonotonicWALFence {
	return e.fence
}

// UpsertMemory applies a new memory assertion or updates an existing memory assertion in PostgreSQL.
// Performs atomic dual-row demotion (Current -> Audit) and validates clock monotonicity against the WAL fence.
func (e *TOKIEngine) UpsertMemory(ctx context.Context, mem *AgentMemory) error {
	if mem == nil {
		return errors.New("toki: memory record cannot be nil")
	}

	// Default infinity for valid_to if zero
	if mem.ValidTo.IsZero() {
		mem.ValidTo = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}
	if mem.ValidFrom.IsZero() {
		mem.ValidFrom = time.Now().UTC()
	}
	if mem.ValidTo.Before(mem.ValidFrom) {
		return ErrInvalidBitemporalRange
	}

	// 1. Enforce Monotonic WAL Replay Fence
	if err := e.fence.ValidateAndAdvance(mem.Clock); err != nil {
		return err
	}

	if mem.SystemFrom.IsZero() {
		mem.SystemFrom = time.Now().UTC()
	}
	mem.SystemTo = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

	derivationJSON, err := json.Marshal(mem.DerivationPath)
	if err != nil {
		derivationJSON = []byte("[]")
	}

	// 2. Begin SQL Transaction for Dual-Row Atomicity
	tx, err := e.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("toki: failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	// 3. Check for existing active record in Current table
	selectQuery := `
		SELECT memory_id, valid_from, valid_to, system_from, levc_epoch_id, levc_lsn, levc_transaction_id,
		       provenance_polynomial, confidence_score, derivation_path, source_uri, payload_signature
		FROM agent_memories_current
		WHERE agent_id = $1 AND entity_id = $2 AND attribute_key = $3
		FOR UPDATE;
	`
	var existing struct {
		MemoryID           string
		ValidFrom          time.Time
		ValidTo            time.Time
		SystemFrom         time.Time
		LEVCEp             uint64
		LEVCLsn            uint64
		LEVCTx             uint64
		Provenance         string
		Confidence         float64
		DerivationPathJSON []byte
		SourceURI          sql.NullString
		PayloadSig         []byte
	}

	row := tx.QueryRowContext(ctx, selectQuery, mem.AgentID, mem.EntityID, mem.AttributeKey)
	err = row.Scan(
		&existing.MemoryID,
		&existing.ValidFrom,
		&existing.ValidTo,
		&existing.SystemFrom,
		&existing.LEVCEp,
		&existing.LEVCLsn,
		&existing.LEVCTx,
		&existing.Provenance,
		&existing.Confidence,
		&existing.DerivationPathJSON,
		&existing.SourceURI,
		&existing.PayloadSig,
	)

	now := time.Now().UTC()

	if err == nil {
		// Existing record found -> Demote to Audit Table
		auditInsert := `
			INSERT INTO agent_memories_audit (
				memory_id, agent_id, entity_id, attribute_key, attribute_value, metadata,
				valid_from, valid_to, system_from, system_to,
				levc_epoch_id, levc_lsn, levc_transaction_id,
				superseded_by_epoch_id, superseded_by_lsn, superseded_by_tx_id,
				provenance_polynomial, confidence_score, derivation_path, source_uri, payload_signature,
				demoted_at, demotion_reason, replaced_by_memory_id
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10,
				$11, $12, $13,
				$14, $15, $16,
				$17, $18, $19, $20, $21,
				$22, $23, $24
			);
		`
		var newMemID *string
		if mem.MemoryID != "" {
			newMemID = &mem.MemoryID
		}

		_, err = tx.ExecContext(ctx, auditInsert,
			existing.MemoryID, mem.AgentID, mem.EntityID, mem.AttributeKey, mem.AttributeValue, mem.Metadata,
			existing.ValidFrom, mem.ValidFrom, existing.SystemFrom, now,
			existing.LEVCEp, existing.LEVCLsn, existing.LEVCTx,
			mem.Clock.EpochID, mem.Clock.LSN, mem.Clock.TransactionID,
			existing.Provenance, existing.Confidence, existing.DerivationPathJSON, existing.SourceURI.String, existing.PayloadSig,
			now, "TOKI Bitemporal Superseded by LEVC "+mem.Clock.String(), newMemID,
		)
		if err != nil {
			return fmt.Errorf("toki: failed to insert demoted audit row: %w", err)
		}

		// Remove demoted row from Current table
		deleteCurrent := `DELETE FROM agent_memories_current WHERE memory_id = $1;`
		if _, err := tx.ExecContext(ctx, deleteCurrent, existing.MemoryID); err != nil {
			return fmt.Errorf("toki: failed to delete demoted row from current: %w", err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("toki: failed to query current memory state: %w", err)
	}

	// 4. Insert new Memory assertion into Current Table
	insertCurrent := `
		INSERT INTO agent_memories_current (
			agent_id, entity_id, attribute_key, attribute_value, metadata,
			valid_from, valid_to, system_from, system_to,
			levc_epoch_id, levc_lsn, levc_transaction_id,
			provenance_polynomial, confidence_score, derivation_path, source_uri, payload_signature
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, $15, $16, $17
		) RETURNING memory_id;
	`
	var createdID string
	err = tx.QueryRowContext(ctx, insertCurrent,
		mem.AgentID, mem.EntityID, mem.AttributeKey, mem.AttributeValue, mem.Metadata,
		mem.ValidFrom, mem.ValidTo, mem.SystemFrom, mem.SystemTo,
		mem.Clock.EpochID, mem.Clock.LSN, mem.Clock.TransactionID,
		mem.ProvenancePolynomial, mem.ConfidenceScore, derivationJSON, mem.SourceURI, mem.PayloadSignature,
	).Scan(&createdID)

	if err != nil {
		return fmt.Errorf("toki: failed to insert new current memory row: %w", err)
	}

	mem.MemoryID = createdID

	// 5. Commit Transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("toki: failed to commit memory upsert tx: %w", err)
	}

	return nil
}

// QueryBitemporal executes a point-in-time or interval bitemporal search over memory assertions.
func (e *TOKIEngine) QueryBitemporal(ctx context.Context, filter BitemporalFilter) ([]*AgentMemory, error) {
	if filter.AsOfValid.IsZero() {
		filter.AsOfValid = time.Now().UTC()
	}
	if filter.AsOfSystem.IsZero() {
		filter.AsOfSystem = time.Now().UTC()
	}

	query := `
		SELECT memory_id, agent_id, entity_id, attribute_key, attribute_value, metadata,
		       valid_from, valid_to, system_from, system_to,
		       levc_epoch_id, levc_lsn, levc_transaction_id,
		       provenance_polynomial, confidence_score, derivation_path, source_uri, payload_signature
		FROM v_agent_memories_bitemporal
		WHERE agent_id = $1 AND entity_id = $2
		  AND valid_from <= $3 AND valid_to > $3
		  AND system_from <= $4 AND system_to > $4
		  AND confidence_score >= $5
	`
	args := []interface{}{
		filter.AgentID, filter.EntityID, filter.AsOfValid, filter.AsOfSystem, filter.MinConfidence,
	}

	if filter.AttributeKey != "" {
		query += " AND attribute_key = $6"
		args = append(args, filter.AttributeKey)
	}

	query += " ORDER BY levc_epoch_id DESC, levc_lsn DESC, levc_transaction_id DESC;"

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("toki: bitemporal query failed: %w", err)
	}
	defer rows.Close()

	var results []*AgentMemory
	for rows.Next() {
		var mem AgentMemory
		var derivationJSON []byte
		var sourceURI sql.NullString

		err := rows.Scan(
			&mem.MemoryID, &mem.AgentID, &mem.EntityID, &mem.AttributeKey, &mem.AttributeValue, &mem.Metadata,
			&mem.ValidFrom, &mem.ValidTo, &mem.SystemFrom, &mem.SystemTo,
			&mem.Clock.EpochID, &mem.Clock.LSN, &mem.Clock.TransactionID,
			&mem.ProvenancePolynomial, &mem.ConfidenceScore, &derivationJSON, &sourceURI, &mem.PayloadSignature,
		)
		if err != nil {
			return nil, fmt.Errorf("toki: scan failed during bitemporal query: %w", err)
		}

		if len(derivationJSON) > 0 {
			_ = json.Unmarshal(derivationJSON, &mem.DerivationPath)
		}
		mem.SourceURI = sourceURI.String

		results = append(results, &mem)
	}

	return results, nil
}

// GetMemoryHistory retrieves the full historical audit trail for a given attribute, ordered chronologically by LEVC.
func (e *TOKIEngine) GetMemoryHistory(ctx context.Context, agentID, entityID, attributeKey string) ([]*AuditRecord, error) {
	query := `
		SELECT audit_id, memory_id, agent_id, entity_id, attribute_key, attribute_value, metadata,
		       valid_from, valid_to, system_from, system_to,
		       levc_epoch_id, levc_lsn, levc_transaction_id,
		       superseded_by_epoch_id, superseded_by_lsn, superseded_by_tx_id,
		       provenance_polynomial, confidence_score, derivation_path, source_uri, payload_signature,
		       demoted_at, demotion_reason, replaced_by_memory_id
		FROM agent_memories_audit
		WHERE agent_id = $1 AND entity_id = $2 AND attribute_key = $3
		ORDER BY levc_epoch_id ASC, levc_lsn ASC, levc_transaction_id ASC;
	`

	rows, err := e.db.QueryContext(ctx, query, agentID, entityID, attributeKey)
	if err != nil {
		return nil, fmt.Errorf("toki: failed to fetch memory history: %w", err)
	}
	defer rows.Close()

	var records []*AuditRecord
	for rows.Next() {
		var rec AuditRecord
		var derivationJSON []byte
		var sourceURI sql.NullString
		var replacedBy sql.NullString

		err := rows.Scan(
			&rec.AuditID, &rec.Memory.MemoryID, &rec.Memory.AgentID, &rec.Memory.EntityID, &rec.Memory.AttributeKey,
			&rec.Memory.AttributeValue, &rec.Memory.Metadata,
			&rec.Memory.ValidFrom, &rec.Memory.ValidTo, &rec.Memory.SystemFrom, &rec.Memory.SystemTo,
			&rec.Memory.Clock.EpochID, &rec.Memory.Clock.LSN, &rec.Memory.Clock.TransactionID,
			&rec.SupersededByClock.EpochID, &rec.SupersededByClock.LSN, &rec.SupersededByClock.TransactionID,
			&rec.Memory.ProvenancePolynomial, &rec.Memory.ConfidenceScore, &derivationJSON, &sourceURI, &rec.Memory.PayloadSignature,
			&rec.DemotedAt, &rec.DemotionReason, &replacedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("toki: scan failed during audit history query: %w", err)
		}

		if len(derivationJSON) > 0 {
			_ = json.Unmarshal(derivationJSON, &rec.Memory.DerivationPath)
		}
		rec.Memory.SourceURI = sourceURI.String
		if replacedBy.Valid {
			rec.ReplacedByMemoryID = &replacedBy.String
		}

		records = append(records, &rec)
	}

	return records, nil
}

// ResolveContradiction evaluates and resolves contradictory agent assertions according to K-semiring polynomial algebra (ADR-04).
func (e *TOKIEngine) ResolveContradiction(ctx context.Context, existing *AgentMemory, incoming *AgentMemory, strategy ContradictionStrategy) error {
	switch strategy {
	case StrategyOverwriteAudit:
		// Standard TOKI demote and overwrite
		return e.UpsertMemory(ctx, incoming)

	case StrategySemiringFusion:
		// K-Semiring Polynomial Addition & Confidence Fusion
		fusedProvenance := fmt.Sprintf("(%s) + (%s)", existing.ProvenancePolynomial, incoming.ProvenancePolynomial)
		fusedConfidence := (existing.ConfidenceScore + incoming.ConfidenceScore) / 2.0
		if fusedConfidence > 1.0 {
			fusedConfidence = 1.0
		}

		fusedDerivation := append(existing.DerivationPath, incoming.DerivationPath...)

		incoming.ProvenancePolynomial = fusedProvenance
		incoming.ConfidenceScore = fusedConfidence
		incoming.DerivationPath = fusedDerivation

		return e.UpsertMemory(ctx, incoming)

	case StrategyBranchVersion:
		// Branching version by attaching LEVC suffix to key
		incoming.AttributeKey = fmt.Sprintf("%s_v%d_%d", incoming.AttributeKey, incoming.Clock.EpochID, incoming.Clock.LSN)
		return e.UpsertMemory(ctx, incoming)

	default:
		return ErrContradictionUnresolved
	}
}
