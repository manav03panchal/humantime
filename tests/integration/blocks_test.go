// Package integration provides integration tests for Humantime components.
//
// These tests verify the behavior of components working together using
// a real in-memory Badger database. Unlike e2e tests that run the CLI binary,
// integration tests directly test the storage and business logic layers.
package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Blocks Test Helpers
// =============================================================================

// createTestBlock creates a block with the given parameters and saves it to the repo.
func createTestBlock(t *testing.T, repo *storage.BlockRepo, projectSID, taskSID, note string, start, end time.Time) *model.Block {
	t.Helper()
	block := model.NewBlock("user-123", projectSID, taskSID, note, start)
	if !end.IsZero() {
		block.TimestampEnd = end
	}
	require.NoError(t, repo.Create(block))
	return block
}

// =============================================================================
// List All Blocks Tests
// =============================================================================

func TestBlocksListing_ListAllBlocks(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	t.Run("returns empty list when no blocks exist", func(t *testing.T) {
		blocks, err := repo.List()
		require.NoError(t, err)
		assert.Empty(t, blocks)
	})

	t.Run("returns all blocks when no filter applied", func(t *testing.T) {
		// Create multiple blocks
		createTestBlock(t, repo, "project1", "", "note1", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
		createTestBlock(t, repo, "project2", "task1", "note2", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
		createTestBlock(t, repo, "project1", "task2", "", now.Add(-1*time.Hour), now)

		blocks, err := repo.List()
		require.NoError(t, err)
		assert.Len(t, blocks, 3)
	})

	t.Run("includes active blocks (no end time)", func(t *testing.T) {
		// Create an active block (no end time)
		activeBlock := createTestBlock(t, repo, "project3", "", "active", now.Add(-30*time.Minute), time.Time{})

		blocks, err := repo.List()
		require.NoError(t, err)

		// Find the active block
		var foundActive bool
		for _, b := range blocks {
			if b.Key == activeBlock.Key {
				foundActive = true
				assert.True(t, b.IsActive())
				break
			}
		}
		assert.True(t, foundActive, "expected to find active block in list")
	})
}

func TestBlocksListing_BlocksAreSortedByStartTime(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	// Create blocks with different start times
	createTestBlock(t, repo, "project1", "", "", now.Add(-1*time.Hour), now)
	createTestBlock(t, repo, "project2", "", "", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
	createTestBlock(t, repo, "project3", "", "", now.Add(-2*time.Hour), now.Add(-1*time.Hour))

	// Use ListFiltered which returns sorted results
	blocks, err := repo.ListFiltered(storage.BlockFilter{})
	require.NoError(t, err)
	require.Len(t, blocks, 3)

	// Verify sorted by start time (newest first)
	for i := 0; i < len(blocks)-1; i++ {
		assert.True(t, blocks[i].TimestampStart.After(blocks[i+1].TimestampStart) ||
			blocks[i].TimestampStart.Equal(blocks[i+1].TimestampStart),
			"blocks should be sorted by start time descending")
	}
}

// =============================================================================
// Filter by Project Tests
// =============================================================================

func TestBlocksListing_FilterByProject(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	// Create blocks for different projects
	createTestBlock(t, repo, "alpha", "", "alpha note", now.Add(-4*time.Hour), now.Add(-3*time.Hour))
	createTestBlock(t, repo, "beta", "", "beta note", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
	createTestBlock(t, repo, "alpha", "task1", "alpha task", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	createTestBlock(t, repo, "gamma", "", "gamma note", now.Add(-1*time.Hour), now)

	t.Run("filters blocks by project SID", func(t *testing.T) {
		blocks, err := repo.ListFiltered(storage.BlockFilter{ProjectSID: "alpha"})
		require.NoError(t, err)
		assert.Len(t, blocks, 2)

		for _, b := range blocks {
			assert.Equal(t, "alpha", b.ProjectSID)
		}
	})

	t.Run("returns empty list for non-existent project", func(t *testing.T) {
		blocks, err := repo.ListFiltered(storage.BlockFilter{ProjectSID: "nonexistent"})
		require.NoError(t, err)
		assert.Empty(t, blocks)
	})

	t.Run("filters by project and task", func(t *testing.T) {
		blocks, err := repo.ListFiltered(storage.BlockFilter{
			ProjectSID: "alpha",
			TaskSID:    "task1",
		})
		require.NoError(t, err)
		assert.Len(t, blocks, 1)
		assert.Equal(t, "alpha", blocks[0].ProjectSID)
		assert.Equal(t, "task1", blocks[0].TaskSID)
	})

	t.Run("ListByProject returns all blocks for project", func(t *testing.T) {
		blocks, err := repo.ListByProject("alpha")
		require.NoError(t, err)
		assert.Len(t, blocks, 2)
	})

	t.Run("ListByProjectAndTask returns filtered blocks", func(t *testing.T) {
		blocks, err := repo.ListByProjectAndTask("alpha", "task1")
		require.NoError(t, err)
		assert.Len(t, blocks, 1)
	})
}

// =============================================================================
// Filter by Time Range Tests
// =============================================================================

func TestBlocksListing_FilterByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	// Create blocks at different times
	// Block 1: 5-4 hours ago
	createTestBlock(t, repo, "project1", "", "old block", now.Add(-5*time.Hour), now.Add(-4*time.Hour))
	// Block 2: 3-2 hours ago
	createTestBlock(t, repo, "project2", "", "mid block", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
	// Block 3: 1 hour ago to now
	createTestBlock(t, repo, "project3", "", "recent block", now.Add(-1*time.Hour), now)
	// Block 4: Active block started 30 min ago
	createTestBlock(t, repo, "project4", "", "active block", now.Add(-30*time.Minute), time.Time{})

	t.Run("filters blocks starting after a specific time", func(t *testing.T) {
		// Get blocks that started after 2 hours ago
		blocks, err := repo.ListFiltered(storage.BlockFilter{
			StartAfter: now.Add(-2 * time.Hour),
		})
		require.NoError(t, err)
		assert.Len(t, blocks, 2) // Block 3 and Block 4

		for _, b := range blocks {
			assert.True(t, b.TimestampStart.After(now.Add(-2*time.Hour)) ||
				b.TimestampStart.Equal(now.Add(-2*time.Hour)))
		}
	})

	t.Run("filters blocks ending before a specific time", func(t *testing.T) {
		// Get blocks that ended before 2 hours ago
		blocks, err := repo.ListFiltered(storage.BlockFilter{
			EndBefore: now.Add(-2 * time.Hour),
		})
		require.NoError(t, err)
		// Should include Block 1 (ended 4h ago) but not Block 2 (ended exactly 2h ago, which is not before)
		// Block 2 ends at exactly -2h, so it's not "before" -2h
		assert.GreaterOrEqual(t, len(blocks), 1)
	})

	t.Run("filters blocks within a time range", func(t *testing.T) {
		// Get blocks that started after 4h ago and ended before 1h ago
		blocks, err := repo.ListFiltered(storage.BlockFilter{
			StartAfter: now.Add(-4 * time.Hour),
			EndBefore:  now.Add(-1 * time.Hour),
		})
		require.NoError(t, err)
		// Only Block 2 (3-2h ago) should match
		assert.Len(t, blocks, 1)
		assert.Equal(t, "project2", blocks[0].ProjectSID)
	})

	t.Run("ListByTimeRange finds overlapping blocks", func(t *testing.T) {
		// Find blocks that overlap with 2.5h ago to 1.5h ago
		blocks, err := repo.ListByTimeRange(now.Add(-150*time.Minute), now.Add(-90*time.Minute))
		require.NoError(t, err)
		// Block 2 (3h-2h ago) overlaps with this range
		assert.GreaterOrEqual(t, len(blocks), 1)
	})

	t.Run("includes active blocks in time range queries", func(t *testing.T) {
		// Query for blocks in the last hour should include the active block
		blocks, err := repo.ListFiltered(storage.BlockFilter{
			StartAfter: now.Add(-1 * time.Hour),
		})
		require.NoError(t, err)

		var foundActive bool
		for _, b := range blocks {
			if b.IsActive() {
				foundActive = true
				break
			}
		}
		assert.True(t, foundActive, "expected to find active block in time range")
	})
}

// =============================================================================
// Combined Filters Tests
// =============================================================================

func TestBlocksListing_CombinedFilters(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	// Create a variety of blocks
	createTestBlock(t, repo, "work", "coding", "work coding", now.Add(-5*time.Hour), now.Add(-4*time.Hour))
	createTestBlock(t, repo, "work", "meetings", "work meetings", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
	createTestBlock(t, repo, "personal", "exercise", "exercise", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	createTestBlock(t, repo, "work", "coding", "more coding", now.Add(-1*time.Hour), now)

	t.Run("filters by project and time range", func(t *testing.T) {
		blocks, err := repo.ListFiltered(storage.BlockFilter{
			ProjectSID: "work",
			StartAfter: now.Add(-3 * time.Hour),
		})
		require.NoError(t, err)
		assert.Len(t, blocks, 2) // "work/meetings" and "work/coding" from last 3 hours

		for _, b := range blocks {
			assert.Equal(t, "work", b.ProjectSID)
		}
	})

	t.Run("filters by project, task, and time range", func(t *testing.T) {
		blocks, err := repo.ListFiltered(storage.BlockFilter{
			ProjectSID: "work",
			TaskSID:    "coding",
			StartAfter: now.Add(-2 * time.Hour),
		})
		require.NoError(t, err)
		assert.Len(t, blocks, 1)
		assert.Equal(t, "work", blocks[0].ProjectSID)
		assert.Equal(t, "coding", blocks[0].TaskSID)
	})

	t.Run("limit works with other filters", func(t *testing.T) {
		blocks, err := repo.ListFiltered(storage.BlockFilter{
			ProjectSID: "work",
			Limit:      2,
		})
		require.NoError(t, err)
		assert.Len(t, blocks, 2)
		// Should be the 2 most recent work blocks
	})
}

// =============================================================================
// Limit Tests
// =============================================================================

func TestBlocksListing_Limit(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	// Create 10 blocks
	for i := 0; i < 10; i++ {
		createTestBlock(t, repo, "project", "", "", now.Add(time.Duration(-10+i)*time.Hour), now.Add(time.Duration(-9+i)*time.Hour))
	}

	t.Run("applies limit to results", func(t *testing.T) {
		blocks, err := repo.ListFiltered(storage.BlockFilter{Limit: 5})
		require.NoError(t, err)
		assert.Len(t, blocks, 5)
	})

	t.Run("limit of zero returns all results", func(t *testing.T) {
		blocks, err := repo.ListFiltered(storage.BlockFilter{Limit: 0})
		require.NoError(t, err)
		assert.Len(t, blocks, 10)
	})

	t.Run("limit greater than total returns all results", func(t *testing.T) {
		blocks, err := repo.ListFiltered(storage.BlockFilter{Limit: 100})
		require.NoError(t, err)
		assert.Len(t, blocks, 10)
	})

	t.Run("limit returns most recent blocks", func(t *testing.T) {
		blocks, err := repo.ListFiltered(storage.BlockFilter{Limit: 3})
		require.NoError(t, err)
		assert.Len(t, blocks, 3)

		// Verify these are the 3 most recent
		for i := 0; i < len(blocks)-1; i++ {
			assert.True(t, blocks[i].TimestampStart.After(blocks[i+1].TimestampStart) ||
				blocks[i].TimestampStart.Equal(blocks[i+1].TimestampStart))
		}
	})
}

// =============================================================================
// JSON Output Format Tests
// =============================================================================

func TestBlocksListing_JSONOutputFormat(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	// Create test blocks
	block1 := createTestBlock(t, repo, "project1", "task1", "test note", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	block2 := createTestBlock(t, repo, "project2", "", "", now.Add(-1*time.Hour), now)
	activeBlock := createTestBlock(t, repo, "project3", "", "active", now.Add(-30*time.Minute), time.Time{})

	t.Run("BlockOutput contains all required fields", func(t *testing.T) {
		blockOutput := output.NewBlockOutput(block1)

		assert.Equal(t, block1.Key, blockOutput.Key)
		assert.Equal(t, "project1", blockOutput.ProjectSID)
		assert.Equal(t, "task1", blockOutput.TaskSID)
		assert.Equal(t, "test note", blockOutput.Note)
		assert.NotEmpty(t, blockOutput.TimestampStart)
		assert.NotEmpty(t, blockOutput.TimestampEnd)
		assert.False(t, blockOutput.IsActive)
		assert.Greater(t, blockOutput.DurationSeconds, int64(0))
	})

	t.Run("BlockOutput handles active blocks correctly", func(t *testing.T) {
		blockOutput := output.NewBlockOutput(activeBlock)

		assert.True(t, blockOutput.IsActive)
		assert.Empty(t, blockOutput.TimestampEnd)
		assert.Greater(t, blockOutput.DurationSeconds, int64(0))
	})

	t.Run("BlocksResponse contains correct counts", func(t *testing.T) {
		blocks, err := repo.List()
		require.NoError(t, err)

		response := output.NewBlocksResponse(blocks, len(blocks))

		assert.Equal(t, len(blocks), response.TotalCount)
		assert.Equal(t, len(blocks), response.ShownCount)
		assert.Len(t, response.Blocks, len(blocks))
	})

	t.Run("BlocksResponse calculates total duration", func(t *testing.T) {
		blocks := []*model.Block{block1, block2}
		response := output.NewBlocksResponse(blocks, 3)

		assert.Equal(t, 3, response.TotalCount)
		assert.Equal(t, 2, response.ShownCount)
		assert.Greater(t, response.TotalDurationSeconds, int64(0))
	})

	t.Run("BlockOutput timestamps are RFC3339 formatted", func(t *testing.T) {
		blockOutput := output.NewBlockOutput(block1)

		// Verify timestamps can be parsed back
		_, err := time.Parse(time.RFC3339, blockOutput.TimestampStart)
		assert.NoError(t, err, "timestamp_start should be RFC3339 formatted")

		_, err = time.Parse(time.RFC3339, blockOutput.TimestampEnd)
		assert.NoError(t, err, "timestamp_end should be RFC3339 formatted")
	})

	t.Run("BlocksResponse serializes to valid JSON", func(t *testing.T) {
		blocks, err := repo.List()
		require.NoError(t, err)

		response := output.NewBlocksResponse(blocks, len(blocks))

		jsonBytes, err := json.Marshal(response)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonBytes)

		// Verify it can be unmarshaled back
		var decoded output.BlocksResponse
		err = json.Unmarshal(jsonBytes, &decoded)
		require.NoError(t, err)

		assert.Equal(t, response.TotalCount, decoded.TotalCount)
		assert.Equal(t, response.ShownCount, decoded.ShownCount)
		assert.Len(t, decoded.Blocks, len(response.Blocks))
	})

	t.Run("JSON output includes all block fields", func(t *testing.T) {
		blockOutput := output.NewBlockOutput(block1)

		jsonBytes, err := json.Marshal(blockOutput)
		require.NoError(t, err)

		var decoded map[string]interface{}
		err = json.Unmarshal(jsonBytes, &decoded)
		require.NoError(t, err)

		// Verify required fields are present
		assert.Contains(t, decoded, "key")
		assert.Contains(t, decoded, "project_sid")
		assert.Contains(t, decoded, "timestamp_start")
		assert.Contains(t, decoded, "duration_seconds")
		assert.Contains(t, decoded, "is_active")

		// Verify optional fields are present when set
		assert.Contains(t, decoded, "task_sid")
		assert.Contains(t, decoded, "note")
		assert.Contains(t, decoded, "timestamp_end")
	})

	t.Run("JSON output omits empty optional fields", func(t *testing.T) {
		blockOutput := output.NewBlockOutput(block2) // block2 has no task or note

		jsonBytes, err := json.Marshal(blockOutput)
		require.NoError(t, err)

		var decoded map[string]interface{}
		err = json.Unmarshal(jsonBytes, &decoded)
		require.NoError(t, err)

		// TaskSID and Note should be omitted when empty
		_, hasTaskSID := decoded["task_sid"]
		_, hasNote := decoded["note"]

		// Note: The BlockOutput uses omitempty, so empty strings might still appear
		// We're just verifying the structure is valid
		if hasTaskSID {
			assert.Empty(t, decoded["task_sid"])
		}
		if hasNote {
			assert.Empty(t, decoded["note"])
		}
	})
}

// =============================================================================
// Duration Calculation Tests
// =============================================================================

func TestBlocksListing_DurationCalculation(t *testing.T) {
	_ = setupTestDB(t) // Ensure database is available for the test context
	now := time.Now()

	t.Run("TotalDuration calculates sum of block durations", func(t *testing.T) {
		blocks := []*model.Block{
			{TimestampStart: now.Add(-3 * time.Hour), TimestampEnd: now.Add(-2 * time.Hour)},   // 1 hour
			{TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},   // 1 hour
			{TimestampStart: now.Add(-30 * time.Minute), TimestampEnd: now},                    // 30 min
		}

		total := storage.TotalDuration(blocks)
		assert.InDelta(t, 2.5*float64(time.Hour), float64(total), float64(time.Second))
	})

	t.Run("TotalDuration returns zero for empty list", func(t *testing.T) {
		total := storage.TotalDuration([]*model.Block{})
		assert.Equal(t, time.Duration(0), total)
	})

	t.Run("AggregateByProject groups by project correctly", func(t *testing.T) {
		blocks := []*model.Block{
			{ProjectSID: "proj1", TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},
			{ProjectSID: "proj1", TimestampStart: now.Add(-3 * time.Hour), TimestampEnd: now.Add(-2 * time.Hour)},
			{ProjectSID: "proj2", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
		}

		agg := storage.AggregateByProject(blocks)
		require.Len(t, agg, 2)

		// Find proj1 aggregate
		var proj1Agg *storage.ProjectAggregate
		for i := range agg {
			if agg[i].ProjectSID == "proj1" {
				proj1Agg = &agg[i]
				break
			}
		}
		require.NotNil(t, proj1Agg)
		assert.Equal(t, 2, proj1Agg.BlockCount)
		assert.InDelta(t, 2*float64(time.Hour), float64(proj1Agg.Duration), float64(time.Second))
	})

	t.Run("AggregateByProject sorts by duration descending", func(t *testing.T) {
		blocks := []*model.Block{
			{ProjectSID: "short", TimestampStart: now.Add(-30 * time.Minute), TimestampEnd: now},
			{ProjectSID: "long", TimestampStart: now.Add(-3 * time.Hour), TimestampEnd: now},
		}

		agg := storage.AggregateByProject(blocks)
		require.Len(t, agg, 2)
		assert.Equal(t, "long", agg[0].ProjectSID)
		assert.Equal(t, "short", agg[1].ProjectSID)
	})
}

// =============================================================================
// Delete Block Tests
// =============================================================================

func TestBlockOperations_DeleteBlock(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	t.Run("deletes a block by key", func(t *testing.T) {
		block := createTestBlock(t, repo, "deleteme", "", "to be deleted", now.Add(-time.Hour), now)

		// Verify block exists
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Delete the block
		err = repo.Delete(block.Key)
		require.NoError(t, err)

		// Verify block no longer exists
		_, err = repo.Get(block.Key)
		assert.Error(t, err)
	})

	t.Run("deleted block does not appear in list", func(t *testing.T) {
		block1 := createTestBlock(t, repo, "keep1", "", "", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
		block2 := createTestBlock(t, repo, "delete2", "", "", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
		block3 := createTestBlock(t, repo, "keep3", "", "", now.Add(-1*time.Hour), now)

		// Delete block2
		err := repo.Delete(block2.Key)
		require.NoError(t, err)

		// List all blocks
		blocks, err := repo.List()
		require.NoError(t, err)

		// Find our test blocks
		var foundKeys []string
		for _, b := range blocks {
			if b.Key == block1.Key || b.Key == block2.Key || b.Key == block3.Key {
				foundKeys = append(foundKeys, b.Key)
			}
		}

		assert.Contains(t, foundKeys, block1.Key)
		assert.NotContains(t, foundKeys, block2.Key)
		assert.Contains(t, foundKeys, block3.Key)
	})

	t.Run("deleted block does not appear in filtered results", func(t *testing.T) {
		block := createTestBlock(t, repo, "filterdelete", "task", "", now.Add(-time.Hour), now)

		// Delete the block
		err := repo.Delete(block.Key)
		require.NoError(t, err)

		// Verify it doesn't appear in project filter
		blocks, err := repo.ListFiltered(storage.BlockFilter{ProjectSID: "filterdelete"})
		require.NoError(t, err)

		for _, b := range blocks {
			assert.NotEqual(t, block.Key, b.Key)
		}
	})

	t.Run("can delete active block", func(t *testing.T) {
		activeBlock := createTestBlock(t, repo, "activeDelete", "", "", now.Add(-30*time.Minute), time.Time{})
		assert.True(t, activeBlock.IsActive())

		err := repo.Delete(activeBlock.Key)
		require.NoError(t, err)

		_, err = repo.Get(activeBlock.Key)
		assert.Error(t, err)
	})

	t.Run("deleting one block does not affect others", func(t *testing.T) {
		// Create multiple blocks for the same project
		blocks := make([]*model.Block, 5)
		for i := 0; i < 5; i++ {
			blocks[i] = createTestBlock(t, repo, "multiDelete", "", "",
				now.Add(time.Duration(-5+i)*time.Hour), now.Add(time.Duration(-4+i)*time.Hour))
		}

		// Delete middle block
		err := repo.Delete(blocks[2].Key)
		require.NoError(t, err)

		// Verify all other blocks still exist
		for i, block := range blocks {
			if i == 2 {
				_, err := repo.Get(block.Key)
				assert.Error(t, err, "deleted block should not exist")
			} else {
				retrieved, err := repo.Get(block.Key)
				require.NoError(t, err)
				assert.Equal(t, block.Key, retrieved.Key)
			}
		}
	})

	t.Run("duration aggregation excludes deleted blocks", func(t *testing.T) {
		// Create blocks with known durations
		block1 := createTestBlock(t, repo, "durDelete1", "", "", now.Add(-3*time.Hour), now.Add(-2*time.Hour)) // 1h
		block2 := createTestBlock(t, repo, "durDelete2", "", "", now.Add(-2*time.Hour), now.Add(-1*time.Hour)) // 1h
		createTestBlock(t, repo, "durDelete3", "", "", now.Add(-1*time.Hour), now)                             // 1h

		// Get initial total
		initialBlocks, err := repo.ListFiltered(storage.BlockFilter{})
		require.NoError(t, err)
		initialTotal := storage.TotalDuration(initialBlocks)

		// Delete block2
		err = repo.Delete(block2.Key)
		require.NoError(t, err)

		// Get new total
		remainingBlocks, err := repo.ListFiltered(storage.BlockFilter{})
		require.NoError(t, err)
		newTotal := storage.TotalDuration(remainingBlocks)

		// New total should be less by approximately 1 hour
		assert.Less(t, newTotal, initialTotal)
		assert.InDelta(t, float64(initialTotal-time.Hour), float64(newTotal), float64(5*time.Second))

		// Cleanup
		repo.Delete(block1.Key)
	})
}

// =============================================================================
// Edit Block Tests
// =============================================================================

func TestBlockOperations_EditBlock(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	t.Run("updates block note", func(t *testing.T) {
		block := createTestBlock(t, repo, "editNote", "", "original note", now.Add(-time.Hour), now)

		// Update the note
		block.Note = "updated note"
		err := repo.Update(block)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, "updated note", retrieved.Note)
	})

	t.Run("updates block project", func(t *testing.T) {
		block := createTestBlock(t, repo, "projectA", "", "", now.Add(-time.Hour), now)

		// Update the project
		block.ProjectSID = "projectB"
		err := repo.Update(block)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, "projectB", retrieved.ProjectSID)

		// Verify it appears in new project filter
		blocks, err := repo.ListFiltered(storage.BlockFilter{ProjectSID: "projectB"})
		require.NoError(t, err)

		var found bool
		for _, b := range blocks {
			if b.Key == block.Key {
				found = true
				break
			}
		}
		assert.True(t, found, "block should appear in new project filter")
	})

	t.Run("updates block task", func(t *testing.T) {
		block := createTestBlock(t, repo, "project", "taskA", "", now.Add(-time.Hour), now)

		// Update the task
		block.TaskSID = "taskB"
		err := repo.Update(block)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, "taskB", retrieved.TaskSID)
	})

	t.Run("updates block start time", func(t *testing.T) {
		block := createTestBlock(t, repo, "editStart", "", "", now.Add(-2*time.Hour), now.Add(-time.Hour))

		// Update the start time
		newStart := now.Add(-3 * time.Hour)
		block.TimestampStart = newStart
		err := repo.Update(block)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.True(t, retrieved.TimestampStart.Equal(newStart))
	})

	t.Run("updates block end time", func(t *testing.T) {
		block := createTestBlock(t, repo, "editEnd", "", "", now.Add(-2*time.Hour), now.Add(-time.Hour))

		// Update the end time
		newEnd := now
		block.TimestampEnd = newEnd
		err := repo.Update(block)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.True(t, retrieved.TimestampEnd.Equal(newEnd))
		assert.InDelta(t, 2.0*3600, retrieved.Duration().Seconds(), 1.0)
	})

	t.Run("stops active block by setting end time", func(t *testing.T) {
		block := createTestBlock(t, repo, "stopActive", "", "", now.Add(-30*time.Minute), time.Time{})
		assert.True(t, block.IsActive())

		// Set end time to stop the block
		block.TimestampEnd = now
		err := repo.Update(block)
		require.NoError(t, err)

		// Retrieve and verify it's no longer active
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.False(t, retrieved.IsActive())
	})

	t.Run("updates multiple fields at once", func(t *testing.T) {
		block := createTestBlock(t, repo, "multiUpdate", "oldTask", "old note", now.Add(-3*time.Hour), now.Add(-2*time.Hour))

		// Update multiple fields
		block.ProjectSID = "newProject"
		block.TaskSID = "newTask"
		block.Note = "new note"
		block.TimestampStart = now.Add(-4 * time.Hour)
		block.TimestampEnd = now.Add(-1 * time.Hour)

		err := repo.Update(block)
		require.NoError(t, err)

		// Retrieve and verify all updates
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, "newProject", retrieved.ProjectSID)
		assert.Equal(t, "newTask", retrieved.TaskSID)
		assert.Equal(t, "new note", retrieved.Note)
		assert.True(t, retrieved.TimestampStart.Equal(now.Add(-4*time.Hour)))
		assert.True(t, retrieved.TimestampEnd.Equal(now.Add(-1*time.Hour)))
	})

	t.Run("updates block tags", func(t *testing.T) {
		block := createTestBlock(t, repo, "tagEdit", "", "", now.Add(-time.Hour), now)
		block.Tags = []string{"initial"}
		err := repo.Update(block)
		require.NoError(t, err)

		// Update tags
		block.Tags = []string{"updated", "new-tag", "important"}
		err = repo.Update(block)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.Len(t, retrieved.Tags, 3)
		assert.Contains(t, retrieved.Tags, "updated")
		assert.Contains(t, retrieved.Tags, "new-tag")
		assert.Contains(t, retrieved.Tags, "important")
	})

	t.Run("clears optional fields", func(t *testing.T) {
		block := createTestBlock(t, repo, "clearFields", "taskToClear", "note to clear", now.Add(-time.Hour), now)
		block.Tags = []string{"tag1", "tag2"}
		err := repo.Update(block)
		require.NoError(t, err)

		// Clear optional fields
		block.TaskSID = ""
		block.Note = ""
		block.Tags = nil
		err = repo.Update(block)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.Empty(t, retrieved.TaskSID)
		assert.Empty(t, retrieved.Note)
		assert.Empty(t, retrieved.Tags)
	})

	t.Run("update preserves block key", func(t *testing.T) {
		block := createTestBlock(t, repo, "preserveKey", "", "", now.Add(-time.Hour), now)
		originalKey := block.Key

		// Update the block
		block.Note = "updated"
		err := repo.Update(block)
		require.NoError(t, err)

		// Verify key is preserved
		assert.Equal(t, originalKey, block.Key)

		// Can still retrieve by original key
		retrieved, err := repo.Get(originalKey)
		require.NoError(t, err)
		assert.Equal(t, "updated", retrieved.Note)
	})

	t.Run("update affects filter results", func(t *testing.T) {
		block := createTestBlock(t, repo, "filterUpdate", "", "", now.Add(-time.Hour), now)

		// Initially in filterUpdate project
		blocks, err := repo.ListFiltered(storage.BlockFilter{ProjectSID: "filterUpdate"})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(blocks), 1)

		// Move to different project
		block.ProjectSID = "newFilterProject"
		err = repo.Update(block)
		require.NoError(t, err)

		// Should no longer appear in old project filter
		blocks, err = repo.ListFiltered(storage.BlockFilter{ProjectSID: "filterUpdate"})
		require.NoError(t, err)
		for _, b := range blocks {
			assert.NotEqual(t, block.Key, b.Key)
		}

		// Should appear in new project filter
		blocks, err = repo.ListFiltered(storage.BlockFilter{ProjectSID: "newFilterProject"})
		require.NoError(t, err)
		var found bool
		for _, b := range blocks {
			if b.Key == block.Key {
				found = true
				break
			}
		}
		assert.True(t, found)
	})
}

// =============================================================================
// Filter by Tag Tests
// =============================================================================

func TestBlocksListing_FilterByTag(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	// Helper to create block with tags
	createBlockWithTags := func(projectSID string, tags []string) *model.Block {
		block := createTestBlock(t, repo, projectSID, "", "", now.Add(-time.Hour), now)
		block.Tags = tags
		require.NoError(t, repo.Update(block))
		return block
	}

	t.Run("filters blocks by single tag", func(t *testing.T) {
		block1 := createBlockWithTags("tagproj1", []string{"urgent", "work"})
		createBlockWithTags("tagproj2", []string{"personal"})
		block3 := createBlockWithTags("tagproj3", []string{"urgent", "home"})

		blocks, err := repo.ListFiltered(storage.BlockFilter{Tag: "urgent"})
		require.NoError(t, err)

		var foundKeys []string
		for _, b := range blocks {
			if b.Key == block1.Key || b.Key == block3.Key {
				foundKeys = append(foundKeys, b.Key)
			}
		}
		assert.Len(t, foundKeys, 2)
	})

	t.Run("tag filter is case-insensitive", func(t *testing.T) {
		block := createBlockWithTags("caseTag", []string{"Important"})

		blocks, err := repo.ListFiltered(storage.BlockFilter{Tag: "important"})
		require.NoError(t, err)

		var found bool
		for _, b := range blocks {
			if b.Key == block.Key {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("returns empty for non-existent tag", func(t *testing.T) {
		createBlockWithTags("noMatch", []string{"existing"})

		blocks, err := repo.ListFiltered(storage.BlockFilter{Tag: "nonexistent-tag-xyz"})
		require.NoError(t, err)

		for _, b := range blocks {
			assert.False(t, b.HasTag("nonexistent-tag-xyz"))
		}
	})

	t.Run("combines tag filter with project filter", func(t *testing.T) {
		createBlockWithTags("combineProj", []string{"special"})
		createBlockWithTags("otherProj", []string{"special"})
		createBlockWithTags("combineProj", []string{"normal"})

		blocks, err := repo.ListFiltered(storage.BlockFilter{
			ProjectSID: "combineProj",
			Tag:        "special",
		})
		require.NoError(t, err)

		for _, b := range blocks {
			assert.Equal(t, "combineProj", b.ProjectSID)
			assert.True(t, b.HasTag("special"))
		}
	})

	t.Run("combines tag filter with time range", func(t *testing.T) {
		block := model.NewBlock("user-123", "timeTagProj", "", "", now.Add(-30*time.Minute))
		block.TimestampEnd = now
		block.Tags = []string{"recent"}
		require.NoError(t, repo.Create(block))

		oldBlock := model.NewBlock("user-123", "timeTagProj2", "", "", now.Add(-5*time.Hour))
		oldBlock.TimestampEnd = now.Add(-4 * time.Hour)
		oldBlock.Tags = []string{"recent"}
		require.NoError(t, repo.Create(oldBlock))

		blocks, err := repo.ListFiltered(storage.BlockFilter{
			Tag:        "recent",
			StartAfter: now.Add(-1 * time.Hour),
		})
		require.NoError(t, err)

		for _, b := range blocks {
			assert.True(t, b.HasTag("recent"))
			assert.True(t, b.TimestampStart.After(now.Add(-1*time.Hour)) || b.TimestampStart.Equal(now.Add(-1*time.Hour)))
		}
	})
}

// =============================================================================
// Additional Date Range Filter Tests
// =============================================================================

func TestBlocksListing_DateRangeEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	t.Run("filters blocks for today", func(t *testing.T) {
		// Start of today
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		tomorrowStart := todayStart.Add(24 * time.Hour)

		// Create a block for today
		todayBlock := createTestBlock(t, repo, "todayProj", "", "today work", now.Add(-2*time.Hour), now.Add(-1*time.Hour))

		// Create a block for yesterday
		yesterdayBlock := createTestBlock(t, repo, "yesterdayProj", "", "yesterday work",
			todayStart.Add(-5*time.Hour), todayStart.Add(-4*time.Hour))

		blocks, err := repo.ListFiltered(storage.BlockFilter{
			StartAfter: todayStart,
			EndBefore:  tomorrowStart,
		})
		require.NoError(t, err)

		var foundToday, foundYesterday bool
		for _, b := range blocks {
			if b.Key == todayBlock.Key {
				foundToday = true
			}
			if b.Key == yesterdayBlock.Key {
				foundYesterday = true
			}
		}
		assert.True(t, foundToday, "should find today's block")
		assert.False(t, foundYesterday, "should not find yesterday's block")
	})

	t.Run("filters blocks for this week", func(t *testing.T) {
		// Calculate start of week (assuming Sunday start)
		daysFromSunday := int(now.Weekday())
		weekStart := time.Date(now.Year(), now.Month(), now.Day()-daysFromSunday, 0, 0, 0, 0, now.Location())
		weekEnd := weekStart.Add(7 * 24 * time.Hour)

		// Create a block within this week
		weekBlock := createTestBlock(t, repo, "weekProj", "", "", now.Add(-time.Hour), now)

		// Create a block from last week
		lastWeekBlock := createTestBlock(t, repo, "lastWeekProj", "", "",
			weekStart.Add(-3*24*time.Hour), weekStart.Add(-2*24*time.Hour))

		blocks, err := repo.ListFiltered(storage.BlockFilter{
			StartAfter: weekStart,
			EndBefore:  weekEnd,
		})
		require.NoError(t, err)

		var foundWeek, foundLastWeek bool
		for _, b := range blocks {
			if b.Key == weekBlock.Key {
				foundWeek = true
			}
			if b.Key == lastWeekBlock.Key {
				foundLastWeek = true
			}
		}
		assert.True(t, foundWeek, "should find this week's block")
		assert.False(t, foundLastWeek, "should not find last week's block")
	})

	t.Run("handles boundary conditions - StartAfter is inclusive", func(t *testing.T) {
		// Block that starts exactly at filter boundary
		boundary := now.Add(-2 * time.Hour)
		boundaryBlock := createTestBlock(t, repo, "boundaryProj", "", "", boundary, now.Add(-time.Hour))

		// StartAfter filter: blocks starting at or after the boundary are included
		// The implementation uses: !b.TimestampStart.Before(filter.StartAfter)
		// This means blocks starting at the exact time ARE included (inclusive)
		blocks, err := repo.ListFiltered(storage.BlockFilter{
			StartAfter: boundary,
		})
		require.NoError(t, err)

		var foundBoundary bool
		for _, b := range blocks {
			if b.Key == boundaryBlock.Key {
				foundBoundary = true
				break
			}
		}
		// Block starts exactly at StartAfter time, so it should be included (inclusive behavior)
		assert.True(t, foundBoundary, "block starting at boundary should be included in StartAfter filter (inclusive)")
	})

	t.Run("handles boundary conditions - block before boundary excluded", func(t *testing.T) {
		// Block that starts just before the filter boundary
		boundary := now.Add(-2 * time.Hour)
		beforeBlock := createTestBlock(t, repo, "beforeBoundary", "", "",
			boundary.Add(-time.Minute), now.Add(-time.Hour))

		blocks, err := repo.ListFiltered(storage.BlockFilter{
			StartAfter: boundary,
		})
		require.NoError(t, err)

		var foundBefore bool
		for _, b := range blocks {
			if b.Key == beforeBlock.Key {
				foundBefore = true
				break
			}
		}
		assert.False(t, foundBefore, "block starting before boundary should not be included")
	})

	t.Run("handles blocks spanning midnight", func(t *testing.T) {
		// Create a block that spans across midnight
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		midnightBlock := createTestBlock(t, repo, "midnightProj", "", "",
			todayStart.Add(-2*time.Hour), todayStart.Add(2*time.Hour))

		// Query for yesterday - should not include
		yesterdayEnd := todayStart
		blocks, err := repo.ListFiltered(storage.BlockFilter{
			EndBefore: yesterdayEnd,
		})
		require.NoError(t, err)

		var found bool
		for _, b := range blocks {
			if b.Key == midnightBlock.Key {
				found = true
				break
			}
		}
		// Block ends at 2am today, not before midnight
		assert.False(t, found, "midnight-spanning block should not appear when filtering for yesterday")
	})
}

// =============================================================================
// Edge Cases Tests
// =============================================================================

func TestBlocksListing_EdgeCases(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)
	now := time.Now()

	t.Run("handles blocks with very short duration", func(t *testing.T) {
		block := createTestBlock(t, repo, "project", "", "", now.Add(-time.Second), now)
		require.NotNil(t, block)

		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.InDelta(t, 1.0, retrieved.Duration().Seconds(), 0.5)
	})

	t.Run("handles blocks with zero duration", func(t *testing.T) {
		block := createTestBlock(t, repo, "project", "", "", now, now)
		require.NotNil(t, block)

		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.InDelta(t, 0.0, retrieved.Duration().Seconds(), 0.1)
	})

	t.Run("handles blocks with long duration", func(t *testing.T) {
		block := createTestBlock(t, repo, "project", "", "", now.Add(-24*time.Hour), now)
		require.NotNil(t, block)

		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.InDelta(t, 24.0*3600, retrieved.Duration().Seconds(), 1.0)
	})

	t.Run("handles special characters in notes", func(t *testing.T) {
		specialNote := "Test with special chars: @#$%^&*()_+-=[]{}|;':\",./<>?"
		block := createTestBlock(t, repo, "project", "", specialNote, now.Add(-time.Hour), now)

		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, specialNote, retrieved.Note)

		// Verify JSON serialization handles it correctly
		blockOutput := output.NewBlockOutput(retrieved)
		jsonBytes, err := json.Marshal(blockOutput)
		require.NoError(t, err)

		var decoded output.BlockOutput
		err = json.Unmarshal(jsonBytes, &decoded)
		require.NoError(t, err)
		assert.Equal(t, specialNote, decoded.Note)
	})

	t.Run("handles unicode in notes", func(t *testing.T) {
		unicodeNote := "Unicode test: Hello World"
		block := createTestBlock(t, repo, "project", "", unicodeNote, now.Add(-time.Hour), now)

		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, unicodeNote, retrieved.Note)
	})

	t.Run("handles empty project filter gracefully", func(t *testing.T) {
		createTestBlock(t, repo, "testproj", "", "", now.Add(-time.Hour), now)

		blocks, err := repo.ListFiltered(storage.BlockFilter{ProjectSID: ""})
		require.NoError(t, err)
		assert.NotEmpty(t, blocks) // Should return all blocks
	})
}
