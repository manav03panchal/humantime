// Package integration provides integration tests for Humantime log feature.
// These tests verify the quick log functionality.
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
// Duration Parsing Tests
// =============================================================================

func TestLogFeature_ParseDuration(t *testing.T) {
	testCases := []struct {
		input    string
		expected time.Duration
		valid    bool
	}{
		// Hours
		{"2h", 2 * time.Hour, true},
		{"2hr", 2 * time.Hour, true},
		{"2hrs", 2 * time.Hour, true},
		{"2 hour", 2 * time.Hour, true},
		{"2 hours", 2 * time.Hour, true},

		// Minutes
		{"30m", 30 * time.Minute, true},
		{"30min", 30 * time.Minute, true},
		{"30mins", 30 * time.Minute, true},
		{"30 minute", 30 * time.Minute, true},
		{"30 minutes", 30 * time.Minute, true},

		// Seconds
		{"45s", 45 * time.Second, true},
		{"45sec", 45 * time.Second, true},
		{"45secs", 45 * time.Second, true},
		{"45 second", 45 * time.Second, true},
		{"45 seconds", 45 * time.Second, true},

		// Combined
		{"1h30m", 90 * time.Minute, true},
		{"2h15m", 135 * time.Minute, true},

		// Decimal hours
		{"1.5h", 90 * time.Minute, true},
		{"2.5 hours", 150 * time.Minute, true},
		{"0.5h", 30 * time.Minute, true},

		// Case insensitive
		{"2H", 2 * time.Hour, true},
		{"30M", 30 * time.Minute, true},
		{"2 HOURS", 2 * time.Hour, true},

		// Invalid
		{"invalid", 0, false},
		{"", 0, false},
		{"abc123", 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := parser.ParseDuration(tc.input)
			assert.Equal(t, tc.valid, result.Valid, "valid mismatch for %s", tc.input)
			if tc.valid {
				assert.Equal(t, tc.expected, result.Duration, "duration mismatch for %s", tc.input)
			}
		})
	}
}

func TestLogFeature_IsDurationLike(t *testing.T) {
	trueCases := []string{
		"2h", "30m", "1.5h", "2 hours", "30 minutes",
		"2hr", "30min", "45s", "2.5h",
	}

	falseCases := []string{
		"project", "on", "with", "yesterday",
		"", "abc", "clientwork",
	}

	for _, tc := range trueCases {
		t.Run("true_"+tc, func(t *testing.T) {
			assert.True(t, parser.IsDurationLike(tc), "%s should be duration-like", tc)
		})
	}

	for _, tc := range falseCases {
		t.Run("false_"+tc, func(t *testing.T) {
			assert.False(t, parser.IsDurationLike(tc), "%s should not be duration-like", tc)
		})
	}
}

// =============================================================================
// Log Command Integration Tests
// =============================================================================

func TestLogFeature_CreateCompletedBlock(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err)

	t.Run("creates a completed block with correct duration", func(t *testing.T) {
		// Simulate what log command does
		duration := 2 * time.Hour
		endTime := time.Now()
		startTime := endTime.Add(-duration)

		// Auto-create project
		_, _, err := projectRepo.GetOrCreate("logtest", "logtest")
		require.NoError(t, err)

		// Create block
		block := model.NewBlock(config.UserKey, "logtest", "", "", startTime)
		block.TimestampEnd = endTime

		err = blockRepo.Create(block)
		require.NoError(t, err)

		// Verify block
		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		assert.Equal(t, "logtest", retrieved.ProjectSID)
		assert.False(t, retrieved.IsActive(), "logged block should be completed")
		assert.False(t, retrieved.TimestampEnd.IsZero())

		// Check duration is approximately correct (within a few seconds)
		actualDuration := retrieved.Duration()
		assert.InDelta(t, duration.Seconds(), actualDuration.Seconds(), 1.0)
	})

	t.Run("creates block with note", func(t *testing.T) {
		duration := 30 * time.Minute
		endTime := time.Now()
		startTime := endTime.Add(-duration)

		_, _, err := projectRepo.GetOrCreate("notelogtest", "notelogtest")
		require.NoError(t, err)

		block := model.NewBlock(config.UserKey, "notelogtest", "", "fixed important bug", startTime)
		block.TimestampEnd = endTime

		err = blockRepo.Create(block)
		require.NoError(t, err)

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, "fixed important bug", retrieved.Note)
	})

	t.Run("creates block with tags", func(t *testing.T) {
		duration := 1 * time.Hour
		endTime := time.Now()
		startTime := endTime.Add(-duration)

		_, _, err := projectRepo.GetOrCreate("taglogtest", "taglogtest")
		require.NoError(t, err)

		block := model.NewBlock(config.UserKey, "taglogtest", "", "", startTime)
		block.TimestampEnd = endTime
		block.Tags = []string{"billable", "meeting"}

		err = blockRepo.Create(block)
		require.NoError(t, err)

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, []string{"billable", "meeting"}, retrieved.Tags)
	})

	t.Run("creates block for past time", func(t *testing.T) {
		duration := 3 * time.Hour
		// End time is yesterday
		endTime := time.Now().Add(-24 * time.Hour)
		startTime := endTime.Add(-duration)

		_, _, err := projectRepo.GetOrCreate("pastlogtest", "pastlogtest")
		require.NoError(t, err)

		block := model.NewBlock(config.UserKey, "pastlogtest", "", "", startTime)
		block.TimestampEnd = endTime

		err = blockRepo.Create(block)
		require.NoError(t, err)

		retrieved, err := blockRepo.Get(block.Key)
		require.NoError(t, err)

		// Check that the block is from yesterday
		yesterday := time.Now().Add(-24 * time.Hour)
		assert.True(t, retrieved.TimestampEnd.Before(time.Now().Add(-23*time.Hour)),
			"block end time should be approximately yesterday")
		assert.True(t, retrieved.TimestampEnd.After(yesterday.Add(-1*time.Hour)),
			"block end time should be approximately yesterday")
	})
}

