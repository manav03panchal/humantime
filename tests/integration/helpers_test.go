// Package integration provides integration tests for Humantime components.
//
// This file contains shared test helpers used across integration tests.
package integration

import (
	"testing"

	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a new in-memory database for testing.
func setupTestDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err, "failed to open in-memory database")
	t.Cleanup(func() {
		err := db.Close()
		assert.NoError(t, err, "failed to close database")
	})
	return db
}
