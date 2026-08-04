package orchestrator

import (
	"context"
	"encoding/json"
	"iter"
)

// RequestPayload represents the flat JSON schema payload limited to two layers deep.
// ADR-04 mandates Two-Pass Deserialization utilizing json.RawMessage.
type RequestPayload struct {
	Action string          `json:"action"`
	Params json.RawMessage `json:"params"`
}

// Event represents a streaming token output, state mutation, or agent response.
// ADR-02 requires documents to utilize _deleted boolean flags.
type Event struct {
	ID        string
	Token     string
	NodeID    string
	IsDeleted bool
}

// EventStream is an iterator yielding events and errors, utilizing Go 1.23 iter.Seq2 semantics.
type EventStream iter.Seq2[Event, error]

// Middleware defines the Go 1.23 interface for orchestrating asynchronous agent message bus communication.
// It intercepts incoming requests and token streams, routing them dynamically.
type Middleware interface {
	// HandleStream processes an incoming payload and streams back token events using iter.Seq2.
	HandleStream(ctx context.Context, payload RequestPayload) EventStream
}
