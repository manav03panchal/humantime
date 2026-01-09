// Package integration provides integration tests for Humantime.
// These tests verify the interaction between the parser and storage layers,
// testing natural language time expressions with a real in-memory database.
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
// Test Helpers
// =============================================================================

// createBlockWithNaturalLanguageTime creates a block using a natural language time expression.
func createBlockWithNaturalLanguageTime(t *testing.T, repo *storage.BlockRepo, ownerKey, projectSID, taskSID, note, timeExpr string) *model.Block {
	t.Helper()

	result := parser.ParseTimestamp(timeExpr)
	require.NoError(t, result.Error, "failed to parse time expression: %s", timeExpr)

	block := model.NewBlock(ownerKey, projectSID, taskSID, note, result.Time)
	require.NoError(t, repo.Create(block), "failed to create block")

	return block
}

// =============================================================================
// Relative Time Expression Tests
// =============================================================================

func TestRelativeTimeExpressions(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)

	t.Run("2 hours ago creates block with correct timestamp", func(t *testing.T) {
		block := createBlockWithNaturalLanguageTime(t, blockRepo, "user-1", "project1", "", "", "2 hours ago")

		// Verify the block was stored and can be retrieved
		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		// Verify the timestamp is approximately 2 hours ago
		expectedTime := time.Now().Add(-2 * time.Hour)
		assert.WithinDuration(t, expectedTime, retrieved.TimestampStart, 2*time.Minute,
			"timestamp should be approximately 2 hours ago")
	})

	t.Run("1 hour ago creates block with correct timestamp", func(t *testing.T) {
		block := createBlockWithNaturalLanguageTime(t, blockRepo, "user-1", "project2", "", "", "1 hour ago")

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		expectedTime := time.Now().Add(-1 * time.Hour)
		assert.WithinDuration(t, expectedTime, retrieved.TimestampStart, 2*time.Minute,
			"timestamp should be approximately 1 hour ago")
	})

	t.Run("30 minutes ago creates block with correct timestamp", func(t *testing.T) {
		block := createBlockWithNaturalLanguageTime(t, blockRepo, "user-1", "project3", "", "", "30 minutes ago")

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		expectedTime := time.Now().Add(-30 * time.Minute)
		assert.WithinDuration(t, expectedTime, retrieved.TimestampStart, 2*time.Minute,
			"timestamp should be approximately 30 minutes ago")
	})

	t.Run("5 days ago creates block with correct timestamp", func(t *testing.T) {
		block := createBlockWithNaturalLanguageTime(t, blockRepo, "user-1", "project4", "", "", "5 days ago")

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		expectedTime := time.Now().Add(-5 * 24 * time.Hour)
		assert.WithinDuration(t, expectedTime, retrieved.TimestampStart, 24*time.Hour,
			"timestamp should be approximately 5 days ago")
	})
}

// =============================================================================
// Absolute Time Expression Tests
// =============================================================================

func TestAbsoluteTimeExpressions(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)

	t.Run("9am creates block with correct time", func(t *testing.T) {
		result := parser.ParseTimestamp("9am")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project1", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		// Verify hour is 9
		assert.Equal(t, 9, retrieved.TimestampStart.Hour(), "hour should be 9")
		assert.Equal(t, 0, retrieved.TimestampStart.Minute(), "minute should be 0")
	})

	t.Run("14:30 creates block with correct time", func(t *testing.T) {
		result := parser.ParseTimestamp("14:30")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project2", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		assert.Equal(t, 14, retrieved.TimestampStart.Hour(), "hour should be 14")
		assert.Equal(t, 30, retrieved.TimestampStart.Minute(), "minute should be 30")
	})

	t.Run("2:30pm creates block with correct time", func(t *testing.T) {
		result := parser.ParseTimestamp("2:30pm")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project3", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		assert.Equal(t, 14, retrieved.TimestampStart.Hour(), "hour should be 14 (2pm)")
		assert.Equal(t, 30, retrieved.TimestampStart.Minute(), "minute should be 30")
	})

	t.Run("9:00 AM creates block with correct time", func(t *testing.T) {
		result := parser.ParseTimestamp("9:00 AM")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project4", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		assert.Equal(t, 9, retrieved.TimestampStart.Hour(), "hour should be 9")
		assert.Equal(t, 0, retrieved.TimestampStart.Minute(), "minute should be 0")
	})

	t.Run("11:45 PM creates block with correct time", func(t *testing.T) {
		result := parser.ParseTimestamp("11:45 PM")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project5", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		assert.Equal(t, 23, retrieved.TimestampStart.Hour(), "hour should be 23 (11pm)")
		assert.Equal(t, 45, retrieved.TimestampStart.Minute(), "minute should be 45")
	})
}

