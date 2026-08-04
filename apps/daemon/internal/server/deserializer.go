package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
)

var (
	// ErrMaxDepthExceeded is returned when the JSON schema exceeds the allowed nesting depth (max 2 layers per ADR-05).
	ErrMaxDepthExceeded = errors.New("json schema nesting depth exceeds maximum allowed 2 layers")
	// ErrInvalidEnvelope is returned when the top-level JSON envelope cannot be parsed.
	ErrInvalidEnvelope = errors.New("invalid request envelope format")
	// ErrEmptyPayload is returned when the raw message payload is empty or null.
	ErrEmptyPayload = errors.New("payload cannot be empty")
	// ErrSubagentPanic is returned when a panic occurs during sub-agent payload deserialization.
	ErrSubagentPanic = errors.New("sub-agent payload parsing panic recovered")
)

// RequestEnvelope represents the standardized Pass-1 top-level request container.
// It defers heavy payload parsing by capturing the payload as raw bytes (json.RawMessage).
type RequestEnvelope struct {
	TraceID   string            `json:"trace_id"`
	Action    string            `json:"action"`
	AgentID   string            `json:"agent_id"`
	SessionID string            `json:"session_id"`
	Timestamp int64             `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Payload   json.RawMessage   `json:"payload"`
}

// TwoPassDeserializer provides high-performance, fault-isolated JSON parsing per ADR-05.
type TwoPassDeserializer struct {
	maxNestingDepth int
}

// NewTwoPassDeserializer constructs a deserializer enforcing the specified maximum nesting depth.
func NewTwoPassDeserializer(maxDepth int) *TwoPassDeserializer {
	if maxDepth <= 0 {
		maxDepth = 2 // Default to 2 layers as specified in ADR-05
	}
	return &TwoPassDeserializer{
		maxNestingDepth: maxDepth,
	}
}

// CheckDepth inspects the raw JSON tokens to enforce the max nesting depth constraint.
// Max depth of 2 permits: Layer 1 (Envelope Object) -> Layer 2 (Field Objects/Arrays).
func (d *TwoPassDeserializer) CheckDepth(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	depth := 0

	for {
		t, err := dec.Token()
		if err != nil {
			break
		}

		switch v := t.(type) {
		case json.Delim:
			if v == json.Delim('{') || v == json.Delim('[') {
				depth++
				if depth > d.maxNestingDepth {
					return fmt.Errorf("%w: current depth %d > max %d", ErrMaxDepthExceeded, depth, d.maxNestingDepth)
				}
			} else if v == json.Delim('}') || v == json.Delim(']') {
				depth--
			}
		}
	}
	return nil
}

// Pass1Unmarshal validates JSON nesting depth and unmarshals the top-level envelope.
// This defers parsing of sub-agent payload bytes, preventing unnecessary GC allocations.
func (d *TwoPassDeserializer) Pass1Unmarshal(data []byte) (*RequestEnvelope, error) {
	if len(data) == 0 {
		return nil, ErrInvalidEnvelope
	}

	if err := d.CheckDepth(data); err != nil {
		return nil, err
	}

	var env RequestEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidEnvelope, err)
	}

	if len(env.Payload) == 0 || string(env.Payload) == "null" {
		return nil, ErrEmptyPayload
	}

	return &env, nil
}

// Pass2Unmarshal safely unmarshals the raw message payload into a target typed struct T.
// Includes panic recovery to isolate sub-agent parsing panics from the main routing loop.
func Pass2Unmarshal[T any](rawPayload json.RawMessage) (result *T, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			err = fmt.Errorf("%w: %v\nstack: %s", ErrSubagentPanic, r, string(stack))
		}
	}()

	if len(rawPayload) == 0 || string(rawPayload) == "null" {
		return nil, ErrEmptyPayload
	}

	var target T
	if err := json.Unmarshal(rawPayload, &target); err != nil {
		return nil, fmt.Errorf("pass 2 payload unmarshal error: %w", err)
	}

	return &target, nil
}

// ParseTwoPass executes combined Pass-1 envelope extraction and Pass-2 typed payload parsing.
func ParseTwoPass[T any](d *TwoPassDeserializer, data []byte) (*RequestEnvelope, *T, error) {
	env, err := d.Pass1Unmarshal(data)
	if err != nil {
		return nil, nil, err
	}

	payload, err := Pass2Unmarshal[T](env.Payload)
	if err != nil {
		return env, nil, err
	}

	return env, payload, nil
}
