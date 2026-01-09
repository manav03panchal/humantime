// Package integration provides integration tests for Humantime tags feature.
// These tests verify tag creation, filtering, and persistence.
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
// Tags Feature Tests
// =============================================================================

func TestTagsFeature_CreateBlockWithTags(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("creates block with single tag", func(t *testing.T) {
		block := model.NewBlock(config.UserKey, "tagproject", "", "", time.Now())
		block.Tags = []string{"billable"}

		err := blockRepo.Create(block)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, []string{"billable"}, retrieved.Tags)
	})

	t.Run("creates block with multiple tags", func(t *testing.T) {
		block := model.NewBlock(config.UserKey, "multitagproject", "", "", time.Now())
		block.Tags = []string{"billable", "urgent", "meeting"}

		err := blockRepo.Create(block)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, []string{"billable", "urgent", "meeting"}, retrieved.Tags)
	})

	t.Run("creates block without tags", func(t *testing.T) {
		block := model.NewBlock(config.UserKey, "notagproject", "", "", time.Now())

		err := blockRepo.Create(block)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)
		assert.Nil(t, retrieved.Tags)
	})
}

func TestTagsFeature_HasTag(t *testing.T) {
	t.Run("HasTag returns true for existing tag", func(t *testing.T) {
		block := &model.Block{Tags: []string{"billable", "urgent"}}
		assert.True(t, block.HasTag("billable"))
		assert.True(t, block.HasTag("urgent"))
	})

	t.Run("HasTag returns false for non-existing tag", func(t *testing.T) {
		block := &model.Block{Tags: []string{"billable"}}
		assert.False(t, block.HasTag("nonexistent"))
	})

	t.Run("HasTag is case-insensitive", func(t *testing.T) {
		block := &model.Block{Tags: []string{"Billable"}}
		assert.True(t, block.HasTag("billable"))
		assert.True(t, block.HasTag("BILLABLE"))
		assert.True(t, block.HasTag("Billable"))
	})

	t.Run("HasTag returns false for empty tags", func(t *testing.T) {
		block := &model.Block{Tags: nil}
		assert.False(t, block.HasTag("any"))

		block2 := &model.Block{Tags: []string{}}
		assert.False(t, block2.HasTag("any"))
	})
}

func TestTagsFeature_FilterByTag(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	// Create test blocks with different tags
	t.Run("setup blocks", func(t *testing.T) {
		// Block with billable tag
		block1 := model.NewBlock(config.UserKey, "client1", "", "", time.Now())
		block1.Tags = []string{"billable"}
		block1.TimestampEnd = time.Now().Add(1 * time.Hour)
		require.NoError(t, blockRepo.Create(block1))

		// Block with billable and urgent tags
		block2 := model.NewBlock(config.UserKey, "client2", "", "", time.Now())
		block2.Tags = []string{"billable", "urgent"}
		block2.TimestampEnd = time.Now().Add(2 * time.Hour)
		require.NoError(t, blockRepo.Create(block2))

		// Block with urgent tag only
		block3 := model.NewBlock(config.UserKey, "internal", "", "", time.Now())
		block3.Tags = []string{"urgent"}
		block3.TimestampEnd = time.Now().Add(30 * time.Minute)
		require.NoError(t, blockRepo.Create(block3))

		// Block without any tags
		block4 := model.NewBlock(config.UserKey, "misc", "", "", time.Now())
		block4.TimestampEnd = time.Now().Add(15 * time.Minute)
		require.NoError(t, blockRepo.Create(block4))
	})

	t.Run("filters blocks by billable tag", func(t *testing.T) {
		filter := storage.BlockFilter{Tag: "billable"}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		assert.Len(t, blocks, 2)
		for _, b := range blocks {
			assert.True(t, b.HasTag("billable"), "block %s should have billable tag", b.ProjectSID)
		}
	})

	t.Run("filters blocks by urgent tag", func(t *testing.T) {
		filter := storage.BlockFilter{Tag: "urgent"}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		assert.Len(t, blocks, 2)
		for _, b := range blocks {
			assert.True(t, b.HasTag("urgent"), "block %s should have urgent tag", b.ProjectSID)
		}
	})

	t.Run("filter with non-existing tag returns empty", func(t *testing.T) {
		filter := storage.BlockFilter{Tag: "nonexistent"}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		assert.Len(t, blocks, 0)
	})

	t.Run("combines tag filter with project filter", func(t *testing.T) {
		filter := storage.BlockFilter{
			Tag:        "billable",
			ProjectSID: "client1",
		}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		assert.Len(t, blocks, 1)
		assert.Equal(t, "client1", blocks[0].ProjectSID)
	})
}

func TestTagsFeature_TagsPersistence(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("tags persist after update", func(t *testing.T) {
		// Create block with tags
		block := model.NewBlock(config.UserKey, "persisttest", "", "", time.Now())
		block.Tags = []string{"original"}
		require.NoError(t, blockRepo.Create(block))

		// Update with new tags
		block.Tags = []string{"updated", "new"}
		require.NoError(t, blockRepo.Update(block))

		// Retrieve and verify
		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, []string{"updated", "new"}, retrieved.Tags)
	})

	t.Run("tags survive stop operation", func(t *testing.T) {
		// Create active block with tags
		block := model.NewBlock(config.UserKey, "stoptest", "", "", time.Now())
		block.Tags = []string{"billable", "important"}
		require.NoError(t, blockRepo.Create(block))

		// Simulate stop by setting end time
		block.TimestampEnd = time.Now()
		require.NoError(t, blockRepo.Update(block))

		// Retrieve and verify tags preserved
		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, []string{"billable", "important"}, retrieved.Tags)
		assert.False(t, retrieved.IsActive())
	})
}
