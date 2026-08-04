package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"aiof/internal/db"
)

var (
	ErrInvalidPullRequest = errors.New("replication: invalid pull request payload or missing checkpoint")
	ErrInvalidPushRequest = errors.New("replication: invalid push request payload")
	ErrClockRegression    = errors.New("replication: mutation rejected due to LEVC clock regression")
)

// WALEntry represents a single change record in the Write-Ahead Log (WAL) per ADR-05 and ADR-13.
type WALEntry struct {
	LSN           uint64          `json:"lsn"`
	EpochID       uint64          `json:"epoch_id"`
	TransactionID uint64          `json:"transaction_id"`
	TraceID       string          `json:"trace_id"`
	Collection    string          `json:"collection"`
	DocID         string          `json:"doc_id"`
	Action        string          `json:"action"` // "INSERT", "UPDATE", "DELETE"
	Payload       json.RawMessage `json:"payload"`
	Timestamp     time.Time       `json:"timestamp"`
}

func (w WALEntry) Clock() db.LEVC {
	return db.NewLEVC(w.EpochID, w.LSN, w.TransactionID)
}

// WALStorage defines the interface for querying and persisting Write-Ahead Log entries.
type WALStorage interface {
	GetEntriesSince(ctx context.Context, checkpoint db.LEVC, limit int) ([]WALEntry, error)
	AppendEntry(ctx context.Context, entry WALEntry) error
}

// InMemoryWALStorage is an in-memory thread-safe implementation of WALStorage for testing.
type InMemoryWALStorage struct {
	mu      sync.RWMutex
	entries []WALEntry
}

func NewInMemoryWALStorage() *InMemoryWALStorage {
	return &InMemoryWALStorage{
		entries: make([]WALEntry, 0),
	}
}

func (s *InMemoryWALStorage) GetEntriesSince(ctx context.Context, checkpoint db.LEVC, limit int) ([]WALEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []WALEntry
	for _, entry := range s.entries {
		if entry.Clock().IsMonotonicAfter(checkpoint) {
			result = append(result, entry)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *InMemoryWALStorage) AppendEntry(ctx context.Context, entry WALEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, entry)
	return nil
}

// --- Replication Engine & Handlers ---

type ReplicationEngine struct {
	wal   WALStorage
	fence *db.MonotonicWALFence
}

func NewReplicationEngine(wal WALStorage, initialClock db.LEVC) *ReplicationEngine {
	return &ReplicationEngine{
		wal:   wal,
		fence: db.NewMonotonicWALFence(initialClock),
	}
}

// Pull Protocol (ADR-05, Blueprint 5)

type PullRequest struct {
	Checkpoint db.LEVC `json:"checkpoint"`
	BatchSize  int     `json:"batch_size"`
}

type PullResponse struct {
	Entries       []WALEntry `json:"entries"`
	NewCheckpoint db.LEVC    `json:"new_checkpoint"`
	HasMore       bool       `json:"has_more"`
}

func (e *ReplicationEngine) HandlePull(ctx context.Context, req PullRequest) (*PullResponse, error) {
	batchSize := req.BatchSize
	if batchSize <= 0 || batchSize > 1000 {
		batchSize = 100
	}

	entries, err := e.wal.GetEntriesSince(ctx, req.Checkpoint, batchSize+1)
	if err != nil {
		return nil, fmt.Errorf("replication pull failed: %w", err)
	}

	hasMore := false
	if len(entries) > batchSize {
		hasMore = true
		entries = entries[:batchSize]
	}

	newCheckpoint := req.Checkpoint
	if len(entries) > 0 {
		newCheckpoint = entries[len(entries)-1].Clock()
	}

	return &PullResponse{
		Entries:       entries,
		NewCheckpoint: newCheckpoint,
		HasMore:       hasMore,
	}, nil
}

// Push Protocol (ADR-05, ADR-13)

type MutationStatus string

const (
	StatusAccepted        MutationStatus = "ACCEPTED"
	StatusClockRegression MutationStatus = "REJECTED_CLOCK_REGRESSION"
	StatusConflict        MutationStatus = "REJECTED_CONFLICT"
)

type PushMutation struct {
	TraceID    string          `json:"trace_id"`
	Collection string          `json:"collection"`
	DocID      string          `json:"doc_id"`
	Action     string          `json:"action"`
	Payload    json.RawMessage `json:"payload"`
	Clock      db.LEVC         `json:"clock"`
}

type PushRequest struct {
	Mutations []PushMutation `json:"mutations"`
}

type PushMutationResult struct {
	DocID   string         `json:"doc_id"`
	Status  MutationStatus `json:"status"`
	Clock   db.LEVC        `json:"clock"`
	Message string         `json:"message,omitempty"`
}

type PushResponse struct {
	Results       []PushMutationResult `json:"results"`
	ServerClock   db.LEVC              `json:"server_clock"`
	AcceptedCount int                  `json:"accepted_count"`
}

func (e *ReplicationEngine) HandlePush(ctx context.Context, req PushRequest) (*PushResponse, error) {
	if len(req.Mutations) == 0 {
		return &PushResponse{ServerClock: e.fence.GetLastClock()}, nil
	}

	results := make([]PushMutationResult, len(req.Mutations))
	acceptedCount := 0

	for i, mut := range req.Mutations {
		// Validate Monotonic WAL Replay Fence (ADR-13)
		err := e.fence.ValidateAndAdvance(mut.Clock)
		if err != nil {
			results[i] = PushMutationResult{
				DocID:   mut.DocID,
				Status:  StatusClockRegression,
				Clock:   mut.Clock,
				Message: fmt.Sprintf("clock regression rejected: %v", err),
			}
			continue
		}

		// Append to WAL
		walEntry := WALEntry{
			LSN:           mut.Clock.LSN,
			EpochID:       mut.Clock.EpochID,
			TransactionID: mut.Clock.TransactionID,
			TraceID:       mut.TraceID,
			Collection:    mut.Collection,
			DocID:         mut.DocID,
			Action:        mut.Action,
			Payload:       mut.Payload,
			Timestamp:     time.Now().UTC(),
		}

		if err := e.wal.AppendEntry(ctx, walEntry); err != nil {
			results[i] = PushMutationResult{
				DocID:   mut.DocID,
				Status:  StatusConflict,
				Clock:   mut.Clock,
				Message: fmt.Sprintf("wal append failed: %v", err),
			}
			continue
		}

		results[i] = PushMutationResult{
			DocID:  mut.DocID,
			Status: StatusAccepted,
			Clock:  mut.Clock,
		}
		acceptedCount++
	}

	return &PushResponse{
		Results:       results,
		ServerClock:   e.fence.GetLastClock(),
		AcceptedCount: acceptedCount,
	}, nil
}

func MakePullHTTPHandler(engine *ReplicationEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req PullRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		resp, err := engine.HandlePull(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func MakePushHTTPHandler(engine *ReplicationEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req PushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		resp, err := engine.HandlePush(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
