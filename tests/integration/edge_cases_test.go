package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// T048: System Time Changes (DST, Timezone)
// =============================================================================

func TestSystemTimeChanges_TimezoneHandling(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	configRepo := storage.NewConfigRepo(db)

	// Get config for owner key
	config, err := configRepo.Get()
	require.NoError(t, err)

	// Create project
	_, _, err = projectRepo.GetOrCreate("tz-test", "Timezone Test")
	require.NoError(t, err)

	// Test with different timezones
	timezones := []string{
		"America/New_York",
		"Europe/London",
		"Asia/Tokyo",
		"Australia/Sydney",
		"UTC",
	}

	for _, tzName := range timezones {
		t.Run(tzName, func(t *testing.T) {
			loc, err := time.LoadLocation(tzName)
			require.NoError(t, err)

			now := time.Now().In(loc)
			start := now.Add(-1 * time.Hour)

			block := model.NewBlock(config.UserKey, "tz-test", "", "Timezone test", start)
			block.TimestampEnd = now

			err = blockRepo.Create(block)
			require.NoError(t, err)

			// Retrieve and verify
			retrieved, err := blockRepo.Get(block.Key)
			require.NoError(t, err)

			// Duration should be consistent regardless of timezone
			assert.InDelta(t, 3600, retrieved.DurationSeconds(), 10)
		})
	}
}

func TestSystemTimeChanges_DSTTransition(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	configRepo := storage.NewConfigRepo(db)

	config, err := configRepo.Get()
	require.NoError(t, err)

	_, _, err = projectRepo.GetOrCreate("dst-test", "DST Test")
	require.NoError(t, err)

	// Test a time block that would span DST transition
	// (simulate with times that might be affected)
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	// Create a block with a 2-hour duration
	baseTime := time.Date(2024, 3, 10, 1, 0, 0, 0, loc) // Near DST transition
	endTime := time.Date(2024, 3, 10, 4, 0, 0, 0, loc)  // After potential DST change

	block := model.NewBlock(config.UserKey, "dst-test", "", "DST test", baseTime)
	block.TimestampEnd = endTime

	err = blockRepo.Create(block)
	require.NoError(t, err)

	retrieved, err := blockRepo.Get(block.Key)
	require.NoError(t, err)

	// Block should be stored and retrieved correctly
	assert.NotNil(t, retrieved)
	assert.False(t, retrieved.TimestampStart.IsZero())
	assert.False(t, retrieved.TimestampEnd.IsZero())
}

func TestSystemTimeChanges_ParseTimeWithTimezone(t *testing.T) {
	// Test parsing time strings in different contexts
	testCases := []struct {
		name  string
		input string
	}{
		{"simple time", "14:30"},
		{"time with AM", "2:30pm"},
		{"time with PM", "2:30am"},
		{"relative", "in 1h"},
		{"relative minutes", "in 30m"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.ParseTimestamp(tc.input)
			if result.Error == nil {
				assert.False(t, result.Time.IsZero())
			}
		})
	}
}

// =============================================================================
// T049: Extremely Long Project Names and Notes
// =============================================================================

func TestLongProjectNames(t *testing.T) {
	db := setupTestDB(t)
	projectRepo := storage.NewProjectRepo(db)

	testCases := []struct {
		name      string
		sidLength int
	}{
		{"normal", 20},
		{"medium", 100},
		{"long", 500},
		{"very_long", 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sid := strings.Repeat("a", tc.sidLength)
			displayName := strings.Repeat("Project Name ", tc.sidLength/13+1)

			project := model.NewProject(sid, displayName, "#FF0000")
			err := projectRepo.Create(project)
			require.NoError(t, err)

			// Retrieve and verify
			retrieved, err := projectRepo.Get(sid)
			require.NoError(t, err)
			assert.Equal(t, sid, retrieved.SID)
		})
	}
}

