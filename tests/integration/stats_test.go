// Package integration provides integration tests for Humantime stats feature.
// These tests verify statistics calculations, time aggregations, and period ranges.
package integration

import (
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Time Calculation Tests
// =============================================================================

func TestStats_TotalDuration(t *testing.T) {
	t.Run("calculates total duration of multiple blocks", func(t *testing.T) {
		now := time.Now()
		blocks := []*model.Block{
			{TimestampStart: now.Add(-3 * time.Hour), TimestampEnd: now.Add(-2 * time.Hour)},   // 1 hour
			{TimestampStart: now.Add(-90 * time.Minute), TimestampEnd: now.Add(-60 * time.Minute)}, // 30 minutes
			{TimestampStart: now.Add(-45 * time.Minute), TimestampEnd: now.Add(-15 * time.Minute)}, // 30 minutes
		}

		total := storage.TotalDuration(blocks)

		assert.Equal(t, 2*time.Hour, total)
	})

	t.Run("handles empty block list", func(t *testing.T) {
		blocks := []*model.Block{}
		total := storage.TotalDuration(blocks)
		assert.Equal(t, time.Duration(0), total)
	})

	t.Run("includes active block duration", func(t *testing.T) {
		now := time.Now()
		blocks := []*model.Block{
			{TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now.Add(-30 * time.Minute)}, // 30 minutes
			{TimestampStart: now.Add(-15 * time.Minute)}, // active, ~15 minutes
		}

		total := storage.TotalDuration(blocks)

		// Should be approximately 45 minutes (30 + ~15)
		assert.InDelta(t, 45*time.Minute, total, float64(1*time.Minute))
	})

	t.Run("handles single block", func(t *testing.T) {
		now := time.Now()
		blocks := []*model.Block{
			{TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},
		}

		total := storage.TotalDuration(blocks)
		assert.Equal(t, 1*time.Hour, total)
	})
}

func TestStats_BlockDuration(t *testing.T) {
	t.Run("calculates completed block duration", func(t *testing.T) {
		now := time.Now()
		block := &model.Block{
			TimestampStart: now.Add(-2 * time.Hour),
			TimestampEnd:   now.Add(-1 * time.Hour),
		}

		assert.Equal(t, 1*time.Hour, block.Duration())
		assert.False(t, block.IsActive())
	})

	t.Run("calculates active block duration from start to now", func(t *testing.T) {
		block := &model.Block{
			TimestampStart: time.Now().Add(-30 * time.Minute),
			// TimestampEnd not set (active)
		}

		assert.True(t, block.IsActive())
		// Duration should be approximately 30 minutes
		assert.InDelta(t, 30*time.Minute, block.Duration(), float64(5*time.Second))
	})

	t.Run("duration in seconds", func(t *testing.T) {
		now := time.Now()
		block := &model.Block{
			TimestampStart: now.Add(-90 * time.Second),
			TimestampEnd:   now,
		}

		assert.Equal(t, int64(90), block.DurationSeconds())
	})
}

// =============================================================================
// Today Summary Tests
// =============================================================================

func TestStats_TodaySummary(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)

	t.Run("calculates today's total time", func(t *testing.T) {
		// Create blocks for today totaling 3 hours
		block1 := model.NewBlock(config.UserKey, "proj1", "", "", todayStart.Add(9*time.Hour))
		block1.TimestampEnd = todayStart.Add(11 * time.Hour) // 2 hours
		require.NoError(t, blockRepo.Create(block1))

		block2 := model.NewBlock(config.UserKey, "proj2", "", "", todayStart.Add(14*time.Hour))
		block2.TimestampEnd = todayStart.Add(15 * time.Hour) // 1 hour
		require.NoError(t, blockRepo.Create(block2))

		filter := storage.BlockFilter{
			StartAfter: todayStart,
			EndBefore:  todayEnd,
		}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		// Filter to only the blocks we created
		var todayBlocks []*model.Block
		for _, b := range blocks {
			if b.ProjectSID == "proj1" || b.ProjectSID == "proj2" {
				todayBlocks = append(todayBlocks, b)
			}
		}

		total := storage.TotalDuration(todayBlocks)
		assert.Equal(t, 3*time.Hour, total)
	})

	t.Run("excludes blocks from other days", func(t *testing.T) {
		// Create a block for yesterday
		yesterdayBlock := model.NewBlock(config.UserKey, "yesterdayproj", "", "",
			todayStart.Add(-25*time.Hour))
		yesterdayBlock.TimestampEnd = todayStart.Add(-24 * time.Hour)
		require.NoError(t, blockRepo.Create(yesterdayBlock))

		filter := storage.BlockFilter{
			StartAfter: todayStart,
			EndBefore:  todayEnd,
		}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		// Verify yesterday's block is not included
		for _, b := range blocks {
			assert.NotEqual(t, "yesterdayproj", b.ProjectSID)
		}
	})
}

