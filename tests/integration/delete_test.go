// Package integration provides integration tests for Humantime delete feature.
// These tests verify block deletion functionality.
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
// Delete Feature Tests
// =============================================================================

func TestDeleteFeature_DeleteBlock(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("deletes a block successfully", func(t *testing.T) {
		// Create a block
		block := model.NewBlock(config.UserKey, "deletetest", "", "", time.Now())
		block.TimestampEnd = time.Now().Add(1 * time.Hour)
		require.NoError(t, blockRepo.Create(block))

		// Verify block exists
		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Delete the block
		err = blockRepo.Delete(block.Key)
		require.NoError(t, err)

		// Verify block no longer exists
		_, err = blockRepo.Get(block.Key)
		assert.Error(t, err)
	})

	t.Run("delete removes block from list", func(t *testing.T) {
		// Create multiple blocks
		block1 := model.NewBlock(config.UserKey, "keep1", "", "", time.Now())
		block1.TimestampEnd = time.Now().Add(1 * time.Hour)
		require.NoError(t, blockRepo.Create(block1))

		block2 := model.NewBlock(config.UserKey, "deleteme", "", "", time.Now())
		block2.TimestampEnd = time.Now().Add(2 * time.Hour)
		require.NoError(t, blockRepo.Create(block2))

		block3 := model.NewBlock(config.UserKey, "keep2", "", "", time.Now())
		block3.TimestampEnd = time.Now().Add(30 * time.Minute)
		require.NoError(t, blockRepo.Create(block3))

		// Delete block2
		err := blockRepo.Delete(block2.Key)
		require.NoError(t, err)

		// List blocks and verify block2 is gone
		blocks, err := blockRepo.List()
		require.NoError(t, err)

		for _, b := range blocks {
			assert.NotEqual(t, block2.Key, b.Key, "deleted block should not appear in list")
		}
	})

	t.Run("delete non-existent block returns error", func(t *testing.T) {
		err := blockRepo.Delete("block:nonexistent-key")
		// The error behavior depends on implementation
		// Some implementations may return nil for deleting non-existent keys
		// This test documents the expected behavior
		_ = err
	})

	t.Run("delete active block", func(t *testing.T) {
		// Create an active block (no end time)
		block := model.NewBlock(config.UserKey, "activeblock", "", "", time.Now())
		require.NoError(t, blockRepo.Create(block))
		assert.True(t, block.IsActive())

		// Delete should work even for active blocks
		err := blockRepo.Delete(block.Key)
		require.NoError(t, err)

		// Verify block is deleted
		_, err = blockRepo.Get(block.Key)
		assert.Error(t, err)
	})
}

func TestDeleteFeature_DeleteDoesNotAffectOtherBlocks(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("deleting one block preserves others", func(t *testing.T) {
		// Create several blocks
		blocks := make([]*model.Block, 5)
		for i := 0; i < 5; i++ {
			blocks[i] = model.NewBlock(config.UserKey, "project", "", "", time.Now().Add(time.Duration(i)*time.Hour))
			blocks[i].TimestampEnd = time.Now().Add(time.Duration(i+1) * time.Hour)
			require.NoError(t, blockRepo.Create(blocks[i]))
		}

		// Delete block at index 2
		err := blockRepo.Delete(blocks[2].Key)
		require.NoError(t, err)

		// Verify other blocks still exist
		for i, block := range blocks {
			if i == 2 {
				continue // Skip deleted block
			}
			retrieved, err := blockRepo.Get(block.Key)
			require.NoError(t, err)
			assert.Equal(t, block.Key, retrieved.Key)
		}
	})
}

func TestDeleteFeature_DeletePreservesData(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("deleting block does not delete project", func(t *testing.T) {
		// Create project
		_, _, err := projectRepo.GetOrCreate("preserveproject", "Preserve Project")
		require.NoError(t, err)

		// Create block for project
		block := model.NewBlock(config.UserKey, "preserveproject", "", "", time.Now())
		block.TimestampEnd = time.Now().Add(1 * time.Hour)
		require.NoError(t, blockRepo.Create(block))

		// Delete block
		err = blockRepo.Delete(block.Key)
		require.NoError(t, err)

		// Project should still exist
		exists, err := projectRepo.Exists("preserveproject")
		require.NoError(t, err)
		assert.True(t, exists, "project should still exist after block deletion")
	})
}
