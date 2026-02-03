package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Block Tests
// =============================================================================

func TestNewBlock(t *testing.T) {
	start := time.Now()
	block := NewBlock("owner1", "myproject", "", "Working on feature", start)

	assert.NotNil(t, block)
	assert.Equal(t, "owner1", block.OwnerKey)
	assert.Equal(t, "myproject", block.ProjectSID)
	assert.Equal(t, "Working on feature", block.Note)
	assert.Equal(t, start, block.TimestampStart)
	assert.True(t, block.TimestampEnd.IsZero())
}

func TestBlockSetGetKey(t *testing.T) {
	block := &Block{}
	block.SetKey("block:abc123")
	assert.Equal(t, "block:abc123", block.GetKey())
}

func TestBlockHasTag(t *testing.T) {
	block := &Block{
		Tags: []string{"urgent", "Bug", "IMPORTANT"},
	}

	// Case-insensitive matching
	assert.True(t, block.HasTag("urgent"))
	assert.True(t, block.HasTag("URGENT"))
	assert.True(t, block.HasTag("bug"))
	assert.True(t, block.HasTag("important"))
	assert.False(t, block.HasTag("feature"))
	assert.False(t, block.HasTag(""))

	// Empty tags
	emptyBlock := &Block{}
	assert.False(t, emptyBlock.HasTag("any"))
}

func TestBlockIsActive(t *testing.T) {
	// Active block (no end time)
	active := &Block{
		TimestampStart: time.Now(),
	}
	assert.True(t, active.IsActive())

	// Completed block
	completed := &Block{
		TimestampStart: time.Now().Add(-1 * time.Hour),
		TimestampEnd:   time.Now(),
	}
	assert.False(t, completed.IsActive())
}

func TestBlockDuration(t *testing.T) {
	t.Run("completed_block", func(t *testing.T) {
		start := time.Now().Add(-2 * time.Hour)
		end := time.Now()
		block := &Block{
			TimestampStart: start,
			TimestampEnd:   end,
		}

		duration := block.Duration()
		assert.InDelta(t, 2*time.Hour, duration, float64(time.Second))
	})

	t.Run("active_block", func(t *testing.T) {
		start := time.Now().Add(-30 * time.Minute)
		block := &Block{
			TimestampStart: start,
		}

		duration := block.Duration()
		assert.InDelta(t, 30*time.Minute, duration, float64(time.Second))
	})
}

func TestBlockDurationSeconds(t *testing.T) {
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	block := &Block{
		TimestampStart: start,
		TimestampEnd:   end,
	}

	seconds := block.DurationSeconds()
	assert.InDelta(t, 3600, seconds, 1)
}

func TestGenerateBlockKey(t *testing.T) {
	key := GenerateBlockKey("abc123")
	assert.Equal(t, "block:abc123", key)
}

// =============================================================================
// Project Tests
// =============================================================================

func TestNewProject(t *testing.T) {
	project := NewProject("myproject", "My Project", "#FF5733")

	assert.NotNil(t, project)
	assert.Equal(t, "project:myproject", project.Key)
	assert.Equal(t, "myproject", project.SID)
	assert.Equal(t, "My Project", project.DisplayName)
	assert.Equal(t, "#FF5733", project.Color)
}

func TestProjectSetGetKey(t *testing.T) {
	project := &Project{}
	project.SetKey("project:test")
	assert.Equal(t, "project:test", project.GetKey())
}

func TestGenerateProjectKey(t *testing.T) {
	key := GenerateProjectKey("myproject")
	assert.Equal(t, "project:myproject", key)
}