// =============================================================================
// Time Range Expression Tests
// =============================================================================

func TestTimeRangeExpressions(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)

	t.Run("from 9am to 11am creates block with start and end times", func(t *testing.T) {
		startResult := parser.ParseTimestamp("9am")
		require.NoError(t, startResult.Error, "failed to parse start time")

		endResult := parser.ParseTimestamp("11am")
		require.NoError(t, endResult.Error, "failed to parse end time")

		block := model.NewBlock("user-1", "project1", "task1", "morning work", startResult.Time)
		block.TimestampEnd = endResult.Time
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		// Verify start time
		assert.Equal(t, 9, retrieved.TimestampStart.Hour(), "start hour should be 9")
		assert.Equal(t, 0, retrieved.TimestampStart.Minute(), "start minute should be 0")

		// Verify end time
		assert.Equal(t, 11, retrieved.TimestampEnd.Hour(), "end hour should be 11")
		assert.Equal(t, 0, retrieved.TimestampEnd.Minute(), "end minute should be 0")

		// Verify block is not active (has end time)
		assert.False(t, retrieved.IsActive(), "block should not be active with end time set")
	})

	t.Run("from 2:30pm to 5:00pm creates block with correct duration", func(t *testing.T) {
		startResult := parser.ParseTimestamp("2:30pm")
		require.NoError(t, startResult.Error)

		endResult := parser.ParseTimestamp("5:00pm")
		require.NoError(t, endResult.Error)

		block := model.NewBlock("user-1", "project2", "task2", "afternoon work", startResult.Time)
		block.TimestampEnd = endResult.Time
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		// Verify duration is approximately 2.5 hours
		expectedDuration := 2*time.Hour + 30*time.Minute
		assert.InDelta(t, expectedDuration.Seconds(), retrieved.Duration().Seconds(), 60,
			"duration should be approximately 2.5 hours")
	})

	t.Run("time range blocks can be filtered by time range", func(t *testing.T) {
		now := time.Now()

		// Create a block that starts 3 hours ago and ended 1 hour ago
		startTime := now.Add(-3 * time.Hour)
		endTime := now.Add(-1 * time.Hour)

		block := model.NewBlock("user-1", "project3", "", "", startTime)
		block.TimestampEnd = endTime
		require.NoError(t, blockRepo.Create(block))

		// Filter for blocks in the last 4 hours
		blocks, err := blockRepo.ListByTimeRange(now.Add(-4*time.Hour), now)
		require.NoError(t, err)

		// Find our block in the results
		var found bool
		for _, b := range blocks {
			if b.Key == block.Key {
				found = true
				break
			}
		}
		assert.True(t, found, "block should be found in time range query")
	})
}

// =============================================================================
// Period Expression Tests
// =============================================================================

func TestPeriodExpressions(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	now := time.Now()

	t.Run("yesterday returns correct date", func(t *testing.T) {
		result := parser.ParseTimestamp("yesterday")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project1", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		// Verify the date is yesterday
		expectedDate := now.AddDate(0, 0, -1)
		assert.Equal(t, expectedDate.Year(), retrieved.TimestampStart.Year())
		assert.Equal(t, expectedDate.Month(), retrieved.TimestampStart.Month())
		assert.Equal(t, expectedDate.Day(), retrieved.TimestampStart.Day())
	})

	t.Run("last week returns start of last week", func(t *testing.T) {
		result := parser.ParseTimestamp("last week")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project2", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		// Should be a Monday (start of week)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		expectedMonday := time.Date(now.Year(), now.Month(), now.Day()-weekday+1-7, 0, 0, 0, 0, now.Location())

		assert.Equal(t, expectedMonday.Year(), retrieved.TimestampStart.Year())
		assert.Equal(t, expectedMonday.Month(), retrieved.TimestampStart.Month())
		assert.Equal(t, expectedMonday.Day(), retrieved.TimestampStart.Day())
	})

	t.Run("this week returns start of current week", func(t *testing.T) {
		result := parser.ParseTimestamp("this week")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project3", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		// Calculate expected start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		expectedMonday := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())

		assert.Equal(t, expectedMonday.Year(), retrieved.TimestampStart.Year())
		assert.Equal(t, expectedMonday.Month(), retrieved.TimestampStart.Month())
		assert.Equal(t, expectedMonday.Day(), retrieved.TimestampStart.Day())
	})

	t.Run("last month returns start of last month", func(t *testing.T) {
		result := parser.ParseTimestamp("last month")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project4", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		expectedStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		assert.Equal(t, expectedStart.Year(), retrieved.TimestampStart.Year())
		assert.Equal(t, expectedStart.Month(), retrieved.TimestampStart.Month())
		assert.Equal(t, 1, retrieved.TimestampStart.Day(), "should be first day of month")
	})

	t.Run("this month returns start of current month", func(t *testing.T) {
		result := parser.ParseTimestamp("this month")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project5", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		expectedStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		assert.Equal(t, expectedStart.Year(), retrieved.TimestampStart.Year())
		assert.Equal(t, expectedStart.Month(), retrieved.TimestampStart.Month())
		assert.Equal(t, 1, retrieved.TimestampStart.Day(), "should be first day of month")
	})

	t.Run("last year returns start of last year", func(t *testing.T) {
		result := parser.ParseTimestamp("last year")
		require.NoError(t, result.Error)

		block := model.NewBlock("user-1", "project6", "", "", result.Time)
		require.NoError(t, blockRepo.Create(block))

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		assert.Equal(t, now.Year()-1, retrieved.TimestampStart.Year())
		assert.Equal(t, time.January, retrieved.TimestampStart.Month())
		assert.Equal(t, 1, retrieved.TimestampStart.Day())
	})
}