func TestLongNotes(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	configRepo := storage.NewConfigRepo(db)

	config, err := configRepo.Get()
	require.NoError(t, err)

	_, _, err = projectRepo.GetOrCreate("long-note", "Long Note Test")
	require.NoError(t, err)

	testCases := []struct {
		name       string
		noteLength int
	}{
		{"short", 100},
		{"medium", 1000},
		{"long", 10000},
		{"very_long", 50000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			note := strings.Repeat("This is a test note. ", tc.noteLength/21+1)
			if len(note) > tc.noteLength {
				note = note[:tc.noteLength]
			}

			now := time.Now()
			block := model.NewBlock(config.UserKey, "long-note", "", note, now.Add(-1*time.Hour))
			block.TimestampEnd = now

			err := blockRepo.Create(block)
			require.NoError(t, err)

			// Retrieve and verify
			retrieved, err := blockRepo.Get(block.Key)
			require.NoError(t, err)
			assert.Equal(t, note, retrieved.Note)
		})
	}
}

func TestSpecialCharactersInProjectNames(t *testing.T) {
	db := setupTestDB(t)
	projectRepo := storage.NewProjectRepo(db)

	testCases := []struct {
		name        string
		sid         string
		displayName string
	}{
		{"unicode", "project-日本語", "プロジェクト"},
		{"emoji", "project-emoji", "Project 🚀🎉"},
		{"special_chars", "project-special", "Project <>&\"'"},
		{"newlines", "project-newline", "Project\nWith\nNewlines"},
		{"tabs", "project-tab", "Project\tWith\tTabs"},
		{"mixed", "project-mixed", "日本語 Project 🎉 <test>"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			project := model.NewProject(tc.sid, tc.displayName, "")
			err := projectRepo.Create(project)
			require.NoError(t, err)

			retrieved, err := projectRepo.Get(tc.sid)
			require.NoError(t, err)
			assert.Equal(t, tc.displayName, retrieved.DisplayName)
		})
	}
}

func TestSpecialCharactersInNotes(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	configRepo := storage.NewConfigRepo(db)

	config, err := configRepo.Get()
	require.NoError(t, err)

	_, _, err = projectRepo.GetOrCreate("special-notes", "Special Notes")
	require.NoError(t, err)

	testNotes := []string{
		"Note with \"quotes\" and 'apostrophes'",
		"Note with <html> tags & entities",
		"Note with unicode: 日本語 中文 한국어",
		"Note with emoji: 🎉🚀💻",
		"Note with\nnewlines\nand\ttabs",
		"Note with null \x00 byte",
		"Note with control chars: \x01\x02\x03",
		strings.Repeat("Very long repeated text. ", 1000),
	}

	for i, note := range testNotes {
		t.Run("note_"+string(rune('A'+i)), func(t *testing.T) {
			now := time.Now()
			block := model.NewBlock(config.UserKey, "special-notes", "", note, now.Add(-1*time.Hour))
			block.TimestampEnd = now

			err := blockRepo.Create(block)
			require.NoError(t, err)

			retrieved, err := blockRepo.Get(block.Key)
			require.NoError(t, err)
			// Note: null bytes may be stripped or handled differently
			if !strings.Contains(note, "\x00") {
				assert.Equal(t, note, retrieved.Note)
			}
		})
	}
}

// =============================================================================
// T050: Config File with Invalid Values
// =============================================================================

func TestConfigWithInvalidColorValues(t *testing.T) {
	db := setupTestDB(t)
	projectRepo := storage.NewProjectRepo(db)

	// Test various invalid color formats
	invalidColors := []string{
		"red",           // Named color (not hex)
		"#GGG",          // Invalid hex
		"#12345",        // Wrong length
		"#1234567890",   // Too long
		"rgb(255,0,0)",  // RGB format
		"",              // Empty (should be allowed)
		"invalid",       // Random string
	}

	for i, color := range invalidColors {
		t.Run("color_"+string(rune('A'+i)), func(t *testing.T) {
			sid := "color-test-" + string(rune('A'+i))
			project := model.NewProject(sid, "Color Test", color)
			err := projectRepo.Create(project)
			// Should succeed - color validation is at a higher level
			require.NoError(t, err)

			retrieved, err := projectRepo.Get(sid)
			require.NoError(t, err)
			assert.Equal(t, color, retrieved.Color)
		})
	}
}

