package store_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jackc/pgx/v5/pgxpool"

	"aiof/internal/store"
)

func TestWALPersistenceLayer(t *testing.T) {
	ctx := context.Background()

	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		t.Skip("Skipping WAL integration tests: TEST_DATABASE_URL not set (Docker/testcontainers not supported in this environment)")
	}

	t.Log("Connecting to real PostgreSQL instance for WAL integration tests...")

	// Execute Schema Migrations
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	
	_, filename, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(filename), "schema.sql")
	schemaBytes, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, string(schemaBytes))
	require.NoError(t, err, "Failed to execute schema.sql")
	pool.Close()

	// Initialize the actual application store
	s, err := store.NewStore(ctx, connStr)
	require.NoError(t, err)
	defer s.Close()

	// Assert WAL Writing
	doc := store.Document{
		ID:           "test-mutation-1",
		DocumentType: "test-event",
		Payload:      json.RawMessage(`{"key":"value"}`),
		UpdatedAt:    time.Now().UnixNano(),
		IsDeleted:    false,
	}

	err = s.WriteMutation(ctx, doc)
	assert.NoError(t, err, "Expected WriteMutation to succeed")

	// Assert Deterministic WAL Reading
	docs, err := s.ReadMutations(ctx, 0, 10)
	assert.NoError(t, err, "Expected ReadMutations to succeed")
	assert.Len(t, docs, 1, "Expected exactly 1 document in the WAL")
	
	assert.Equal(t, "test-mutation-1", docs[0].ID)
	assert.Equal(t, "test-event", docs[0].DocumentType)
	assert.Equal(t, json.RawMessage(`{"key":"value"}`), docs[0].Payload)
	assert.False(t, docs[0].IsDeleted)
}