// =============================================================================
// GetPeriodRange Integration Tests
// =============================================================================

func TestGetPeriodRangeWithDatabase(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	now := time.Now()

	t.Run("blocks can be filtered by today period", func(t *testing.T) {
		// Create a block for today
		todayBlock := model.NewBlock("user-1", "today-project", "", "", now.Add(-1*time.Hour))
		todayBlock.TimestampEnd = now
		require.NoError(t, blockRepo.Create(todayBlock))

		// Create a block for yesterday
		yesterdayStart := now.AddDate(0, 0, -1)
		yesterdayBlock := model.NewBlock("user-1", "yesterday-project", "", "", yesterdayStart)
		yesterdayBlock.TimestampEnd = yesterdayStart.Add(1 * time.Hour)
		require.NoError(t, blockRepo.Create(yesterdayBlock))

		// Get period range for today
		periodRange := parser.GetPeriodRange("today")

		// Filter blocks by today's range
		blocks, err := blockRepo.ListByTimeRange(periodRange.Start, periodRange.End)
		require.NoError(t, err)

		// Verify only today's block is returned
		var foundToday, foundYesterday bool
		for _, b := range blocks {
			if b.Key == todayBlock.Key {
				foundToday = true
			}
			if b.Key == yesterdayBlock.Key {
				foundYesterday = true
			}
		}
		assert.True(t, foundToday, "today's block should be in the results")
		assert.False(t, foundYesterday, "yesterday's block should not be in today's results")
	})

	t.Run("blocks can be filtered by yesterday period", func(t *testing.T) {
		// Get period range for yesterday
		periodRange := parser.GetPeriodRange("yesterday")

		// Create a block within yesterday's range
		yesterdayBlock := model.NewBlock("user-1", "yesterday-test", "", "", periodRange.Start.Add(2*time.Hour))
		yesterdayBlock.TimestampEnd = periodRange.Start.Add(3 * time.Hour)
		require.NoError(t, blockRepo.Create(yesterdayBlock))

		// Filter blocks by yesterday's range
		blocks, err := blockRepo.ListByTimeRange(periodRange.Start, periodRange.End)
		require.NoError(t, err)

		var found bool
		for _, b := range blocks {
			if b.Key == yesterdayBlock.Key {
				found = true
				break
			}
		}
		assert.True(t, found, "yesterday's block should be found in yesterday's range")
	})

	t.Run("this week range contains expected days", func(t *testing.T) {
		periodRange := parser.GetPeriodRange("this week")

		// Verify range is 7 days
		duration := periodRange.End.Sub(periodRange.Start)
		assert.Equal(t, 7*24*time.Hour, duration, "this week range should be 7 days")

		// Verify start is a Monday at midnight
		assert.Equal(t, time.Monday, periodRange.Start.Weekday(), "week should start on Monday")
		assert.Equal(t, 0, periodRange.Start.Hour(), "should start at midnight")
		assert.Equal(t, 0, periodRange.Start.Minute())
		assert.Equal(t, 0, periodRange.Start.Second())
	})
}

