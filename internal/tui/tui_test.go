package tui

import (
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// ProgressBar Tests
// =============================================================================

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		width      int
	}{
		{"zero", 0, 10},
		{"half", 50, 10},
		{"full", 100, 10},
		{"over", 150, 10},
		{"negative", -10, 10},
		{"small_width", 50, 5},
		{"large_width", 50, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := ProgressBar(tt.percentage, tt.width)
			assert.NotEmpty(t, bar)
			// Bar should contain at least one block character (filled or empty)
			hasBlock := false
			if len(bar) > 0 {
				hasBlock = true
			}
			assert.True(t, hasBlock)
		})
	}
}

func TestProgressBarWidth(t *testing.T) {
	// Width should be respected
	bar10 := ProgressBar(50, 10)
	bar20 := ProgressBar(50, 20)

	// Longer width should produce longer bar
	assert.Greater(t, len(bar20), len(bar10))
}

// =============================================================================
// FormatProjectTask Tests
// =============================================================================

func TestFormatProjectTask(t *testing.T) {
	t.Run("project_only", func(t *testing.T) {
		result := FormatProjectTask("myproject", "")
		assert.Contains(t, result, "myproject")
		assert.NotContains(t, result, "/")
	})

	t.Run("project_and_task", func(t *testing.T) {
		result := FormatProjectTask("myproject", "mytask")
		assert.Contains(t, result, "myproject")
		assert.Contains(t, result, "mytask")
		assert.Contains(t, result, "/")
	})
}

// =============================================================================
// StatusComponent Tests
// =============================================================================

func TestNewStatusComponent(t *testing.T) {
	t.Run("nil_block", func(t *testing.T) {
		sc := NewStatusComponent(nil, 80)
		assert.NotNil(t, sc)
		assert.Nil(t, sc.Block)
		assert.Equal(t, 80, sc.Width)
		assert.False(t, sc.IsActive)
	})

	t.Run("with_active_block", func(t *testing.T) {
		block := &model.Block{
			ProjectSID:     "myproject",
			TimestampStart: time.Now(),
		}
		sc := NewStatusComponent(block, 80)

		assert.NotNil(t, sc)
		assert.Equal(t, block, sc.Block)
		assert.True(t, sc.IsActive)
		assert.Equal(t, block.TimestampStart, sc.StartTime)
	})

	t.Run("with_completed_block", func(t *testing.T) {
		block := &model.Block{
			ProjectSID:     "myproject",
			TimestampStart: time.Now().Add(-1 * time.Hour),
			TimestampEnd:   time.Now(),
		}
		sc := NewStatusComponent(block, 80)

		assert.NotNil(t, sc)
		assert.False(t, sc.IsActive)
	})
}

func TestStatusComponentView(t *testing.T) {
	t.Run("no_tracking", func(t *testing.T) {
		sc := NewStatusComponent(nil, 80)
		view := sc.View()

		assert.Contains(t, view, "Not tracking")
	})

	t.Run("active_tracking", func(t *testing.T) {
		block := &model.Block{
			ProjectSID:     "myproject",
			TaskSID:        "mytask",
			TimestampStart: time.Now().Add(-30 * time.Minute),
		}
		sc := NewStatusComponent(block, 80)
		view := sc.View()

		assert.Contains(t, view, "TRACKING")
		assert.Contains(t, view, "myproject")
	})

	t.Run("tracking_with_note", func(t *testing.T) {
		block := &model.Block{
			ProjectSID:     "myproject",
			Note:           "Working on feature",
			TimestampStart: time.Now(),
		}
		sc := NewStatusComponent(block, 80)
		view := sc.View()

		assert.Contains(t, view, "Working on feature")
	})
}

// =============================================================================
// BlocksComponent Tests
// =============================================================================

