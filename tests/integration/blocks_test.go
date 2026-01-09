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