// =============================================================================
// Project Breakdown Tests
// =============================================================================

func TestStats_ProjectBreakdown(t *testing.T) {
	t.Run("aggregates time by project correctly", func(t *testing.T) {
		now := time.Now()
		blocks := []*model.Block{
			{ProjectSID: "projectA", TimestampStart: now.Add(-4 * time.Hour), TimestampEnd: now.Add(-3 * time.Hour)},
			{ProjectSID: "projectA", TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},
			{ProjectSID: "projectB", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
		}

		aggs := storage.AggregateByProject(blocks)

		// Find projectA and projectB
		var aggA, aggB *storage.ProjectAggregate
		for i := range aggs {
			if aggs[i].ProjectSID == "projectA" {
				aggA = &aggs[i]
			}
			if aggs[i].ProjectSID == "projectB" {
				aggB = &aggs[i]
			}
		}

		require.NotNil(t, aggA)
		require.NotNil(t, aggB)

		assert.Equal(t, 2*time.Hour, aggA.Duration)
		assert.Equal(t, 2, aggA.BlockCount)
		assert.Equal(t, 1*time.Hour, aggB.Duration)
		assert.Equal(t, 1, aggB.BlockCount)
	})

	t.Run("sorts projects by duration descending", func(t *testing.T) {
		now := time.Now()
		blocks := []*model.Block{
			{ProjectSID: "short", TimestampStart: now.Add(-15 * time.Minute), TimestampEnd: now},
			{ProjectSID: "longest", TimestampStart: now.Add(-4 * time.Hour), TimestampEnd: now},
			{ProjectSID: "medium", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
		}

		aggs := storage.AggregateByProject(blocks)

		assert.Equal(t, "longest", aggs[0].ProjectSID)
		assert.Equal(t, "medium", aggs[1].ProjectSID)
		assert.Equal(t, "short", aggs[2].ProjectSID)
	})

	t.Run("calculates percentage correctly", func(t *testing.T) {
		now := time.Now()
		blocks := []*model.Block{
			{ProjectSID: "half", TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now},
			{ProjectSID: "quarter", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
			{ProjectSID: "quarter2", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
		}

		aggs := storage.AggregateByProject(blocks)

		var totalDuration time.Duration
		for _, agg := range aggs {
			totalDuration += agg.Duration
		}

		// Verify percentage calculations
		for _, agg := range aggs {
			percentage := float64(agg.Duration) / float64(totalDuration) * 100
			if agg.ProjectSID == "half" {
				assert.InDelta(t, 50.0, percentage, 1.0)
			} else {
				assert.InDelta(t, 25.0, percentage, 1.0)
			}
		}
	})

	t.Run("handles single project", func(t *testing.T) {
		now := time.Now()
		blocks := []*model.Block{
			{ProjectSID: "only", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
		}

		aggs := storage.AggregateByProject(blocks)

		assert.Len(t, aggs, 1)
		assert.Equal(t, "only", aggs[0].ProjectSID)
		assert.Equal(t, 1*time.Hour, aggs[0].Duration)
	})

	t.Run("handles empty blocks", func(t *testing.T) {
		blocks := []*model.Block{}
		aggs := storage.AggregateByProject(blocks)
		assert.Empty(t, aggs)
	})
}

// =============================================================================
// Weekly Stats Tests
// =============================================================================

func TestStats_WeeklyStats(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("filters blocks for this week", func(t *testing.T) {
		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		weekStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		weekEnd := weekStart.AddDate(0, 0, 7)

		// Create block for this week
		thisWeekBlock := model.NewBlock(config.UserKey, "weekproject", "", "", weekStart.Add(24*time.Hour))
		thisWeekBlock.TimestampEnd = weekStart.Add(26 * time.Hour)
		require.NoError(t, blockRepo.Create(thisWeekBlock))

		// Create block for last week
		lastWeekBlock := model.NewBlock(config.UserKey, "lastweekproject", "", "", weekStart.Add(-48*time.Hour))
		lastWeekBlock.TimestampEnd = weekStart.Add(-46 * time.Hour)
		require.NoError(t, blockRepo.Create(lastWeekBlock))

		filter := storage.BlockFilter{
			StartAfter: weekStart,
			EndBefore:  weekEnd,
		}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		// Verify this week's block is included
		foundThisWeek := false
		foundLastWeek := false
		for _, b := range blocks {
			if b.ProjectSID == "weekproject" {
				foundThisWeek = true
			}
			if b.ProjectSID == "lastweekproject" {
				foundLastWeek = true
			}
		}

		assert.True(t, foundThisWeek, "this week's block should be included")
		assert.False(t, foundLastWeek, "last week's block should not be included")
	})

	t.Run("aggregates weekly totals by project", func(t *testing.T) {
		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		weekStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		weekEnd := weekStart.AddDate(0, 0, 7)

		// Create multiple blocks for the week
		for i := 0; i < 5; i++ {
			block := model.NewBlock(config.UserKey, "weeklyproj", "", "",
				weekStart.Add(time.Duration(i*24+9)*time.Hour))
			block.TimestampEnd = weekStart.Add(time.Duration(i*24+10) * time.Hour) // 1 hour each
			require.NoError(t, blockRepo.Create(block))
		}

		filter := storage.BlockFilter{
			ProjectSID: "weeklyproj",
			StartAfter: weekStart,
			EndBefore:  weekEnd,
		}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		total := storage.TotalDuration(blocks)
		assert.Equal(t, 5*time.Hour, total)
	})
}

// =============================================================================
// Time Range Parsing Tests
// =============================================================================

func TestStats_PeriodRanges(t *testing.T) {
	now := time.Now()

	t.Run("today range", func(t *testing.T) {
		tr := parser.GetPeriodRange("today")

		expectedStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		expectedEnd := expectedStart.AddDate(0, 0, 1)

		assert.Equal(t, expectedStart, tr.Start)
		assert.Equal(t, expectedEnd, tr.End)
	})

	t.Run("yesterday range", func(t *testing.T) {
		tr := parser.GetPeriodRange("yesterday")

		expectedStart := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		expectedEnd := expectedStart.AddDate(0, 0, 1)

		assert.Equal(t, expectedStart, tr.Start)
		assert.Equal(t, expectedEnd, tr.End)
	})

	t.Run("this week range", func(t *testing.T) {
		tr := parser.GetPeriodRange("this week")

		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		expectedStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		expectedEnd := expectedStart.AddDate(0, 0, 7)

		assert.Equal(t, expectedStart, tr.Start)
		assert.Equal(t, expectedEnd, tr.End)
	})

	t.Run("last week range", func(t *testing.T) {
		tr := parser.GetPeriodRange("last week")

		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		thisWeekStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		expectedStart := thisWeekStart.AddDate(0, 0, -7)
		expectedEnd := thisWeekStart

		assert.Equal(t, expectedStart, tr.Start)
		assert.Equal(t, expectedEnd, tr.End)
	})

	t.Run("this month range", func(t *testing.T) {
		tr := parser.GetPeriodRange("this month")

		expectedStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		expectedEnd := expectedStart.AddDate(0, 1, 0)

		assert.Equal(t, expectedStart, tr.Start)
		assert.Equal(t, expectedEnd, tr.End)
	})

	t.Run("last month range", func(t *testing.T) {
		tr := parser.GetPeriodRange("last month")

		expectedStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		expectedEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

		assert.Equal(t, expectedStart, tr.Start)
		assert.Equal(t, expectedEnd, tr.End)
	})

	t.Run("this year range", func(t *testing.T) {
		tr := parser.GetPeriodRange("this year")

		expectedStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		expectedEnd := expectedStart.AddDate(1, 0, 0)

		assert.Equal(t, expectedStart, tr.Start)
		assert.Equal(t, expectedEnd, tr.End)
	})

	t.Run("default to today for unknown period", func(t *testing.T) {
		tr := parser.GetPeriodRange("invalid")

		expectedStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		expectedEnd := expectedStart.AddDate(0, 0, 1)

		assert.Equal(t, expectedStart, tr.Start)
		assert.Equal(t, expectedEnd, tr.End)
	})
}

// =============================================================================
// Filter Tests
// =============================================================================

func TestStats_BlockFilter(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	now := time.Now()

	t.Run("filters by project", func(t *testing.T) {
		block1 := model.NewBlock(config.UserKey, "filterproj1", "", "", now.Add(-1*time.Hour))
		block1.TimestampEnd = now
		require.NoError(t, blockRepo.Create(block1))

		block2 := model.NewBlock(config.UserKey, "filterproj2", "", "", now.Add(-30*time.Minute))
		block2.TimestampEnd = now
		require.NoError(t, blockRepo.Create(block2))

		filter := storage.BlockFilter{ProjectSID: "filterproj1"}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		for _, b := range blocks {
			assert.Equal(t, "filterproj1", b.ProjectSID)
		}
	})

	t.Run("filters by task", func(t *testing.T) {
		block := model.NewBlock(config.UserKey, "taskproj", "task1", "", now.Add(-1*time.Hour))
		block.TimestampEnd = now
		require.NoError(t, blockRepo.Create(block))

		block2 := model.NewBlock(config.UserKey, "taskproj", "task2", "", now.Add(-30*time.Minute))
		block2.TimestampEnd = now
		require.NoError(t, blockRepo.Create(block2))

		filter := storage.BlockFilter{ProjectSID: "taskproj", TaskSID: "task1"}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		for _, b := range blocks {
			assert.Equal(t, "task1", b.TaskSID)
		}
	})

	t.Run("filters by tag", func(t *testing.T) {
		block := model.NewBlock(config.UserKey, "tagproj", "", "", now.Add(-1*time.Hour))
		block.TimestampEnd = now
		block.Tags = []string{"urgent", "work"}
		require.NoError(t, blockRepo.Create(block))

		block2 := model.NewBlock(config.UserKey, "tagproj", "", "", now.Add(-30*time.Minute))
		block2.TimestampEnd = now
		block2.Tags = []string{"personal"}
		require.NoError(t, blockRepo.Create(block2))

		filter := storage.BlockFilter{Tag: "urgent"}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		for _, b := range blocks {
			assert.True(t, b.HasTag("urgent"))
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		// Create several blocks
		for i := 0; i < 10; i++ {
			block := model.NewBlock(config.UserKey, "limitproj", "", "",
				now.Add(-time.Duration(i)*time.Hour))
			block.TimestampEnd = now.Add(-time.Duration(i)*time.Hour + 30*time.Minute)
			require.NoError(t, blockRepo.Create(block))
		}

		filter := storage.BlockFilter{ProjectSID: "limitproj", Limit: 3}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(blocks), 3)
	})

	t.Run("combines multiple filters", func(t *testing.T) {
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		todayEnd := todayStart.AddDate(0, 0, 1)

		block := model.NewBlock(config.UserKey, "combofilter", "task", "", now.Add(-1*time.Hour))
		block.TimestampEnd = now
		block.Tags = []string{"important"}
		require.NoError(t, blockRepo.Create(block))

		filter := storage.BlockFilter{
			ProjectSID: "combofilter",
			TaskSID:    "task",
			Tag:        "important",
			StartAfter: todayStart,
			EndBefore:  todayEnd,
		}
		blocks, err := blockRepo.ListFiltered(filter)
		require.NoError(t, err)

		require.NotEmpty(t, blocks)
		for _, b := range blocks {
			assert.Equal(t, "combofilter", b.ProjectSID)
			assert.Equal(t, "task", b.TaskSID)
			assert.True(t, b.HasTag("important"))
		}
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestStats_EdgeCases(t *testing.T) {
	t.Run("handles very short durations", func(t *testing.T) {
		now := time.Now()
		blocks := []*model.Block{
			{TimestampStart: now.Add(-1 * time.Second), TimestampEnd: now},
		}

		total := storage.TotalDuration(blocks)
		assert.Equal(t, 1*time.Second, total)
	})

	t.Run("handles very long durations", func(t *testing.T) {
		now := time.Now()
		blocks := []*model.Block{
			{TimestampStart: now.Add(-24 * time.Hour), TimestampEnd: now},
		}

		total := storage.TotalDuration(blocks)
		assert.Equal(t, 24*time.Hour, total)
	})

	t.Run("aggregates projects with same name correctly", func(t *testing.T) {
		now := time.Now()
		blocks := []*model.Block{
			{ProjectSID: "sameproj", TimestampStart: now.Add(-3 * time.Hour), TimestampEnd: now.Add(-2 * time.Hour)},
			{ProjectSID: "sameproj", TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},
			{ProjectSID: "sameproj", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
		}

		aggs := storage.AggregateByProject(blocks)

		require.Len(t, aggs, 1)
		assert.Equal(t, "sameproj", aggs[0].ProjectSID)
		assert.Equal(t, 3*time.Hour, aggs[0].Duration)
		assert.Equal(t, 3, aggs[0].BlockCount)
	})

	t.Run("handles blocks spanning midnight", func(t *testing.T) {
		now := time.Now()
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		// Block that started yesterday and ended today
		block := &model.Block{
			ProjectSID:     "overnight",
			TimestampStart: todayStart.Add(-2 * time.Hour), // 10 PM yesterday
			TimestampEnd:   todayStart.Add(2 * time.Hour),  // 2 AM today
		}

		// Total duration should be 4 hours
		assert.Equal(t, 4*time.Hour, block.Duration())
	})
}