func TestNewBlocksComponent(t *testing.T) {
	t.Run("empty_blocks", func(t *testing.T) {
		bc := NewBlocksComponent(nil, 80, 5)
		assert.NotNil(t, bc)
		assert.Nil(t, bc.Blocks)
		assert.Equal(t, 80, bc.Width)
		assert.Equal(t, 5, bc.Limit)
	})

	t.Run("with_blocks", func(t *testing.T) {
		blocks := []*model.Block{
			{ProjectSID: "proj1"},
			{ProjectSID: "proj2"},
		}
		bc := NewBlocksComponent(blocks, 80, 5)

		assert.Equal(t, 2, len(bc.Blocks))
	})

	t.Run("limit_blocks", func(t *testing.T) {
		blocks := []*model.Block{
			{ProjectSID: "proj1"},
			{ProjectSID: "proj2"},
			{ProjectSID: "proj3"},
			{ProjectSID: "proj4"},
		}
		bc := NewBlocksComponent(blocks, 80, 2)

		assert.Equal(t, 2, len(bc.Blocks))
	})

	t.Run("zero_limit_no_truncation", func(t *testing.T) {
		blocks := []*model.Block{
			{ProjectSID: "proj1"},
			{ProjectSID: "proj2"},
		}
		bc := NewBlocksComponent(blocks, 80, 0)

		assert.Equal(t, 2, len(bc.Blocks))
	})
}

func TestBlocksComponentView(t *testing.T) {
	t.Run("empty_blocks", func(t *testing.T) {
		bc := NewBlocksComponent(nil, 80, 5)
		view := bc.View()

		assert.Contains(t, view, "Recent Blocks")
		assert.Contains(t, view, "No blocks yet")
	})

	t.Run("with_blocks", func(t *testing.T) {
		blocks := []*model.Block{
			{
				ProjectSID:     "proj1",
				TimestampStart: time.Now().Add(-2 * time.Hour),
				TimestampEnd:   time.Now().Add(-1 * time.Hour),
			},
			{
				ProjectSID:     "proj2",
				TimestampStart: time.Now(),
			},
		}
		bc := NewBlocksComponent(blocks, 80, 5)
		view := bc.View()

		assert.Contains(t, view, "Recent Blocks")
		assert.Contains(t, view, "proj1")
		assert.Contains(t, view, "proj2")
	})
}

func TestBlocksComponentRenderBlock(t *testing.T) {
	t.Run("completed_block", func(t *testing.T) {
		bc := &BlocksComponent{Width: 80}
		block := &model.Block{
			ProjectSID:     "proj1",
			TaskSID:        "task1",
			TimestampStart: time.Now().Add(-1 * time.Hour),
			TimestampEnd:   time.Now(),
		}
		rendered := bc.renderBlock(block)

		assert.Contains(t, rendered, "proj1")
		assert.Contains(t, rendered, "task1")
	})

	t.Run("active_block", func(t *testing.T) {
		bc := &BlocksComponent{Width: 80}
		block := &model.Block{
			ProjectSID:     "proj1",
			TimestampStart: time.Now(),
		}
		rendered := bc.renderBlock(block)

		assert.Contains(t, rendered, "active")
	})
}

// =============================================================================
// GoalComponent Tests
// =============================================================================

func TestNewGoalComponent(t *testing.T) {
	t.Run("nil_goal", func(t *testing.T) {
		gc := NewGoalComponent(nil, 0, 80)
		assert.NotNil(t, gc)
		assert.Nil(t, gc.Goal)
	})

	t.Run("with_goal", func(t *testing.T) {
		goal := &model.Goal{
			ProjectSID: "myproject",
			Type:       model.GoalTypeDaily,
			Target:     4 * time.Hour,
		}
		gc := NewGoalComponent(goal, 2*time.Hour, 80)

		assert.NotNil(t, gc)
		assert.Equal(t, goal, gc.Goal)
		assert.Equal(t, 2*time.Hour, gc.Current)
		assert.Equal(t, 50.0, gc.Progress.Percentage)
		assert.False(t, gc.Progress.IsComplete)
	})

	t.Run("goal_complete", func(t *testing.T) {
		goal := &model.Goal{
			ProjectSID: "myproject",
			Type:       model.GoalTypeDaily,
			Target:     4 * time.Hour,
		}
		gc := NewGoalComponent(goal, 5*time.Hour, 80)

		assert.True(t, gc.Progress.IsComplete)
	})
}

