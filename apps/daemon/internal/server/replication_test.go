package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"aiof/internal/db"
)

func TestWALReplicationPull(t *testing.T) {
	wal := NewInMemoryWALStorage()
	initialClock := db.NewLEVC(1, 10, 1)
	engine := NewReplicationEngine(wal, initialClock)

	ctx := context.Background()

	_ = wal.AppendEntry(ctx, WALEntry{LSN: 11, EpochID: 1, TransactionID: 1, DocID: "doc-1", Action: "INSERT", Payload: json.RawMessage(`{"a":1}`), Timestamp: time.Now()})
	_ = wal.AppendEntry(ctx, WALEntry{LSN: 12, EpochID: 1, TransactionID: 1, DocID: "doc-2", Action: "INSERT", Payload: json.RawMessage(`{"b":2}`), Timestamp: time.Now()})
	_ = wal.AppendEntry(ctx, WALEntry{LSN: 13, EpochID: 1, TransactionID: 1, DocID: "doc-3", Action: "UPDATE", Payload: json.RawMessage(`{"c":3}`), Timestamp: time.Now()})

	req := PullRequest{
		Checkpoint: db.NewLEVC(1, 10, 1),
		BatchSize:  2,
	}

	resp, err := engine.HandlePull(ctx, req)
	if err != nil {
		t.Fatalf("HandlePull failed: %v", err)
	}

	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries in batch, got %d", len(resp.Entries))
	}

	if !resp.HasMore {
		t.Errorf("expected HasMore to be true")
	}

	if resp.NewCheckpoint.LSN != 12 {
		t.Errorf("expected new checkpoint LSN 12, got %d", resp.NewCheckpoint.LSN)
	}
}

func TestWALReplicationPushAndClockRegression(t *testing.T) {
	wal := NewInMemoryWALStorage()
	initialClock := db.NewLEVC(1, 100, 1)
	engine := NewReplicationEngine(wal, initialClock)

	ctx := context.Background()

	// Valid monotonic mutation
	validMut := PushMutation{
		TraceID:    "trace-1",
		Collection: "canvas_artifacts",
		DocID:      "art-100",
		Action:     "UPDATE",
		Payload:    json.RawMessage(`{"title":"New Diagram"}`),
		Clock:      db.NewLEVC(1, 101, 1),
	}

	// Regressed clock mutation (same or lower clock)
	regressedMut := PushMutation{
		TraceID:    "trace-2",
		Collection: "canvas_artifacts",
		DocID:      "art-101",
		Action:     "UPDATE",
		Payload:    json.RawMessage(`{"title":"Stale Diagram"}`),
		Clock:      db.NewLEVC(1, 100, 1),
	}

	req := PushRequest{
		Mutations: []PushMutation{validMut, regressedMut},
	}

	resp, err := engine.HandlePush(ctx, req)
	if err != nil {
		t.Fatalf("HandlePush failed: %v", err)
	}

	if resp.AcceptedCount != 1 {
		t.Errorf("expected 1 accepted mutation, got %d", resp.AcceptedCount)
	}

	if resp.Results[0].Status != StatusAccepted {
		t.Errorf("expected first mutation to be ACCEPTED, got %s", resp.Results[0].Status)
	}

	if resp.Results[1].Status != StatusClockRegression {
		t.Errorf("expected second mutation to be REJECTED_CLOCK_REGRESSION, got %s", resp.Results[1].Status)
	}
}
