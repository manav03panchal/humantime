// Package integration provides integration tests for Humantime undo feature.
// These tests verify undo functionality for start, stop, and delete actions.
package integration

import (
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Undo Feature Tests
// =============================================================================

func TestUndoFeature_UndoRepo(t *testing.T) {
	db := setupTestDB(t)
	undoRepo := storage.NewUndoRepo(db)

	t.Run("returns nil when no undo state exists", func(t *testing.T) {
		state, err := undoRepo.Get()
		require.NoError(t, err)
		assert.Nil(t, state)
	})

	t.Run("saves and retrieves undo state for start", func(t *testing.T) {
		err := undoRepo.SaveUndoStart("block:test-key-123")
		require.NoError(t, err)

		state, err := undoRepo.Get()
		require.NoError(t, err)
		require.NotNil(t, state)

		assert.Equal(t, model.UndoActionStart, state.Action)
		assert.Equal(t, "block:test-key-123", state.BlockKey)
		assert.Nil(t, state.BlockSnapshot)
	})

	t.Run("saves and retrieves undo state for stop", func(t *testing.T) {
		block := &model.Block{
			Key:            "block:stop-test-123",
			OwnerKey:       "user:test",
			ProjectSID:     "testproject",
			TaskSID:        "testtask",
			Note:           "test note",
			TimestampStart: time.Now().Add(-1 * time.Hour),
			TimestampEnd:   time.Now(),
		}

		err := undoRepo.SaveUndoStop(block)
		require.NoError(t, err)

		state, err := undoRepo.Get()
		require.NoError(t, err)
		require.NotNil(t, state)

		assert.Equal(t, model.UndoActionStop, state.Action)
		assert.Equal(t, "block:stop-test-123", state.BlockKey)
		require.NotNil(t, state.BlockSnapshot)
		assert.Equal(t, "testproject", state.BlockSnapshot.ProjectSID)
		assert.Equal(t, "testtask", state.BlockSnapshot.TaskSID)
	})

	t.Run("saves and retrieves undo state for delete", func(t *testing.T) {
		block := &model.Block{
			Key:            "block:delete-test-123",
			OwnerKey:       "user:test",
			ProjectSID:     "deleteproject",
			TaskSID:        "",
			Note:           "deleted note",
			Tags:           []string{"billable", "important"},
			TimestampStart: time.Now().Add(-2 * time.Hour),
			TimestampEnd:   time.Now().Add(-1 * time.Hour),
		}

		err := undoRepo.SaveUndoDelete(block)
		require.NoError(t, err)

		state, err := undoRepo.Get()
		require.NoError(t, err)
		require.NotNil(t, state)

		assert.Equal(t, model.UndoActionDelete, state.Action)
		assert.Equal(t, "block:delete-test-123", state.BlockKey)
		require.NotNil(t, state.BlockSnapshot)
		assert.Equal(t, "deleteproject", state.BlockSnapshot.ProjectSID)
		assert.Equal(t, "deleted note", state.BlockSnapshot.Note)
		assert.Equal(t, []string{"billable", "important"}, state.BlockSnapshot.Tags)
	})

	t.Run("clears undo state", func(t *testing.T) {
		err := undoRepo.SaveUndoStart("block:clear-test")
		require.NoError(t, err)

		state, err := undoRepo.Get()
		require.NoError(t, err)
		require.NotNil(t, state)

		err = undoRepo.Clear()
		require.NoError(t, err)

		state, err = undoRepo.Get()
		require.NoError(t, err)
		assert.Nil(t, state)
	})

	t.Run("new save overwrites previous", func(t *testing.T) {
		err := undoRepo.SaveUndoStart("block:first")
		require.NoError(t, err)

		err = undoRepo.SaveUndoStart("block:second")
		require.NoError(t, err)

		state, err := undoRepo.Get()
		require.NoError(t, err)
		require.NotNil(t, state)

		assert.Equal(t, "block:second", state.BlockKey)
	})
}

func TestUndoFeature_UndoStartIntegration(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	undoRepo := storage.NewUndoRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("can undo start by deleting block", func(t *testing.T) {
		// Create a block (simulating start)
		block := model.NewBlock(config.UserKey, "undostartproject", "", "", time.Now())
		require.NoError(t, blockRepo.Create(block))

		// Save undo state
		require.NoError(t, undoRepo.SaveUndoStart(block.Key))

		// Verify block exists
		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Simulate undo by deleting
		state, err := undoRepo.Get()
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Equal(t, model.UndoActionStart, state.Action)

		err = blockRepo.Delete(state.BlockKey)
		require.NoError(t, err)

		// Verify block is deleted
		_, err = blockRepo.Get(block.Key)
		assert.Error(t, err)
	})
}

func TestUndoFeature_UndoStopIntegration(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	undoRepo := storage.NewUndoRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("can undo stop by reopening block", func(t *testing.T) {
		// Create and stop a block
		block := model.NewBlock(config.UserKey, "undostopproject", "task1", "", time.Now().Add(-1*time.Hour))
		require.NoError(t, blockRepo.Create(block))

		// Simulate stop
		block.TimestampEnd = time.Now()
		require.NoError(t, blockRepo.Update(block))

		// Save undo state
		require.NoError(t, undoRepo.SaveUndoStop(block))

		// Simulate undo by reopening
		state, err := undoRepo.Get()
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Equal(t, model.UndoActionStop, state.Action)

		// Get block and clear end time
		retrieved, err := blockRepo.Get(state.BlockKey)
		require.NoError(t, err)

		retrieved.TimestampEnd = time.Time{}
		require.NoError(t, blockRepo.Update(retrieved))
		require.NoError(t, activeBlockRepo.SetActive(retrieved.Key))

		// Verify block is active again
		activeBlock, err := activeBlockRepo.GetActiveBlock(blockRepo)
		require.NoError(t, err)
		require.NotNil(t, activeBlock)
		assert.True(t, activeBlock.IsActive())
		assert.Equal(t, "undostopproject", activeBlock.ProjectSID)
	})
}

func TestUndoFeature_UndoDeleteIntegration(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	undoRepo := storage.NewUndoRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("can undo delete by restoring from snapshot", func(t *testing.T) {
		// Create a block
		block := model.NewBlock(config.UserKey, "undodeleteproject", "", "important work", time.Now().Add(-2*time.Hour))
		block.TimestampEnd = time.Now().Add(-1 * time.Hour)
		block.Tags = []string{"billable"}
		require.NoError(t, blockRepo.Create(block))

		// Save undo state before delete
		require.NoError(t, undoRepo.SaveUndoDelete(block))

		// Delete the block
		require.NoError(t, blockRepo.Delete(block.Key))

		// Verify block is deleted
		_, err = blockRepo.Get(block.Key)
		assert.Error(t, err)

		// Simulate undo by restoring from snapshot
		state, err := undoRepo.Get()
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Equal(t, model.UndoActionDelete, state.Action)
		require.NotNil(t, state.BlockSnapshot)

		// Restore the block
		restoredBlock := state.BlockSnapshot
		err = db.Set(restoredBlock)
		require.NoError(t, err)

		// Verify block is restored with all data
		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, "undodeleteproject", retrieved.ProjectSID)
		assert.Equal(t, "important work", retrieved.Note)
		assert.Equal(t, []string{"billable"}, retrieved.Tags)
		assert.False(t, retrieved.TimestampEnd.IsZero())
	})
}