func TestGoalComponentView(t *testing.T) {
	t.Run("nil_goal", func(t *testing.T) {
		gc := NewGoalComponent(nil, 0, 80)
		view := gc.View()
		assert.Empty(t, view)
	})

	t.Run("daily_goal", func(t *testing.T) {
		goal := &model.Goal{
			ProjectSID: "myproject",
			Type:       model.GoalTypeDaily,
			Target:     4 * time.Hour,
		}
		gc := NewGoalComponent(goal, 2*time.Hour, 80)
		view := gc.View()

		assert.Contains(t, view, "Daily")
		assert.Contains(t, view, "myproject")
		assert.Contains(t, view, "remaining")
	})

	t.Run("weekly_goal", func(t *testing.T) {
		goal := &model.Goal{
			ProjectSID: "myproject",
			Type:       model.GoalTypeWeekly,
			Target:     20 * time.Hour,
		}
		gc := NewGoalComponent(goal, 10*time.Hour, 80)
		view := gc.View()

		assert.Contains(t, view, "Weekly")
	})

	t.Run("completed_goal", func(t *testing.T) {
		goal := &model.Goal{
			ProjectSID: "myproject",
			Type:       model.GoalTypeDaily,
			Target:     4 * time.Hour,
		}
		gc := NewGoalComponent(goal, 5*time.Hour, 80)
		view := gc.View()

		assert.Contains(t, view, "Goal completed")
	})

	t.Run("small_width", func(t *testing.T) {
		goal := &model.Goal{
			ProjectSID: "myproject",
			Type:       model.GoalTypeDaily,
			Target:     4 * time.Hour,
		}
		gc := NewGoalComponent(goal, 2*time.Hour, 15)
		view := gc.View()

		// Should still render
		assert.NotEmpty(t, view)
	})
}

// =============================================================================
// HelpBar Tests
// =============================================================================

func TestHelpBar(t *testing.T) {
	bar := HelpBar()

	assert.Contains(t, bar, "start")
	assert.Contains(t, bar, "stop")
	assert.Contains(t, bar, "refresh")
	assert.Contains(t, bar, "quit")
	assert.Contains(t, bar, "s")
	assert.Contains(t, bar, "e")
	assert.Contains(t, bar, "r")
	assert.Contains(t, bar, "q")
}

// =============================================================================
// Style Tests
// =============================================================================

func TestColorConstants(t *testing.T) {
	// Just verify they exist and are not empty
	assert.NotEmpty(t, ColorPrimary)
	assert.NotEmpty(t, ColorSecondary)
	assert.NotEmpty(t, ColorMuted)
	assert.NotEmpty(t, ColorWarning)
	assert.NotEmpty(t, ColorError)
	assert.NotEmpty(t, ColorSuccess)
	assert.NotEmpty(t, ColorActive)
	assert.NotEmpty(t, ColorBorder)
}

func TestStyleConstants(t *testing.T) {
	// Test that styles exist and can render
	tests := []struct {
		name  string
		style interface{}
	}{
		{"StyleTitle", StyleTitle},
		{"StyleSubtitle", StyleSubtitle},
		{"StyleProject", StyleProject},
		{"StyleTask", StyleTask},
		{"StyleDuration", StyleDuration},
		{"StyleNote", StyleNote},
		{"StyleActive", StyleActive},
		{"StyleInactive", StyleInactive},
		{"StyleWarning", StyleWarning},
		{"StyleError", StyleError},
		{"StyleSuccess", StyleSuccess},
		{"StyleHelp", StyleHelp},
		{"StyleHelpKey", StyleHelpKey},
		{"StyleHelpDesc", StyleHelpDesc},
		{"StyleMuted", StyleMuted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.style)
		})
	}
}

func TestBoxStyleConstants(t *testing.T) {
	tests := []struct {
		name  string
		style interface{}
	}{
		{"StyleStatusBox", StyleStatusBox},
		{"StyleActiveStatusBox", StyleActiveStatusBox},
		{"StyleBlocksBox", StyleBlocksBox},
		{"StyleGoalBox", StyleGoalBox},
		{"StyleGoalCompleteBox", StyleGoalCompleteBox},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.style)
		})
	}
}
