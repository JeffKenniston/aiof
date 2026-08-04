package canvas

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrArtifactNotFound = errors.New("canvas: artifact not found")
	ErrVersionMismatch  = errors.New("canvas: artifact version mismatch during reactive state update")
	ErrInvalidArtifact  = errors.New("canvas: invalid artifact specification or empty payload")
)

type ArtifactType string

const (
	ArtifactSVG          ArtifactType = "SVG"
	ArtifactWireframe    ArtifactType = "REACT_WIREFRAME"
	ArtifactSpreadsheet  ArtifactType = "SPREADSHEET"
	ArtifactUMLDiagram   ArtifactType = "UML_DIAGRAM"
	ArtifactRichDocument ArtifactType = "RICH_DOCUMENT"
)

// CanvasArtifact represents a structured non-text visual artifact rendered in the client canvas per ADR-25.
type CanvasArtifact struct {
	ArtifactID   string            `json:"artifact_id"`
	ProjectID    string            `json:"project_id"`
	Title        string            `json:"title"`
	Type         ArtifactType      `json:"type"`
	Version      uint64            `json:"version"`
	Data         json.RawMessage   `json:"data"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	AuthorAgent  string            `json:"author_agent"`
	LastModified time.Time         `json:"last_modified"`
	DataHash     string            `json:"data_hash"`
}

func (a *CanvasArtifact) CalculateHash() string {
	record := fmt.Sprintf("%s|%s|%s|%d|%s|%s", a.ArtifactID, a.ProjectID, string(a.Type), a.Version, string(a.Data), a.AuthorAgent)
	h := sha256.Sum256([]byte(record))
	return hex.EncodeToString(h[:])
}

// CanvasEvent represents a reactive state mutation broadcast to RxDB client observables.
type CanvasEvent struct {
	EventID    string         `json:"event_id"`
	ArtifactID string         `json:"artifact_id"`
	Action     string         `json:"action"` // "CREATED", "UPDATED", "DELETED"
	Artifact   CanvasArtifact `json:"artifact"`
	Timestamp  time.Time      `json:"timestamp"`
}

// MultimodalCanvasEngine manages real-time collaborative workspace artifacts and reactive state streams.
type MultimodalCanvasEngine struct {
	mu          sync.RWMutex
	artifacts   map[string]*CanvasArtifact
	subscribers map[string]chan CanvasEvent
}

func NewMultimodalCanvasEngine() *MultimodalCanvasEngine {
	return &MultimodalCanvasEngine{
		artifacts:   make(map[string]*CanvasArtifact),
		subscribers: make(map[string]chan CanvasEvent),
	}
}

// UpsertArtifact creates or updates a visual canvas artifact with reactive versioning.
func (e *MultimodalCanvasEngine) UpsertArtifact(
	artifactID, projectID, title string,
	aType ArtifactType,
	data json.RawMessage,
	authorAgent string,
	metadata map[string]string,
) (*CanvasArtifact, error) {
	if artifactID == "" || len(data) == 0 {
		return nil, ErrInvalidArtifact
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	existing, exists := e.artifacts[artifactID]
	var ver uint64 = 1
	action := "CREATED"

	if exists {
		ver = existing.Version + 1
		action = "UPDATED"
	}

	art := &CanvasArtifact{
		ArtifactID:   artifactID,
		ProjectID:    projectID,
		Title:        title,
		Type:         aType,
		Version:      ver,
		Data:         data,
		Metadata:     metadata,
		AuthorAgent:  authorAgent,
		LastModified: time.Now().UTC(),
	}
	art.DataHash = art.CalculateHash()

	e.artifacts[artifactID] = art

	// Broadcast reactive canvas event
	evt := CanvasEvent{
		EventID:    fmt.Sprintf("evt-canvas-%d", time.Now().UnixNano()),
		ArtifactID: artifactID,
		Action:     action,
		Artifact:   *art,
		Timestamp:  time.Now().UTC(),
	}

	for _, ch := range e.subscribers {
		select {
		case ch <- evt:
		default:
		}
	}

	return art, nil
}

// GetArtifact retrieves a canvas artifact by ID.
func (e *MultimodalCanvasEngine) GetArtifact(artifactID string) (*CanvasArtifact, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	art, exists := e.artifacts[artifactID]
	if !exists {
		return nil, ErrArtifactNotFound
	}

	cp := *art
	return &cp, nil
}

// Subscribe returns a channel receiving real-time canvas events for client RxDB replication.
func (e *MultimodalCanvasEngine) Subscribe(subID string) chan CanvasEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch := make(chan CanvasEvent, 100)
	e.subscribers[subID] = ch
	return ch
}

// Unsubscribe closes and removes a client canvas event stream subscription.
func (e *MultimodalCanvasEngine) Unsubscribe(subID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ch, exists := e.subscribers[subID]; exists {
		close(ch)
		delete(e.subscribers, subID)
	}
}
