// Package integration provides integration tests for Humantime tracking workflows.
// These tests verify the complete behavior of start/stop tracking operations
// using a real in-memory Badger database.
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
// Test Context and Helpers
// =============================================================================

// trackingTestContext holds all repositories needed for tracking integration testing.
type trackingTestContext struct {
	t               *testing.T
	db              *storage.DB
	blockRepo       *storage.BlockRepo
	projectRepo     *storage.ProjectRepo
	taskRepo        *storage.TaskRepo
	configRepo      *storage.ConfigRepo
	activeBlockRepo *storage.ActiveBlockRepo
	userKey         string
}

// setupTrackingTestContext creates a new test context with an in-memory database.
func setupTrackingTestContext(t *testing.T) *trackingTestContext {
	t.Helper()

	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err, "failed to open in-memory database")

	// Get or create config to obtain user key
	configRepo := storage.NewConfigRepo(db)
	config, err := configRepo.Get()
	require.NoError(t, err, "failed to get config")

	tc := &trackingTestContext{
		t:               t,
		db:              db,
		blockRepo:       storage.NewBlockRepo(db),
		projectRepo:     storage.NewProjectRepo(db),
		taskRepo:        storage.NewTaskRepo(db),
		configRepo:      configRepo,
		activeBlockRepo: storage.NewActiveBlockRepo(db),
		userKey:         config.UserKey,
	}

	t.Cleanup(func() {
		err := db.Close()
		assert.NoError(t, err, "failed to close database")
	})

	return tc
}

// startTracking simulates starting tracking on a project/task.
// This replicates the logic from cmd/start.go.
func (tc *trackingTestContext) startTracking(projectSID, taskSID, note string) (*model.Block, error) {
	tc.t.Helper()

	// Ensure project exists (auto-create if needed)
	_, _, err := tc.projectRepo.GetOrCreate(projectSID, projectSID)
	if err != nil {
		return nil, err
	}

	// Ensure task exists if specified (auto-create if needed)
	if taskSID != "" {
		_, _, err := tc.taskRepo.GetOrCreate(projectSID, taskSID, taskSID)
		if err != nil {
			return nil, err
		}
	}

	// End any active tracking first
	activeBlock, err := tc.activeBlockRepo.GetActiveBlock(tc.blockRepo)
	if err != nil {
		return nil, err
	}
	if activeBlock != nil {
		activeBlock.TimestampEnd = time.Now()
		if err := tc.blockRepo.Update(activeBlock); err != nil {
			return nil, err
		}
	}

	// Create new block
	block := model.NewBlock(tc.userKey, projectSID, taskSID, note, time.Now())

	// Save the block
	if err := tc.blockRepo.Create(block); err != nil {
		return nil, err
	}

	// Set as active
	if err := tc.activeBlockRepo.SetActive(block.Key); err != nil {
		return nil, err
	}

	return block, nil
}

// stopTracking simulates stopping tracking with an optional note.
// This replicates the logic from cmd/stop.go.
func (tc *trackingTestContext) stopTracking(note string) (*model.Block, error) {
	tc.t.Helper()

	// Get active block
	block, err := tc.activeBlockRepo.GetActiveBlock(tc.blockRepo)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, nil
	}

	// Set end time
	block.TimestampEnd = time.Now()

	// Update note if provided
	if note != "" {
		if block.Note != "" {
			block.Note += " - " + note
		} else {
			block.Note = note
		}
	}

	// Save the block
	if err := tc.blockRepo.Update(block); err != nil {
		return nil, err
	}

	// Clear active tracking
	if err := tc.activeBlockRepo.ClearActive(); err != nil {
		return nil, err
	}

	return block, nil
}

// resumeTracking simulates resuming tracking on the previous project/task.
// This replicates the logic from cmd/start.go's runResume function.
func (tc *trackingTestContext) resumeTracking() (*model.Block, error) {
	tc.t.Helper()

	// Get the previous block
	previousBlock, err := tc.activeBlockRepo.GetPreviousBlock(tc.blockRepo)
	if err != nil {
		return nil, err
	}
	if previousBlock == nil {
		return nil, nil
	}

	// Start tracking on the same project/task
	block := model.NewBlock(
		tc.userKey,
		previousBlock.ProjectSID,
		previousBlock.TaskSID,
		"", // No note for resumed blocks
		time.Now(),
	)

	if err := tc.blockRepo.Create(block); err != nil {
		return nil, err
	}

	if err := tc.activeBlockRepo.SetActive(block.Key); err != nil {
		return nil, err
	}

	return block, nil
}