func TestValidateColor(t *testing.T) {
	tests := []struct {
		color string
		valid bool
	}{
		{"", true},
		{"#FF0000", true},
		{"#00FF00", true},
		{"#0000FF", true},
		{"#ff5733", true},
		{"#ABCDEF", true},
		{"FF0000", false},     // Missing #
		{"#FFF", false},       // Too short
		{"#FFFFFFF", false},   // Too long
		{"#GGGGGG", false},    // Invalid hex
		{"red", false},        // Named color
		{"#12345G", false},    // Invalid char
	}

	for _, tt := range tests {
		t.Run(tt.color, func(t *testing.T) {
			result := ValidateColor(tt.color)
			assert.Equal(t, tt.valid, result)
		})
	}
}

// =============================================================================
// ActiveBlock Tests
// =============================================================================

func TestNewActiveBlock(t *testing.T) {
	ab := NewActiveBlock()
	assert.NotNil(t, ab)
	assert.Equal(t, KeyActiveBlock, ab.Key)
	assert.Empty(t, ab.ActiveBlockKey)
	assert.Empty(t, ab.PreviousBlockKey)
}

func TestActiveBlockSetGetKey(t *testing.T) {
	ab := &ActiveBlock{}
	ab.SetKey("activeblock")
	assert.Equal(t, "activeblock", ab.GetKey())
}

func TestActiveBlockIsTracking(t *testing.T) {
	ab := NewActiveBlock()
	assert.False(t, ab.IsTracking())

	ab.ActiveBlockKey = "block:123"
	assert.True(t, ab.IsTracking())
}

func TestActiveBlockSetActive(t *testing.T) {
	ab := NewActiveBlock()

	// First activation
	ab.SetActive("block:1")
	assert.Equal(t, "block:1", ab.ActiveBlockKey)
	assert.Empty(t, ab.PreviousBlockKey)

	// Second activation (previous saved)
	ab.SetActive("block:2")
	assert.Equal(t, "block:2", ab.ActiveBlockKey)
	assert.Equal(t, "block:1", ab.PreviousBlockKey)
}

func TestActiveBlockClearActive(t *testing.T) {
	ab := NewActiveBlock()
	ab.ActiveBlockKey = "block:1"

	ab.ClearActive()
	assert.Empty(t, ab.ActiveBlockKey)
	assert.Equal(t, "block:1", ab.PreviousBlockKey)

	// Clearing again when already empty
	ab.ClearActive()
	assert.Empty(t, ab.ActiveBlockKey)
	assert.Equal(t, "block:1", ab.PreviousBlockKey)
}

// =============================================================================
// UndoState Tests
// =============================================================================

func TestNewUndoState(t *testing.T) {
	block := &Block{Key: "block:123", ProjectSID: "proj"}
	state := NewUndoState(UndoActionStop, "block:123", block)

	assert.NotNil(t, state)
	assert.Equal(t, KeyUndo, state.Key)
	assert.Equal(t, UndoActionStop, state.Action)
	assert.Equal(t, "block:123", state.BlockKey)
	assert.Equal(t, block, state.BlockSnapshot)
}

func TestUndoStateSetGetKey(t *testing.T) {
	state := &UndoState{}
	state.SetKey("undo")
	assert.Equal(t, "undo", state.GetKey())
}

func TestUndoActionConstants(t *testing.T) {
	assert.Equal(t, UndoAction("start"), UndoActionStart)
	assert.Equal(t, UndoAction("stop"), UndoActionStop)
	assert.Equal(t, UndoAction("delete"), UndoActionDelete)
}

// =============================================================================
// Model Interface Tests
// =============================================================================

func TestModelInterface(t *testing.T) {
	// Verify all types implement Model interface
	var _ Model = &Block{}
	var _ Model = &Project{}
	var _ Model = &ActiveBlock{}
	var _ Model = &UndoState{}
}

// =============================================================================
// Key Prefix Constants Tests
// =============================================================================

func TestKeyPrefixConstants(t *testing.T) {
	assert.Equal(t, "block", PrefixBlock)
	assert.Equal(t, "project", PrefixProject)
	assert.Equal(t, "activeblock", KeyActiveBlock)
	assert.Equal(t, "undo", KeyUndo)
}