// =============================================================================
// Complex Workflow Integration Tests
// =============================================================================

func TestComplexNaturalLanguageWorkflow(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	taskRepo := storage.NewTaskRepo(db)

	t.Run("full workflow with natural language times", func(t *testing.T) {
		// Create a project
		project := model.NewProject("client-work", "Client Work", "#FF5733")
		require.NoError(t, projectRepo.Create(project))

		// Create a task
		task := model.NewTask("client-work", "billing", "Billing Task", "#00FF00")
		require.NoError(t, taskRepo.Create(task))

		// Parse time expressions for a work session
		startResult := parser.ParseTimestamp("9am")
		require.NoError(t, startResult.Error)

		endResult := parser.ParseTimestamp("11am")
		require.NoError(t, endResult.Error)

		// Create a block for morning work
		morningBlock := model.NewBlock("user-1", "client-work", "billing", "Working on invoices", startResult.Time)
		morningBlock.TimestampEnd = endResult.Time
		require.NoError(t, blockRepo.Create(morningBlock))

		// Parse times for afternoon session
		afternoonStartResult := parser.ParseTimestamp("2:30pm")
		require.NoError(t, afternoonStartResult.Error)

		afternoonEndResult := parser.ParseTimestamp("5:00pm")
		require.NoError(t, afternoonEndResult.Error)

		// Create afternoon block
		afternoonBlock := model.NewBlock("user-1", "client-work", "billing", "Finishing invoices", afternoonStartResult.Time)
		afternoonBlock.TimestampEnd = afternoonEndResult.Time
		require.NoError(t, blockRepo.Create(afternoonBlock))

		// Verify blocks can be retrieved by project
		blocks, err := blockRepo.ListByProject("client-work")
		require.NoError(t, err)
		assert.Len(t, blocks, 2, "should have 2 blocks for client-work")

		// Verify blocks can be retrieved by project and task
		taskBlocks, err := blockRepo.ListByProjectAndTask("client-work", "billing")
		require.NoError(t, err)
		assert.Len(t, taskBlocks, 2, "should have 2 blocks for billing task")

		// Verify total duration
		totalDuration := storage.TotalDuration(blocks)
		// Morning: 2 hours, Afternoon: 2.5 hours = 4.5 hours
		expectedDuration := 4*time.Hour + 30*time.Minute
		assert.InDelta(t, expectedDuration.Seconds(), totalDuration.Seconds(), 120,
			"total duration should be approximately 4.5 hours")
	})
}

// =============================================================================
// Error Handling Integration Tests
// =============================================================================

func TestInvalidTimeExpressionHandling(t *testing.T) {
	t.Run("invalid time expression returns error", func(t *testing.T) {
		result := parser.ParseTimestamp("not a valid time expression xyz123")
		assert.Error(t, result.Error, "should return error for invalid time expression")
	})

	t.Run("special characters only returns error", func(t *testing.T) {
		result := parser.ParseTimestamp("!@#$%")
		assert.Error(t, result.Error, "should return error for special characters only")
	})

	t.Run("empty string returns current time", func(t *testing.T) {
		result := parser.ParseTimestamp("")
		assert.NoError(t, result.Error)
		assert.WithinDuration(t, time.Now(), result.Time, time.Second,
			"empty string should return current time")
	})

	t.Run("now returns current time", func(t *testing.T) {
		result := parser.ParseTimestamp("now")
		assert.NoError(t, result.Error)
		assert.WithinDuration(t, time.Now(), result.Time, time.Second,
			"'now' should return current time")
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkParseAndStoreRelativeTime(b *testing.B) {
	db, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	blockRepo := storage.NewBlockRepo(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := parser.ParseTimestamp("2 hours ago")
		if result.Error != nil {
			b.Fatalf("parse error: %v", result.Error)
		}

		block := model.NewBlock("user-1", "benchmark-project", "", "", result.Time)
		if err := blockRepo.Create(block); err != nil {
			b.Fatalf("create error: %v", err)
		}
	}
}

func BenchmarkParseAndStoreAbsoluteTime(b *testing.B) {
	db, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	blockRepo := storage.NewBlockRepo(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := parser.ParseTimestamp("9:30am")
		if result.Error != nil {
			b.Fatalf("parse error: %v", result.Error)
		}

		block := model.NewBlock("user-1", "benchmark-project", "", "", result.Time)
		if err := blockRepo.Create(block); err != nil {
			b.Fatalf("create error: %v", err)
		}
	}
}