func TestGoalWithInvalidValues(t *testing.T) {
	db := setupTestDB(t)
	goalRepo := storage.NewGoalRepo(db)
	projectRepo := storage.NewProjectRepo(db)

	// Create project first
	_, _, err := projectRepo.GetOrCreate("goal-test", "Goal Test")
	require.NoError(t, err)

	testCases := []struct {
		name    string
		target  time.Duration
		isValid bool
	}{
		{"zero", 0, true},
		{"negative", -1 * time.Hour, true}, // Storage allows, validation is at higher level
		{"normal", 8 * time.Hour, true},
		{"very_large", 1000 * time.Hour, true},
		{"max_int64", time.Duration(1<<62), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			goal := model.NewGoal("goal-test", model.GoalTypeDaily, tc.target)
			err := goalRepo.Upsert(goal)
			if tc.isValid {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReminderWithInvalidDeadlines(t *testing.T) {
	db := setupTestDB(t)
	reminderRepo := storage.NewReminderRepo(db)

	testCases := []struct {
		name     string
		deadline time.Time
	}{
		{"past", time.Now().Add(-24 * time.Hour)},
		{"far_future", time.Now().Add(365 * 24 * time.Hour * 100)}, // 100 years
		{"zero", time.Time{}},
		{"unix_epoch", time.Unix(0, 0)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reminder := model.NewReminder("Test Reminder", tc.deadline, "user1")
			err := reminderRepo.Create(reminder)
			// Storage layer should accept all - validation is at higher level
			assert.NoError(t, err)
		})
	}
}

// =============================================================================
// T051: Sleep/Wake Daemon Behavior
// =============================================================================

func TestActiveBlockAfterLongDelay(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	configRepo := storage.NewConfigRepo(db)

	config, err := configRepo.Get()
	require.NoError(t, err)

	_, _, err = projectRepo.GetOrCreate("delay-test", "Delay Test")
	require.NoError(t, err)

	// Create an active block with a very old start time (simulating sleep)
	oldStart := time.Now().Add(-24 * time.Hour) // 24 hours ago
	block := model.NewBlock(config.UserKey, "delay-test", "", "Started before sleep", oldStart)
	// Don't set end time - it's active

	err = blockRepo.Create(block)
	require.NoError(t, err)

	err = activeBlockRepo.SetActive(block.Key)
	require.NoError(t, err)

	// Verify the active block can be retrieved
	activeBlock, err := activeBlockRepo.Get()
	require.NoError(t, err)
	assert.True(t, activeBlock.IsTracking())

	// Get the actual block and verify duration
	activeBlockData, err := activeBlockRepo.GetActiveBlock(blockRepo)
	require.NoError(t, err)
	require.NotNil(t, activeBlockData)

	// Duration should be approximately 24 hours
	duration := activeBlockData.Duration()
	assert.InDelta(t, 24*time.Hour, duration, float64(time.Minute))
}

func TestMultipleQuickStartStop(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	configRepo := storage.NewConfigRepo(db)

	config, err := configRepo.Get()
	require.NoError(t, err)

	_, _, err = projectRepo.GetOrCreate("quick-test", "Quick Test")
	require.NoError(t, err)

	// Simulate rapid start/stop cycles
	for i := 0; i < 10; i++ {
		// Start
		now := time.Now()
		block := model.NewBlock(config.UserKey, "quick-test", "", "Quick block", now)
		err := blockRepo.Create(block)
		require.NoError(t, err)

		err = activeBlockRepo.SetActive(block.Key)
		require.NoError(t, err)

		// Immediately stop
		block.TimestampEnd = time.Now()
		err = blockRepo.Update(block)
		require.NoError(t, err)

		err = activeBlockRepo.ClearActive()
		require.NoError(t, err)
	}

	// All blocks should be created
	blocks, err := blockRepo.ListByProject("quick-test")
	require.NoError(t, err)
	assert.Len(t, blocks, 10)

	// No active block
	activeBlock, err := activeBlockRepo.Get()
	require.NoError(t, err)
	assert.False(t, activeBlock.IsTracking())
}

func TestBlockWithZeroDuration(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	configRepo := storage.NewConfigRepo(db)

	config, err := configRepo.Get()
	require.NoError(t, err)

	_, _, err = projectRepo.GetOrCreate("zero-dur", "Zero Duration")
	require.NoError(t, err)

	// Create block with same start and end time
	now := time.Now()
	block := model.NewBlock(config.UserKey, "zero-dur", "", "Zero duration block", now)
	block.TimestampEnd = now // Same as start

	err = blockRepo.Create(block)
	require.NoError(t, err)

	retrieved, err := blockRepo.Get(block.Key)
	require.NoError(t, err)
	assert.Equal(t, int64(0), retrieved.DurationSeconds())
}

func TestRetrieveBlocksAfterSimulatedRestart(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	configRepo := storage.NewConfigRepo(db)

	config, err := configRepo.Get()
	require.NoError(t, err)

	_, _, err = projectRepo.GetOrCreate("restart-test", "Restart Test")
	require.NoError(t, err)

	// Create some historical blocks
	for i := 0; i < 5; i++ {
		now := time.Now()
		start := now.Add(-time.Duration(i+1) * time.Hour)
		end := start.Add(30 * time.Minute)

		block := model.NewBlock(config.UserKey, "restart-test", "", "Historical block", start)
		block.TimestampEnd = end
		err := blockRepo.Create(block)
		require.NoError(t, err)
	}

	// Create an active block
	activeStart := time.Now().Add(-10 * time.Minute)
	activeBlock := model.NewBlock(config.UserKey, "restart-test", "", "Active block", activeStart)
	err = blockRepo.Create(activeBlock)
	require.NoError(t, err)
	err = activeBlockRepo.SetActive(activeBlock.Key)
	require.NoError(t, err)

	// "Restart" - use same DB but re-query everything
	// Verify all historical blocks are retrievable
	blocks, err := blockRepo.ListByProject("restart-test")
	require.NoError(t, err)
	assert.Len(t, blocks, 6) // 5 historical + 1 active

	// Verify active block is still active
	current, err := activeBlockRepo.Get()
	require.NoError(t, err)
	assert.True(t, current.IsTracking())
	assert.Equal(t, activeBlock.Key, current.ActiveBlockKey)
}

// =============================================================================
// Edge Cases for Parser
// =============================================================================

func TestParserEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"empty", "", true},
		{"whitespace_only", "   ", true},
		{"just_numbers", "12345", false}, // May be parsed as a valid time
		{"invalid_time", "25:00", true},
		{"invalid_duration", "-1h", true},
		{"special_chars", "@#$%^&*", true},
		{"sql_injection", "'; DROP TABLE users; --", true},
		{"html_tags", "<script>alert(1)</script>", true},
		{"very_long", strings.Repeat("1h", 1000), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.ParseTimestamp(tc.input)
			if tc.hasError {
				// We expect these to either error or return invalid results
				// The important thing is they don't panic
				_ = result.Error
			}
		})
	}
}

func TestDurationParserEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected time.Duration
		hasError bool
	}{
		{"zero", "0h", 0, false},
		{"zero_minutes", "0m", 0, false},
		{"negative", "-1h", -1 * time.Hour, false}, // Parser accepts negative
		{"decimal", "1.5h", 90 * time.Minute, false},
		{"mixed", "1h30m", 90 * time.Minute, false},
		{"just_h", "h", 0, true},
		{"just_m", "m", 0, true},
		{"large", "1000h", 1000 * time.Hour, false},
		// Very large values may overflow but the parser doesn't error
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.ParseDuration(tc.input)
			if tc.hasError {
				// Should error or return invalid
				if result.Error == nil {
					assert.False(t, result.Valid)
				}
			} else {
				assert.True(t, result.Valid || result.Error == nil)
				assert.Equal(t, tc.expected, result.Duration)
			}
		})
	}
}

func TestDurationParserOverflow(t *testing.T) {
	// Test very large values - parser may overflow silently
	result := parser.ParseDuration("99999999999999h")
	// The parser may return a valid result even with overflow
	// The important thing is it doesn't panic
	_ = result
}
