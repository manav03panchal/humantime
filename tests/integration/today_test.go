// Package integration provides integration tests for Humantime today feature.
// These tests verify the today summary functionality.
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
// Today Feature Tests
// =============================================================================

func TestTodayFeature_GetTodaysBlocks(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)

	t.Run("returns only today's blocks", func(t *testing.T) {
		// Create blocks for today
		block1 := model.NewBlock(config.UserKey, "todayproject1", "", "", time.Now().Add(-2*time.Hour))
		block1.TimestampEnd = time.Now().Add(-1 * time.Hour)
		require.NoError(t, blockRepo.Create(block1))

		block2 := model.NewBlock(config.UserKey, "todayproject2", "", "", time.Now().Add(-30*time.Minute))
		block2.TimestampEnd = time.Now()
		require.NoError(t, blockRepo.Create(block2))

		// Create block for yesterday
		yesterdayBlock := model.NewBlock(config.UserKey, "yesterdayproject", "", "",
			time.Now().Add(-25*time.Hour))
		yesterdayBlock.TimestampEnd = time.Now().Add(-24 * time.Hour)
		require.NoError(t, blockRepo.Create(yesterdayBlock))

		// Filter for today
		filter := storage.BlockFilter{
			StartAfter: todayStart,
			EndBefore:  todayEnd,
		}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		// Should only have today's blocks
		todayCount := 0
		yesterdayCount := 0
		for _, b := range blocks {
			if b.ProjectSID == "todayproject1" || b.ProjectSID == "todayproject2" {
				todayCount++
			}
			if b.ProjectSID == "yesterdayproject" {
				yesterdayCount++
			}
		}

		assert.GreaterOrEqual(t, todayCount, 2)
		assert.Equal(t, 0, yesterdayCount)
	})

	t.Run("includes active blocks started today", func(t *testing.T) {
		// Create an active block
		activeBlock := model.NewBlock(config.UserKey, "activetoday", "", "", time.Now().Add(-15*time.Minute))
		require.NoError(t, blockRepo.Create(activeBlock))

		// Filter for today
		filter := storage.BlockFilter{
			StartAfter: todayStart,
			EndBefore:  todayEnd,
		}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		// Find our active block
		found := false
		for _, b := range blocks {
			if b.ProjectSID == "activetoday" {
				found = true
				assert.True(t, b.IsActive())
			}
		}
		assert.True(t, found, "active block should be included")
	})
}

func TestTodayFeature_AggregateByProject(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("aggregates blocks by project", func(t *testing.T) {
		// Create multiple blocks for same project
		block1 := model.NewBlock(config.UserKey, "aggproject", "", "", time.Now().Add(-3*time.Hour))
		block1.TimestampEnd = time.Now().Add(-2 * time.Hour)
		require.NoError(t, blockRepo.Create(block1))

		block2 := model.NewBlock(config.UserKey, "aggproject", "", "", time.Now().Add(-1*time.Hour))
		block2.TimestampEnd = time.Now()
		require.NoError(t, blockRepo.Create(block2))

		block3 := model.NewBlock(config.UserKey, "otherproject", "", "", time.Now().Add(-30*time.Minute))
		block3.TimestampEnd = time.Now().Add(-15 * time.Minute)
		require.NoError(t, blockRepo.Create(block3))

		// Get all blocks and aggregate
		blocks, err := blockRepo.List()
		require.NoError(t, err)

		aggs := storage.AggregateByProject(blocks)

		// Find our projects
		var aggProjectAgg, otherProjectAgg *storage.ProjectAggregate
		for i := range aggs {
			if aggs[i].ProjectSID == "aggproject" {
				aggProjectAgg = &aggs[i]
			}
			if aggs[i].ProjectSID == "otherproject" {
				otherProjectAgg = &aggs[i]
			}
		}

		require.NotNil(t, aggProjectAgg)
		require.NotNil(t, otherProjectAgg)

		// aggproject should have 2 blocks totaling about 2 hours
		assert.Equal(t, 2, aggProjectAgg.BlockCount)
		assert.InDelta(t, 2*time.Hour, aggProjectAgg.Duration, float64(5*time.Minute))

		// otherproject should have 1 block
		assert.Equal(t, 1, otherProjectAgg.BlockCount)
	})

	t.Run("sorts by duration descending", func(t *testing.T) {
		blocks := []*model.Block{
			{ProjectSID: "short", TimestampStart: time.Now().Add(-30 * time.Minute), TimestampEnd: time.Now()},
			{ProjectSID: "long", TimestampStart: time.Now().Add(-3 * time.Hour), TimestampEnd: time.Now()},
			{ProjectSID: "medium", TimestampStart: time.Now().Add(-1 * time.Hour), TimestampEnd: time.Now()},
		}

		aggs := storage.AggregateByProject(blocks)

		// Should be sorted by duration, longest first
		assert.Equal(t, "long", aggs[0].ProjectSID)
		assert.Equal(t, "medium", aggs[1].ProjectSID)
		assert.Equal(t, "short", aggs[2].ProjectSID)
	})
}

func TestTodayFeature_GoalProgress(t *testing.T) {
	db := setupTestDB(t)
	goalRepo := storage.NewGoalRepo(db)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("calculates goal progress correctly", func(t *testing.T) {
		// Create a daily goal of 4 hours
		goal := model.NewGoal("goalproject", model.GoalTypeDaily, 4*time.Hour)
		require.NoError(t, goalRepo.Create(goal))

		// Create blocks totaling 2 hours
		block := model.NewBlock(config.UserKey, "goalproject", "", "", time.Now().Add(-2*time.Hour))
		block.TimestampEnd = time.Now()
		require.NoError(t, blockRepo.Create(block))

		// Get goal and calculate progress
		savedGoal, err := goalRepo.Get("goalproject")
		require.NoError(t, err)
		require.NotNil(t, savedGoal)

		progress := savedGoal.CalculateProgress(2 * time.Hour)

		assert.InDelta(t, 50.0, progress.Percentage, 1.0)
		assert.Equal(t, 2*time.Hour, progress.Remaining)
		assert.False(t, progress.IsComplete)
	})

	t.Run("handles goal completion", func(t *testing.T) {
		goal := model.NewGoal("completeproject", model.GoalTypeDaily, 2*time.Hour)

		// Calculate progress when goal is met
		progress := goal.CalculateProgress(2 * time.Hour)
		assert.True(t, progress.IsComplete)
		assert.InDelta(t, 100.0, progress.Percentage, 1.0)

		// Calculate progress when goal is exceeded
		progressOver := goal.CalculateProgress(3 * time.Hour)
		assert.True(t, progressOver.IsComplete)
		assert.Greater(t, progressOver.Percentage, 100.0)
	})
}