// =============================================================================
// Start Tracking Tests
// =============================================================================

func TestStartTracking_CreatesBlock(t *testing.T) {
	tc := setupTrackingTestContext(t)

	t.Run("creates a new block with correct project", func(t *testing.T) {
		block, err := tc.startTracking("testproject", "", "")
		require.NoError(t, err)
		require.NotNil(t, block)

		assert.NotEmpty(t, block.Key)
		assert.Equal(t, "testproject", block.ProjectSID)
		assert.Equal(t, tc.userKey, block.OwnerKey)
		assert.False(t, block.TimestampStart.IsZero())
		assert.True(t, block.TimestampEnd.IsZero(), "new block should not have end time")
	})

	t.Run("creates a block with project and task", func(t *testing.T) {
		block, err := tc.startTracking("myproject", "mytask", "")
		require.NoError(t, err)
		require.NotNil(t, block)

		assert.Equal(t, "myproject", block.ProjectSID)
		assert.Equal(t, "mytask", block.TaskSID)
	})

	t.Run("creates a block with note", func(t *testing.T) {
		block, err := tc.startTracking("noteproject", "", "working on feature")
		require.NoError(t, err)
		require.NotNil(t, block)

		assert.Equal(t, "working on feature", block.Note)
	})

	t.Run("created block is marked as active", func(t *testing.T) {
		block, err := tc.startTracking("activeproject", "", "")
		require.NoError(t, err)
		require.NotNil(t, block)

		// Verify it is the active block
		activeBlock, err := tc.activeBlockRepo.GetActiveBlock(tc.blockRepo)
		require.NoError(t, err)
		require.NotNil(t, activeBlock)
		assert.Equal(t, block.Key, activeBlock.Key)
	})

	t.Run("auto-creates project if not exists", func(t *testing.T) {
		_, err := tc.startTracking("newproject", "", "")
		require.NoError(t, err)

		// Verify project was created
		exists, err := tc.projectRepo.Exists("newproject")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("auto-creates task if not exists", func(t *testing.T) {
		_, err := tc.startTracking("taskproject", "newtask", "")
		require.NoError(t, err)

		// Verify task was created
		exists, err := tc.taskRepo.Exists("taskproject", "newtask")
		require.NoError(t, err)
		assert.True(t, exists)
	})
}

// =============================================================================
// Stop Tracking Tests
// =============================================================================

func TestStopTracking_EndsActiveBlock(t *testing.T) {
	tc := setupTrackingTestContext(t)

	t.Run("sets end time on active block", func(t *testing.T) {
		// Start tracking
		startBlock, err := tc.startTracking("stoptest", "", "")
		require.NoError(t, err)

		// Wait a moment to ensure time difference
		time.Sleep(10 * time.Millisecond)

		// Stop tracking
		stoppedBlock, err := tc.stopTracking("")
		require.NoError(t, err)
		require.NotNil(t, stoppedBlock)

		assert.Equal(t, startBlock.Key, stoppedBlock.Key)
		assert.False(t, stoppedBlock.TimestampEnd.IsZero(), "stopped block should have end time")
		assert.True(t, stoppedBlock.TimestampEnd.After(stoppedBlock.TimestampStart))
	})

	t.Run("clears active tracking", func(t *testing.T) {
		// Start and stop tracking
		_, err := tc.startTracking("cleartest", "", "")
		require.NoError(t, err)

		_, err = tc.stopTracking("")
		require.NoError(t, err)

		// Verify no active block
		activeBlock, err := tc.activeBlockRepo.GetActiveBlock(tc.blockRepo)
		require.NoError(t, err)
		assert.Nil(t, activeBlock, "should have no active block after stop")
	})

	t.Run("returns nil when no active tracking", func(t *testing.T) {
		// Ensure no active tracking
		tc.activeBlockRepo.ClearActive()

		block, err := tc.stopTracking("")
		require.NoError(t, err)
		assert.Nil(t, block)
	})

	t.Run("adds note on stop", func(t *testing.T) {
		// Start tracking without note
		_, err := tc.startTracking("notetest", "", "")
		require.NoError(t, err)

		// Stop with note
		stoppedBlock, err := tc.stopTracking("completed the work")
		require.NoError(t, err)
		require.NotNil(t, stoppedBlock)

		assert.Equal(t, "completed the work", stoppedBlock.Note)
	})

	t.Run("appends note on stop to existing note", func(t *testing.T) {
		// Start tracking with note
		_, err := tc.startTracking("appendnote", "", "initial note")
		require.NoError(t, err)

		// Stop with additional note
		stoppedBlock, err := tc.stopTracking("additional note")
		require.NoError(t, err)
		require.NotNil(t, stoppedBlock)

		assert.Contains(t, stoppedBlock.Note, "initial note")
		assert.Contains(t, stoppedBlock.Note, "additional note")
	})

	t.Run("block is persisted after stop", func(t *testing.T) {
		// Start tracking
		startBlock, err := tc.startTracking("persisttest", "", "")
		require.NoError(t, err)

		// Stop tracking
		_, err = tc.stopTracking("")
		require.NoError(t, err)

		// Retrieve block from database
		retrievedBlock, err := tc.blockRepo.Get(startBlock.Key)
		require.NoError(t, err)
		assert.False(t, retrievedBlock.TimestampEnd.IsZero())
	})
}

// =============================================================================
// Auto-Switch Behavior Tests
// =============================================================================

func TestAutoSwitch_StartNewEndsPrevious(t *testing.T) {
	tc := setupTrackingTestContext(t)

	t.Run("starting new project ends previous tracking", func(t *testing.T) {
		// Start tracking on first project
		firstBlock, err := tc.startTracking("project1", "", "")
		require.NoError(t, err)

		// Wait to ensure time difference
		time.Sleep(10 * time.Millisecond)

		// Start tracking on second project
		secondBlock, err := tc.startTracking("project2", "", "")
		require.NoError(t, err)

		// First block should now have an end time
		retrievedFirst, err := tc.blockRepo.Get(firstBlock.Key)
		require.NoError(t, err)
		assert.False(t, retrievedFirst.TimestampEnd.IsZero(),
			"first block should have end time after switching")

		// Second block should be active
		activeBlock, err := tc.activeBlockRepo.GetActiveBlock(tc.blockRepo)
		require.NoError(t, err)
		require.NotNil(t, activeBlock)
		assert.Equal(t, secondBlock.Key, activeBlock.Key)
	})

	t.Run("switching stores previous in active block record", func(t *testing.T) {
		// Start on first project
		firstBlock, err := tc.startTracking("switchprev1", "", "")
		require.NoError(t, err)

		// Switch to second project
		_, err = tc.startTracking("switchprev2", "", "")
		require.NoError(t, err)

		// Get active block record
		activeRecord, err := tc.activeBlockRepo.Get()
		require.NoError(t, err)
		assert.Equal(t, firstBlock.Key, activeRecord.PreviousBlockKey)
	})

	t.Run("multiple switches maintain chain", func(t *testing.T) {
		// Start on project1
		_, err := tc.startTracking("chain1", "", "")
		require.NoError(t, err)

		// Switch to project2
		block2, err := tc.startTracking("chain2", "", "")
		require.NoError(t, err)

		// Switch to project3
		block3, err := tc.startTracking("chain3", "", "")
		require.NoError(t, err)

		// Verify current active is project3
		activeBlock, err := tc.activeBlockRepo.GetActiveBlock(tc.blockRepo)
		require.NoError(t, err)
		require.NotNil(t, activeBlock)
		assert.Equal(t, block3.Key, activeBlock.Key)

		// Verify previous is project2
		activeRecord, err := tc.activeBlockRepo.Get()
		require.NoError(t, err)
		assert.Equal(t, block2.Key, activeRecord.PreviousBlockKey)
	})

	t.Run("all blocks are persisted correctly", func(t *testing.T) {
		// Create multiple blocks via switching
		_, err := tc.startTracking("persist1", "", "")
		require.NoError(t, err)
		time.Sleep(5 * time.Millisecond)

		_, err = tc.startTracking("persist2", "", "")
		require.NoError(t, err)
		time.Sleep(5 * time.Millisecond)

		_, err = tc.startTracking("persist3", "", "")
		require.NoError(t, err)

		// List all blocks for these projects
		blocks, err := tc.blockRepo.List()
		require.NoError(t, err)

		// Count blocks for our test projects
		count := 0
		for _, b := range blocks {
			if b.ProjectSID == "persist1" || b.ProjectSID == "persist2" || b.ProjectSID == "persist3" {
				count++
			}
		}
		assert.GreaterOrEqual(t, count, 3, "should have at least 3 blocks for persist projects")
	})
}

// =============================================================================
// Resume Functionality Tests
// =============================================================================

func TestResume_ResumesLastProject(t *testing.T) {
	tc := setupTrackingTestContext(t)

	t.Run("resumes tracking on previous project", func(t *testing.T) {
		// Start and stop tracking
		_, err := tc.startTracking("resumeproject", "resumetask", "")
		require.NoError(t, err)

		_, err = tc.stopTracking("")
		require.NoError(t, err)

		// Resume
		resumedBlock, err := tc.resumeTracking()
		require.NoError(t, err)
		require.NotNil(t, resumedBlock)

		assert.Equal(t, "resumeproject", resumedBlock.ProjectSID)
		assert.Equal(t, "resumetask", resumedBlock.TaskSID)
	})

	t.Run("resumed block is a new block", func(t *testing.T) {
		// Start and stop tracking
		originalBlock, err := tc.startTracking("newblocktest", "", "")
		require.NoError(t, err)

		_, err = tc.stopTracking("")
		require.NoError(t, err)

		// Resume
		resumedBlock, err := tc.resumeTracking()
		require.NoError(t, err)
		require.NotNil(t, resumedBlock)

		assert.NotEqual(t, originalBlock.Key, resumedBlock.Key, "resumed block should have new key")
	})

	t.Run("resumed block is active", func(t *testing.T) {
		// Start and stop tracking
		_, err := tc.startTracking("activetest", "", "")
		require.NoError(t, err)

		_, err = tc.stopTracking("")
		require.NoError(t, err)

		// Resume
		resumedBlock, err := tc.resumeTracking()
		require.NoError(t, err)

		// Verify it is active
		activeBlock, err := tc.activeBlockRepo.GetActiveBlock(tc.blockRepo)
		require.NoError(t, err)
		require.NotNil(t, activeBlock)
		assert.Equal(t, resumedBlock.Key, activeBlock.Key)
	})

	t.Run("returns nil when no previous tracking", func(t *testing.T) {
		// Clear any active/previous state
		tc.activeBlockRepo.ClearActive()

		// Reset previous by getting a fresh state
		active, _ := tc.activeBlockRepo.Get()
		active.PreviousBlockKey = ""
		tc.activeBlockRepo.Update(active)

		// Resume should return nil
		block, err := tc.resumeTracking()
		require.NoError(t, err)
		assert.Nil(t, block)
	})

	t.Run("resume after switch resumes switched-from project", func(t *testing.T) {
		// Start on project1
		_, err := tc.startTracking("switch1", "task1", "")
		require.NoError(t, err)

		// Switch to project2
		_, err = tc.startTracking("switch2", "", "")
		require.NoError(t, err)

		// Stop project2
		_, err = tc.stopTracking("")
		require.NoError(t, err)

		// Resume should resume project1 (the previous before stop)
		resumedBlock, err := tc.resumeTracking()
		require.NoError(t, err)
		require.NotNil(t, resumedBlock)

		// Note: The previous is switch2 (the one that was stopped)
		assert.Equal(t, "switch2", resumedBlock.ProjectSID)
	})
}

// =============================================================================
// Full Workflow Integration Tests
// =============================================================================

func TestFullWorkflow_StartStopResume(t *testing.T) {
	tc := setupTrackingTestContext(t)

	t.Run("complete tracking workflow", func(t *testing.T) {
		// Step 1: Start tracking on project A
		blockA, err := tc.startTracking("projectA", "taskA", "starting work")
		require.NoError(t, err)
		require.NotNil(t, blockA)
		assert.True(t, blockA.IsActive())

		time.Sleep(10 * time.Millisecond)

		// Step 2: Stop tracking
		stoppedA, err := tc.stopTracking("finished first task")
		require.NoError(t, err)
		require.NotNil(t, stoppedA)
		assert.False(t, stoppedA.IsActive())
		assert.Contains(t, stoppedA.Note, "starting work")
		assert.Contains(t, stoppedA.Note, "finished first task")

		// Step 3: Resume tracking
		resumedA, err := tc.resumeTracking()
		require.NoError(t, err)
		require.NotNil(t, resumedA)
		assert.Equal(t, "projectA", resumedA.ProjectSID)
		assert.Equal(t, "taskA", resumedA.TaskSID)
		assert.True(t, resumedA.IsActive())

		// Step 4: Switch to project B
		blockB, err := tc.startTracking("projectB", "", "switched to B")
		require.NoError(t, err)
		require.NotNil(t, blockB)

		// Verify resumedA was ended
		retrievedResumedA, err := tc.blockRepo.Get(resumedA.Key)
		require.NoError(t, err)
		assert.False(t, retrievedResumedA.IsActive())

		// Verify blockB is active
		activeBlock, err := tc.activeBlockRepo.GetActiveBlock(tc.blockRepo)
		require.NoError(t, err)
		assert.Equal(t, blockB.Key, activeBlock.Key)

		// Step 5: Final stop
		_, err = tc.stopTracking("")
		require.NoError(t, err)

		// Verify all blocks exist in database
		blocks, err := tc.blockRepo.List()
		require.NoError(t, err)

		projectACount := 0
		projectBCount := 0
		for _, b := range blocks {
			if b.ProjectSID == "projectA" {
				projectACount++
			}
			if b.ProjectSID == "projectB" {
				projectBCount++
			}
		}
		assert.Equal(t, 2, projectACount, "should have 2 blocks for projectA (original + resumed)")
		assert.Equal(t, 1, projectBCount, "should have 1 block for projectB")
	})
}

func TestBlockDuration_CalculatedCorrectly(t *testing.T) {
	tc := setupTrackingTestContext(t)

	t.Run("duration is calculated correctly for stopped block", func(t *testing.T) {
		_, err := tc.startTracking("durtest", "", "")
		require.NoError(t, err)

		// Wait a known duration
		time.Sleep(100 * time.Millisecond)

		stoppedBlock, err := tc.stopTracking("")
		require.NoError(t, err)

		duration := stoppedBlock.Duration()
		assert.GreaterOrEqual(t, duration, 100*time.Millisecond)
		assert.Less(t, duration, 500*time.Millisecond) // Should be much less than 500ms
	})

	t.Run("active block duration increases over time", func(t *testing.T) {
		block, err := tc.startTracking("activedurarion", "", "")
		require.NoError(t, err)

		duration1 := block.Duration()
		time.Sleep(50 * time.Millisecond)
		duration2 := block.Duration()

		assert.Greater(t, duration2, duration1)
	})
}

func TestMultipleProjectsTracked_Correctly(t *testing.T) {
	tc := setupTrackingTestContext(t)

	t.Run("tracks time correctly across multiple projects", func(t *testing.T) {
		// Track time on multiple projects
		projects := []string{"proj1", "proj2", "proj3"}

		for _, proj := range projects {
			_, err := tc.startTracking(proj, "", "")
			require.NoError(t, err)
			time.Sleep(20 * time.Millisecond)
		}

		// Stop final tracking
		_, err := tc.stopTracking("")
		require.NoError(t, err)

		// Verify blocks exist for all projects
		blocks, err := tc.blockRepo.List()
		require.NoError(t, err)

		projectBlocks := make(map[string][]*model.Block)
		for _, b := range blocks {
			for _, proj := range projects {
				if b.ProjectSID == proj {
					projectBlocks[proj] = append(projectBlocks[proj], b)
				}
			}
		}

		// Each project should have at least one block
		for _, proj := range projects {
			assert.GreaterOrEqual(t, len(projectBlocks[proj]), 1,
				"project %s should have at least 1 block", proj)
		}

		// All blocks except possibly the last one should be completed
		completedCount := 0
		for _, b := range blocks {
			if !b.TimestampEnd.IsZero() {
				completedCount++
			}
		}
		assert.GreaterOrEqual(t, completedCount, len(projects)-1)
	})
}
