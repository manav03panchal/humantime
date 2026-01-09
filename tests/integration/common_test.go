package integration

import (
	"testing"

	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates an in-memory database for testing.
func setupTestDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err, "failed to open in-memory database")
	t.Cleanup(func() {
		db.Close()
	})
	return db
}
