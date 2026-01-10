package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create an in-memory database for testing
func setupTestDB(t *testing.T) *DB {
	db, err := Open(Options{InMemory: true})
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

// =============================================================================
// DB Tests
// =============================================================================

func TestOpenClose(t *testing.T) {
	t.Run("in_memory", func(t *testing.T) {
		db, err := Open(Options{InMemory: true})
		require.NoError(t, err)
		assert.NotNil(t, db)
		err = db.Close()
		assert.NoError(t, err)
	})

	t.Run("empty_path_uses_in_memory", func(t *testing.T) {
		db, err := Open(Options{Path: ""})
		require.NoError(t, err)
		assert.NotNil(t, db)
		db.Close()
	})
}

func TestDBPath(t *testing.T) {
	db, err := Open(Options{InMemory: true})
	require.NoError(t, err)
	defer db.Close()

	// In-memory DB has empty path
	assert.Equal(t, "", db.Path())
}

func TestDBBadger(t *testing.T) {
	db, err := Open(Options{InMemory: true})
	require.NoError(t, err)
	defer db.Close()

	assert.NotNil(t, db.Badger())
}

func TestCheckIntegrity(t *testing.T) {
	db, err := Open(Options{InMemory: true})
	require.NoError(t, err)
	defer db.Close()

	err = db.CheckIntegrity()
	assert.NoError(t, err)
}

func TestOpenWithIntegrityCheck(t *testing.T) {
	db, err := OpenWithIntegrityCheck(Options{InMemory: true})
	require.NoError(t, err)
	defer db.Close()
	assert.NotNil(t, db)
}

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	assert.Contains(t, path, "humantime")
	assert.Contains(t, path, "db")
}

// =============================================================================
// ProjectRepo Tests
// =============================================================================

func TestProjectRepoCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepo(db)

	project := model.NewProject("test-project", "Test Project", "A test project")
	err := repo.Create(project)
	require.NoError(t, err)
	assert.NotEmpty(t, project.Key)
}

func TestProjectRepoGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepo(db)

	// Create
	project := model.NewProject("test-project", "Test Project", "")
	err := repo.Create(project)
	require.NoError(t, err)

	// Get
	retrieved, err := repo.Get("test-project")
	require.NoError(t, err)
	assert.Equal(t, "test-project", retrieved.SID)
	assert.Equal(t, "Test Project", retrieved.DisplayName)
}

func TestProjectRepoGetNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepo(db)

	_, err := repo.Get("nonexistent")
	assert.Error(t, err)
	assert.True(t, IsErrKeyNotFound(err))
}

func TestProjectRepoGetOrCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepo(db)

	// First call creates
	project, created, err := repo.GetOrCreate("new-project", "New Project")
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, "new-project", project.SID)

	// Second call gets existing
	project2, created2, err := repo.GetOrCreate("new-project", "New Project")
	require.NoError(t, err)
	assert.False(t, created2)
	assert.Equal(t, project.SID, project2.SID)
}

func TestProjectRepoUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepo(db)

	project := model.NewProject("test-project", "Test Project", "")
	err := repo.Create(project)
	require.NoError(t, err)

	project.DisplayName = "Updated Name"
	err = repo.Update(project)
	require.NoError(t, err)

	retrieved, err := repo.Get("test-project")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.DisplayName)
}

func TestProjectRepoDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepo(db)

	project := model.NewProject("test-project", "Test Project", "")
	err := repo.Create(project)
	require.NoError(t, err)

	err = repo.Delete("test-project")
	require.NoError(t, err)

	_, err = repo.Get("test-project")
	assert.True(t, IsErrKeyNotFound(err))
}

func TestProjectRepoList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepo(db)

	// Create multiple projects
	for i := 0; i < 3; i++ {
		project := model.NewProject(
			"project-"+string(rune('a'+i)),
			"Project "+string(rune('A'+i)),
			"",
		)
		err := repo.Create(project)
		require.NoError(t, err)
	}

	projects, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, projects, 3)
}

func TestProjectRepoExists(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepo(db)

	project := model.NewProject("test-project", "Test Project", "")
	err := repo.Create(project)
	require.NoError(t, err)

	exists, err := repo.Exists("test-project")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.Exists("nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

// =============================================================================
// BlockRepo Tests
// =============================================================================

func TestBlockRepoCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlockRepo(db)

	block := &model.Block{
		ProjectSID:     "test-project",
		TimestampStart: time.Now(),
	}
	err := repo.Create(block)
	require.NoError(t, err)
	assert.NotEmpty(t, block.Key)
}

func TestBlockRepoGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlockRepo(db)

	block := &model.Block{
		ProjectSID:     "test-project",
		TimestampStart: time.Now(),
	}
	err := repo.Create(block)
	require.NoError(t, err)

	retrieved, err := repo.Get(block.Key)
	require.NoError(t, err)
	assert.Equal(t, block.ProjectSID, retrieved.ProjectSID)
}

func TestBlockRepoUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlockRepo(db)

	block := &model.Block{
		ProjectSID:     "test-project",
		TimestampStart: time.Now(),
	}
	err := repo.Create(block)
	require.NoError(t, err)

	block.Note = "Updated note"
	err = repo.Update(block)
	require.NoError(t, err)

	retrieved, err := repo.Get(block.Key)
	require.NoError(t, err)
	assert.Equal(t, "Updated note", retrieved.Note)
}

func TestBlockRepoDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlockRepo(db)

	block := &model.Block{
		ProjectSID:     "test-project",
		TimestampStart: time.Now(),
	}
	err := repo.Create(block)
	require.NoError(t, err)

	err = repo.Delete(block.Key)
	require.NoError(t, err)

	_, err = repo.Get(block.Key)
	assert.Error(t, err)
}

func TestBlockRepoList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlockRepo(db)

	// Create multiple blocks
	for i := 0; i < 3; i++ {
		block := &model.Block{
			ProjectSID:     "test-project",
			TimestampStart: time.Now(),
		}
		err := repo.Create(block)
		require.NoError(t, err)
	}

	blocks, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, blocks, 3)
}

func TestBlockRepoListByProject(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlockRepo(db)

	// Create blocks for different projects
	for i := 0; i < 2; i++ {
		block := &model.Block{
			ProjectSID:     "project-a",
			TimestampStart: time.Now(),
		}
		err := repo.Create(block)
		require.NoError(t, err)
	}

	block := &model.Block{
		ProjectSID:     "project-b",
		TimestampStart: time.Now(),
	}
	err := repo.Create(block)
	require.NoError(t, err)

	blocks, err := repo.ListByProject("project-a")
	require.NoError(t, err)
	assert.Len(t, blocks, 2)

	blocks, err = repo.ListByProject("project-b")
	require.NoError(t, err)
	assert.Len(t, blocks, 1)
}

func TestBlockRepoListByProjectAndTask(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlockRepo(db)

	// Create blocks
	block1 := &model.Block{
		ProjectSID:     "project-a",
		TaskSID:        "task-1",
		TimestampStart: time.Now(),
	}
	err := repo.Create(block1)
	require.NoError(t, err)

	block2 := &model.Block{
		ProjectSID:     "project-a",
		TaskSID:        "task-2",
		TimestampStart: time.Now(),
	}
	err = repo.Create(block2)
	require.NoError(t, err)

	blocks, err := repo.ListByProjectAndTask("project-a", "task-1")
	require.NoError(t, err)
	assert.Len(t, blocks, 1)
}

func TestBlockRepoListByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlockRepo(db)

	now := time.Now()

	// Create block in range
	block1 := &model.Block{
		ProjectSID:     "test",
		TimestampStart: now.Add(-1 * time.Hour),
		TimestampEnd:   now,
	}
	err := repo.Create(block1)
	require.NoError(t, err)

	// Create block outside range
	block2 := &model.Block{
		ProjectSID:     "test",
		TimestampStart: now.Add(-24 * time.Hour),
		TimestampEnd:   now.Add(-23 * time.Hour),
	}
	err = repo.Create(block2)
	require.NoError(t, err)

	// Query for last 2 hours
	blocks, err := repo.ListByTimeRange(now.Add(-2*time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	assert.Len(t, blocks, 1)
}

func TestBlockRepoListFiltered(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlockRepo(db)

	now := time.Now()

	// Create blocks
	for i := 0; i < 5; i++ {
		block := &model.Block{
			ProjectSID:     "test-project",
			TimestampStart: now.Add(time.Duration(-i) * time.Hour),
			TimestampEnd:   now.Add(time.Duration(-i)*time.Hour + 30*time.Minute),
		}
		err := repo.Create(block)
		require.NoError(t, err)
	}

	// Filter with limit
	blocks, err := repo.ListFiltered(BlockFilter{
		ProjectSID: "test-project",
		Limit:      3,
	})
	require.NoError(t, err)
	assert.Len(t, blocks, 3)
}

// =============================================================================
// Aggregate Tests
// =============================================================================

func TestTotalDuration(t *testing.T) {
	now := time.Now()
	blocks := []*model.Block{
		{TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},
		{TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
	}

	total := TotalDuration(blocks)
	assert.Equal(t, 2*time.Hour, total)
}

func TestAggregateByProject(t *testing.T) {
	now := time.Now()
	blocks := []*model.Block{
		{ProjectSID: "project-a", TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},
		{ProjectSID: "project-a", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
		{ProjectSID: "project-b", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
	}

	agg := AggregateByProject(blocks)
	assert.Len(t, agg, 2)

	// Project A should have 2 hours, Project B should have 1 hour
	// Sorted by duration, A should be first
	assert.Equal(t, "project-a", agg[0].ProjectSID)
	assert.Equal(t, 2, agg[0].BlockCount)
	assert.Equal(t, 2*time.Hour, agg[0].Duration)
}

// =============================================================================
// Safety Tests
// =============================================================================

func TestDiskSpaceInfo(t *testing.T) {
	t.Run("free_percent_zero_total", func(t *testing.T) {
		info := &DiskSpaceInfo{TotalBytes: 0, FreeBytes: 100}
		assert.Equal(t, 0.0, info.FreePercent())
	})

	t.Run("free_percent_calculation", func(t *testing.T) {
		info := &DiskSpaceInfo{TotalBytes: 1000, FreeBytes: 250}
		assert.Equal(t, 25.0, info.FreePercent())
	})
}

func TestGetDiskSpace(t *testing.T) {
	t.Run("current_directory", func(t *testing.T) {
		info, err := GetDiskSpace(".")
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Greater(t, info.TotalBytes, uint64(0))
	})

	t.Run("nonexistent_uses_parent", func(t *testing.T) {
		info, err := GetDiskSpace("/nonexistent/path/here")
		// Should find the root directory
		require.NoError(t, err)
		assert.NotNil(t, info)
	})
}

func TestCheckDiskSpace(t *testing.T) {
	t.Run("current_directory_has_space", func(t *testing.T) {
		err := CheckDiskSpace(".")
		// Unless running on a nearly-full disk, this should pass
		assert.NoError(t, err)
	})
}

func TestCheckDiskSpaceWarning(t *testing.T) {
	t.Run("no_warning_on_normal_disk", func(t *testing.T) {
		warning := CheckDiskSpaceWarning(".")
		// Unless running on a nearly-full disk
		assert.Empty(t, warning)
	})
}

func TestIsDiskFullError(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		assert.False(t, isDiskFullError(nil))
	})

	t.Run("regular_error", func(t *testing.T) {
		err := fmt.Errorf("some error")
		assert.False(t, isDiskFullError(err))
	})
}

func TestEnsureDirectory(t *testing.T) {
	t.Run("creates_temp_directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "subdir", "nested")

		err := EnsureDirectory(testPath)
		require.NoError(t, err)

		info, err := os.Stat(testPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

// =============================================================================
// Recovery Tests
// =============================================================================

func TestCheckDatabaseIntegrity(t *testing.T) {
	t.Run("nil_database", func(t *testing.T) {
		status := CheckDatabaseIntegrity(nil)
		assert.False(t, status.Healthy)
		assert.True(t, status.Corrupted)
	})

	t.Run("healthy_database", func(t *testing.T) {
		db := setupTestDB(t)
		status := CheckDatabaseIntegrity(db)
		assert.True(t, status.Healthy)
		assert.False(t, status.Corrupted)
	})
}

func TestRecoveryStatus(t *testing.T) {
	status := &RecoveryStatus{
		Healthy:     true,
		LastCheck:   time.Now(),
		ErrorCount:  0,
		Recoverable: false,
	}

	assert.True(t, status.Healthy)
	assert.Zero(t, status.ErrorCount)
}

func TestIsDatabaseCorrupted(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		assert.False(t, IsDatabaseCorrupted(nil))
	})

	t.Run("regular_error", func(t *testing.T) {
		err := fmt.Errorf("some error")
		assert.False(t, IsDatabaseCorrupted(err))
	})

	t.Run("checksum_mismatch", func(t *testing.T) {
		err := fmt.Errorf("checksum mismatch detected")
		assert.True(t, IsDatabaseCorrupted(err))
	})

	t.Run("corrupt_in_message", func(t *testing.T) {
		err := fmt.Errorf("data corrupt")
		assert.True(t, IsDatabaseCorrupted(err))
	})

	t.Run("invalid_data", func(t *testing.T) {
		err := fmt.Errorf("invalid data format")
		assert.True(t, IsDatabaseCorrupted(err))
	})
}

func TestContains(t *testing.T) {
	assert.True(t, contains("Hello World", "world"))
	assert.True(t, contains("HELLO", "hello"))
	assert.False(t, contains("Hello", "xyz"))
	assert.False(t, contains("Hi", "Hello"))
}

func TestEqualFold(t *testing.T) {
	assert.True(t, equalFold("hello", "HELLO"))
	assert.True(t, equalFold("Hello", "hElLo"))
	assert.False(t, equalFold("hello", "world"))
	assert.False(t, equalFold("hi", "hello"))
}

func TestCreateBackupEmptyPath(t *testing.T) {
	_, err := CreateBackup("")
	assert.Error(t, err)
}

// =============================================================================
// TaskRepo Tests
// =============================================================================

func TestTaskRepoCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepo(db)

	task := model.NewTask("myproject", "mytask", "My Task", "#FF0000")
	err := repo.Create(task)
	require.NoError(t, err)
	assert.NotEmpty(t, task.Key)
}

func TestTaskRepoGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepo(db)

	task := model.NewTask("myproject", "mytask", "My Task", "")
	err := repo.Create(task)
	require.NoError(t, err)

	retrieved, err := repo.Get("myproject", "mytask")
	require.NoError(t, err)
	assert.Equal(t, "mytask", retrieved.SID)
	assert.Equal(t, "myproject", retrieved.ProjectSID)
}

func TestTaskRepoGetNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepo(db)

	_, err := repo.Get("nonexistent", "task")
	assert.Error(t, err)
	assert.True(t, IsErrKeyNotFound(err))
}

func TestTaskRepoGetOrCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepo(db)

	// First call creates
	task, created, err := repo.GetOrCreate("proj", "task1", "Task 1")
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, "task1", task.SID)

	// Second call returns existing
	task2, created2, err := repo.GetOrCreate("proj", "task1", "Task 1")
	require.NoError(t, err)
	assert.False(t, created2)
	assert.Equal(t, task.Key, task2.Key)
}

func TestTaskRepoUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepo(db)

	task := model.NewTask("proj", "task", "Original", "")
	err := repo.Create(task)
	require.NoError(t, err)

	task.DisplayName = "Updated"
	err = repo.Update(task)
	require.NoError(t, err)

	retrieved, err := repo.Get("proj", "task")
	require.NoError(t, err)
	assert.Equal(t, "Updated", retrieved.DisplayName)
}

func TestTaskRepoDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepo(db)

	task := model.NewTask("proj", "task", "Task", "")
	err := repo.Create(task)
	require.NoError(t, err)

	err = repo.Delete("proj", "task")
	require.NoError(t, err)

	_, err = repo.Get("proj", "task")
	assert.True(t, IsErrKeyNotFound(err))
}

func TestTaskRepoList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepo(db)

	// Create multiple tasks
	for i := 0; i < 3; i++ {
		task := model.NewTask("proj", fmt.Sprintf("task%d", i), fmt.Sprintf("Task %d", i), "")
		err := repo.Create(task)
		require.NoError(t, err)
	}

	tasks, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func TestTaskRepoListByProject(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepo(db)

	// Create tasks for different projects
	task1 := model.NewTask("proj-a", "task1", "Task 1", "")
	err := repo.Create(task1)
	require.NoError(t, err)

	task2 := model.NewTask("proj-a", "task2", "Task 2", "")
	err = repo.Create(task2)
	require.NoError(t, err)

	task3 := model.NewTask("proj-b", "task1", "Task 1", "")
	err = repo.Create(task3)
	require.NoError(t, err)

	tasks, err := repo.ListByProject("proj-a")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	tasks, err = repo.ListByProject("proj-b")
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestTaskRepoExists(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepo(db)

	task := model.NewTask("proj", "task", "Task", "")
	err := repo.Create(task)
	require.NoError(t, err)

	exists, err := repo.Exists("proj", "task")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.Exists("proj", "nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

// =============================================================================
// GoalRepo Tests
// =============================================================================

func TestGoalRepoCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGoalRepo(db)

	goal := model.NewGoal("myproject", model.GoalTypeDaily, 4*time.Hour)
	err := repo.Create(goal)
	require.NoError(t, err)
	assert.NotEmpty(t, goal.Key)
}

func TestGoalRepoGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGoalRepo(db)

	goal := model.NewGoal("myproject", model.GoalTypeDaily, 4*time.Hour)
	err := repo.Create(goal)
	require.NoError(t, err)

	retrieved, err := repo.Get("myproject")
	require.NoError(t, err)
	assert.Equal(t, "myproject", retrieved.ProjectSID)
	assert.Equal(t, model.GoalTypeDaily, retrieved.Type)
}

func TestGoalRepoGetNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGoalRepo(db)

	_, err := repo.Get("nonexistent")
	assert.Error(t, err)
	assert.True(t, IsErrKeyNotFound(err))
}

func TestGoalRepoUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGoalRepo(db)

	goal := model.NewGoal("proj", model.GoalTypeDaily, 4*time.Hour)
	err := repo.Create(goal)
	require.NoError(t, err)

	goal.Target = 8 * time.Hour
	err = repo.Update(goal)
	require.NoError(t, err)

	retrieved, err := repo.Get("proj")
	require.NoError(t, err)
	assert.Equal(t, 8*time.Hour, retrieved.Target)
}

func TestGoalRepoUpsert(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGoalRepo(db)

	// First upsert creates
	goal := model.NewGoal("proj", model.GoalTypeDaily, 4*time.Hour)
	err := repo.Upsert(goal)
	require.NoError(t, err)

	retrieved, err := repo.Get("proj")
	require.NoError(t, err)
	assert.Equal(t, 4*time.Hour, retrieved.Target)

	// Second upsert updates
	goal.Target = 6 * time.Hour
	err = repo.Upsert(goal)
	require.NoError(t, err)

	retrieved, err = repo.Get("proj")
	require.NoError(t, err)
	assert.Equal(t, 6*time.Hour, retrieved.Target)
}

func TestGoalRepoDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGoalRepo(db)

	goal := model.NewGoal("proj", model.GoalTypeDaily, 4*time.Hour)
	err := repo.Create(goal)
	require.NoError(t, err)

	err = repo.Delete("proj")
	require.NoError(t, err)

	_, err = repo.Get("proj")
	assert.True(t, IsErrKeyNotFound(err))
}

func TestGoalRepoList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGoalRepo(db)

	// Create multiple goals
	for i := 0; i < 3; i++ {
		goal := model.NewGoal(fmt.Sprintf("proj%d", i), model.GoalTypeDaily, 4*time.Hour)
		err := repo.Create(goal)
		require.NoError(t, err)
	}

	goals, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, goals, 3)
}

func TestGoalRepoExists(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGoalRepo(db)

	goal := model.NewGoal("proj", model.GoalTypeDaily, 4*time.Hour)
	err := repo.Create(goal)
	require.NoError(t, err)

	exists, err := repo.Exists("proj")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.Exists("nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

// =============================================================================
// UndoRepo Tests
// =============================================================================

func TestUndoRepoGetEmpty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUndoRepo(db)

	state, err := repo.Get()
	require.NoError(t, err)
	assert.Nil(t, state)
}

func TestUndoRepoSetAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUndoRepo(db)

	state := model.NewUndoState(model.UndoActionStart, "block:123", nil)
	err := repo.Set(state)
	require.NoError(t, err)

	retrieved, err := repo.Get()
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, model.UndoActionStart, retrieved.Action)
	assert.Equal(t, "block:123", retrieved.BlockKey)
}

func TestUndoRepoClear(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUndoRepo(db)

	state := model.NewUndoState(model.UndoActionStart, "block:123", nil)
	err := repo.Set(state)
	require.NoError(t, err)

	err = repo.Clear()
	require.NoError(t, err)

	retrieved, err := repo.Get()
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestUndoRepoSaveUndoStart(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUndoRepo(db)

	err := repo.SaveUndoStart("block:abc123")
	require.NoError(t, err)

	state, err := repo.Get()
	require.NoError(t, err)
	assert.Equal(t, model.UndoActionStart, state.Action)
	assert.Equal(t, "block:abc123", state.BlockKey)
	assert.Nil(t, state.BlockSnapshot)
}

func TestUndoRepoSaveUndoStop(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUndoRepo(db)

	block := &model.Block{
		Key:            "block:123",
		ProjectSID:     "myproject",
		TimestampStart: time.Now().Add(-1 * time.Hour),
		TimestampEnd:   time.Now(),
	}
	err := repo.SaveUndoStop(block)
	require.NoError(t, err)

	state, err := repo.Get()
	require.NoError(t, err)
	assert.Equal(t, model.UndoActionStop, state.Action)
	assert.Equal(t, "block:123", state.BlockKey)
	assert.NotNil(t, state.BlockSnapshot)
}

func TestUndoRepoSaveUndoDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUndoRepo(db)

	block := &model.Block{
		Key:            "block:456",
		OwnerKey:       "owner1",
		ProjectSID:     "myproject",
		TaskSID:        "mytask",
		Note:           "Some note",
		Tags:           []string{"tag1", "tag2"},
		TimestampStart: time.Now().Add(-2 * time.Hour),
		TimestampEnd:   time.Now().Add(-1 * time.Hour),
	}
	err := repo.SaveUndoDelete(block)
	require.NoError(t, err)

	state, err := repo.Get()
	require.NoError(t, err)
	assert.Equal(t, model.UndoActionDelete, state.Action)
	assert.Equal(t, "block:456", state.BlockKey)
	assert.NotNil(t, state.BlockSnapshot)
	assert.Equal(t, "myproject", state.BlockSnapshot.ProjectSID)
	assert.Equal(t, "mytask", state.BlockSnapshot.TaskSID)
	assert.Equal(t, "Some note", state.BlockSnapshot.Note)
}

// =============================================================================
// ConfigRepo Tests
// =============================================================================

func TestConfigRepoGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConfigRepo(db)

	config, err := repo.Get()
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.NotEmpty(t, config.UserKey)
}

func TestConfigRepoUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConfigRepo(db)

	config, err := repo.Get()
	require.NoError(t, err)

	originalKey := config.UserKey

	// Get again - should return same key
	config2, err := repo.Get()
	require.NoError(t, err)
	assert.Equal(t, originalKey, config2.UserKey)
}

// =============================================================================
// ActiveBlockRepo Tests
// =============================================================================

func TestActiveBlockRepoGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewActiveBlockRepo(db)

	ab, err := repo.Get()
	require.NoError(t, err)
	assert.NotNil(t, ab)
	assert.Equal(t, model.KeyActiveBlock, ab.Key)
}

func TestActiveBlockRepoSetActive(t *testing.T) {
	db := setupTestDB(t)
	repo := NewActiveBlockRepo(db)

	err := repo.SetActive("block:123")
	require.NoError(t, err)

	ab, err := repo.Get()
	require.NoError(t, err)
	assert.Equal(t, "block:123", ab.ActiveBlockKey)
	assert.True(t, ab.IsTracking())
}

func TestActiveBlockRepoClearActive(t *testing.T) {
	db := setupTestDB(t)
	repo := NewActiveBlockRepo(db)

	err := repo.SetActive("block:123")
	require.NoError(t, err)

	err = repo.ClearActive()
	require.NoError(t, err)

	ab, err := repo.Get()
	require.NoError(t, err)
	assert.Empty(t, ab.ActiveBlockKey)
	assert.False(t, ab.IsTracking())
	assert.Equal(t, "block:123", ab.PreviousBlockKey)
}

// =============================================================================
// ReminderRepo Tests
// =============================================================================

func TestReminderRepoCreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	reminder := model.NewReminder("Test Reminder", time.Now().Add(24*time.Hour), "owner1")
	err := repo.Create(reminder)
	require.NoError(t, err)
	assert.NotEmpty(t, reminder.Key)

	retrieved, err := repo.Get(reminder.Key)
	require.NoError(t, err)
	assert.Equal(t, "Test Reminder", retrieved.Title)
}

func TestReminderRepoList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	// Create multiple reminders
	for i := 0; i < 3; i++ {
		reminder := model.NewReminder(fmt.Sprintf("Reminder %d", i), time.Now().Add(time.Duration(i)*time.Hour), "owner1")
		err := repo.Create(reminder)
		require.NoError(t, err)
	}

	reminders, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, reminders, 3)
}

func TestReminderRepoUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	reminder := model.NewReminder("Original", time.Now().Add(1*time.Hour), "owner1")
	err := repo.Create(reminder)
	require.NoError(t, err)

	reminder.Title = "Updated"
	err = repo.Update(reminder)
	require.NoError(t, err)

	retrieved, err := repo.Get(reminder.Key)
	require.NoError(t, err)
	assert.Equal(t, "Updated", retrieved.Title)
}

func TestReminderRepoDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	reminder := model.NewReminder("To Delete", time.Now().Add(1*time.Hour), "owner1")
	err := repo.Create(reminder)
	require.NoError(t, err)

	err = repo.Delete(reminder.Key)
	require.NoError(t, err)

	_, err = repo.Get(reminder.Key)
	assert.True(t, IsErrKeyNotFound(err))
}

func TestReminderRepoListPending(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	// Create pending reminder
	pending := model.NewReminder("Pending", time.Now().Add(1*time.Hour), "owner1")
	err := repo.Create(pending)
	require.NoError(t, err)

	// Create completed reminder
	completed := model.NewReminder("Completed", time.Now().Add(1*time.Hour), "owner1")
	completed.Completed = true
	err = repo.Create(completed)
	require.NoError(t, err)

	reminders, err := repo.ListPending()
	require.NoError(t, err)
	assert.Len(t, reminders, 1)
	assert.Equal(t, "Pending", reminders[0].Title)
}

// =============================================================================
// WebhookRepo Tests
// =============================================================================

func TestWebhookRepoCreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)

	webhook := model.NewWebhook("my-hook", model.WebhookTypeDiscord, "https://discord.com/api/webhooks/123")
	err := repo.Create(webhook)
	require.NoError(t, err)

	retrieved, err := repo.Get("my-hook")
	require.NoError(t, err)
	assert.Equal(t, "my-hook", retrieved.Name)
	assert.Equal(t, model.WebhookTypeDiscord, retrieved.Type)
}

func TestWebhookRepoList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)

	// Create multiple webhooks
	for i := 0; i < 3; i++ {
		webhook := model.NewWebhook(fmt.Sprintf("hook%d", i), model.WebhookTypeGeneric, fmt.Sprintf("https://example.com/hook%d", i))
		err := repo.Create(webhook)
		require.NoError(t, err)
	}

	webhooks, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, webhooks, 3)
}

func TestWebhookRepoUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)

	webhook := model.NewWebhook("my-hook", model.WebhookTypeGeneric, "https://example.com/original")
	err := repo.Create(webhook)
	require.NoError(t, err)

	webhook.URL = "https://example.com/updated"
	err = repo.Update(webhook)
	require.NoError(t, err)

	retrieved, err := repo.Get("my-hook")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/updated", retrieved.URL)
}

func TestWebhookRepoDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)

	webhook := model.NewWebhook("to-delete", model.WebhookTypeGeneric, "https://example.com/delete")
	err := repo.Create(webhook)
	require.NoError(t, err)

	err = repo.Delete("to-delete")
	require.NoError(t, err)

	_, err = repo.Get("to-delete")
	assert.True(t, IsErrKeyNotFound(err))
}

func TestWebhookRepoListEnabled(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)

	// Create enabled webhook
	enabled := model.NewWebhook("enabled", model.WebhookTypeGeneric, "https://example.com/enabled")
	enabled.Enabled = true
	err := repo.Create(enabled)
	require.NoError(t, err)

	// Create disabled webhook
	disabled := model.NewWebhook("disabled", model.WebhookTypeGeneric, "https://example.com/disabled")
	disabled.Enabled = false
	err = repo.Create(disabled)
	require.NoError(t, err)

	webhooks, err := repo.ListEnabled()
	require.NoError(t, err)
	assert.Len(t, webhooks, 1)
	assert.Equal(t, "enabled", webhooks[0].Name)
}

func TestWebhookRepoExists(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)

	webhook := model.NewWebhook("my-hook", model.WebhookTypeGeneric, "https://example.com/hook")
	err := repo.Create(webhook)
	require.NoError(t, err)

	exists, err := repo.Exists("my-hook")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.Exists("nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

// =============================================================================
// NotifyConfigRepo Tests
// =============================================================================

func TestNotifyConfigRepoGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNotifyConfigRepo(db)

	config, err := repo.Get()
	require.NoError(t, err)
	assert.NotNil(t, config)
	// Should have default values
	assert.Equal(t, 30*time.Minute, config.IdleAfter)
}

func TestNotifyConfigRepoUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNotifyConfigRepo(db)

	config, err := repo.Get()
	require.NoError(t, err)

	config.IdleAfter = 45 * time.Minute
	err = repo.Update(config)
	require.NoError(t, err)

	retrieved, err := repo.Get()
	require.NoError(t, err)
	assert.Equal(t, 45*time.Minute, retrieved.IdleAfter)
}

// =============================================================================
// Additional ReminderRepo Tests
// =============================================================================

func TestReminderRepoGetByShortID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	// Create a reminder
	reminder := model.NewReminder("Test Reminder", time.Now().Add(1*time.Hour), "owner1")
	err := repo.Create(reminder)
	require.NoError(t, err)

	shortID := reminder.ShortID()

	// Get by full short ID
	found, err := repo.GetByShortID(shortID)
	require.NoError(t, err)
	assert.Equal(t, reminder.Title, found.Title)

	// Get by partial prefix (first 4 chars of short ID if long enough)
	if len(shortID) >= 4 {
		found, err = repo.GetByShortID(shortID[:4])
		require.NoError(t, err)
		assert.Equal(t, reminder.Title, found.Title)
	}

	// Non-existent short ID
	_, err = repo.GetByShortID("nonexistent123")
	assert.Error(t, err)
}

func TestReminderRepoGetByShortIDAmbiguous(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	// Create multiple reminders
	for i := 0; i < 5; i++ {
		reminder := model.NewReminder(fmt.Sprintf("Reminder %d", i), time.Now().Add(time.Duration(i+1)*time.Hour), "owner1")
		err := repo.Create(reminder)
		require.NoError(t, err)
	}

	// Try to match with a very short prefix that might match multiple
	// This tests the AmbiguousMatchError path
	reminders, _ := repo.List()
	if len(reminders) > 1 {
		// Get the first character of the first reminder's short ID
		firstChar := reminders[0].ShortID()[:1]

		// Count how many match
		matchCount := 0
		for _, r := range reminders {
			if len(r.ShortID()) > 0 && r.ShortID()[:1] == firstChar {
				matchCount++
			}
		}

		if matchCount > 1 {
			_, err := repo.GetByShortID(firstChar)
			assert.Error(t, err)
			var ambErr *AmbiguousMatchError
			assert.ErrorAs(t, err, &ambErr)
		}
	}
}

func TestReminderRepoGetByTitle(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	// Create reminders with different titles
	reminder1 := model.NewReminder("Unique Title", time.Now().Add(1*time.Hour), "owner1")
	err := repo.Create(reminder1)
	require.NoError(t, err)

	reminder2 := model.NewReminder("Another Title", time.Now().Add(2*time.Hour), "owner1")
	err = repo.Create(reminder2)
	require.NoError(t, err)

	// Find by exact title
	found, err := repo.GetByTitle("Unique Title")
	require.NoError(t, err)
	assert.Equal(t, "Unique Title", found.Title)

	// Non-existent title
	_, err = repo.GetByTitle("Nonexistent Title")
	assert.Error(t, err)
}

func TestReminderRepoListDue(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	// Create a reminder that's already due (past deadline)
	dueReminder := model.NewReminder("Due Reminder", time.Now().Add(-1*time.Hour), "owner1")
	err := repo.Create(dueReminder)
	require.NoError(t, err)

	// Create a reminder in the near future (within 2 hours)
	nearFutureReminder := model.NewReminder("Near Future Reminder", time.Now().Add(30*time.Minute), "owner1")
	err = repo.Create(nearFutureReminder)
	require.NoError(t, err)

	// Create a reminder in the far future
	farFutureReminder := model.NewReminder("Far Future Reminder", time.Now().Add(24*time.Hour), "owner1")
	err = repo.Create(farFutureReminder)
	require.NoError(t, err)

	// List reminders due within 1 hour
	due, err := repo.ListDue(1 * time.Hour)
	require.NoError(t, err)
	// Should include the due reminder and near future reminder (within 1 hour)
	assert.GreaterOrEqual(t, len(due), 1)
}

func TestReminderRepoListByProject(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	// Create reminders for different projects
	reminder1 := model.NewReminder("Reminder 1", time.Now().Add(1*time.Hour), "owner1")
	reminder1.ProjectSID = "project-a"
	err := repo.Create(reminder1)
	require.NoError(t, err)

	reminder2 := model.NewReminder("Reminder 2", time.Now().Add(2*time.Hour), "owner1")
	reminder2.ProjectSID = "project-b"
	err = repo.Create(reminder2)
	require.NoError(t, err)

	reminder3 := model.NewReminder("Reminder 3", time.Now().Add(3*time.Hour), "owner1")
	reminder3.ProjectSID = "project-a"
	err = repo.Create(reminder3)
	require.NoError(t, err)

	// List by project
	projectAReminders, err := repo.ListByProject("project-a")
	require.NoError(t, err)
	assert.Len(t, projectAReminders, 2)
}

func TestReminderRepoMarkComplete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	// Create a reminder
	reminder := model.NewReminder("To Complete", time.Now().Add(1*time.Hour), "owner1")
	err := repo.Create(reminder)
	require.NoError(t, err)

	// Mark complete
	err = repo.MarkComplete(reminder.Key)
	require.NoError(t, err)

	// Verify
	found, err := repo.Get(reminder.Key)
	require.NoError(t, err)
	assert.True(t, found.Completed)
	assert.False(t, found.CompletedAt.IsZero())
}

func TestReminderRepoCreateNextRecurrence(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	// Create a recurring reminder
	reminder := model.NewReminder("Recurring", time.Now().Add(1*time.Hour), "owner1")
	reminder.RepeatRule = "daily"
	err := repo.Create(reminder)
	require.NoError(t, err)

	// Create next recurrence
	next, err := repo.CreateNextRecurrence(reminder)
	require.NoError(t, err)
	assert.NotNil(t, next)
	assert.NotEqual(t, reminder.Key, next.Key)
	assert.Equal(t, reminder.Title, next.Title)
}

func TestReminderRepoCreateNextRecurrenceNonRecurring(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReminderRepo(db)

	// Create a non-recurring reminder
	reminder := model.NewReminder("One-time", time.Now().Add(1*time.Hour), "owner1")
	err := repo.Create(reminder)
	require.NoError(t, err)

	// CreateNextRecurrence should return nil for non-recurring
	next, err := repo.CreateNextRecurrence(reminder)
	require.NoError(t, err)
	assert.Nil(t, next)
}

func TestAmbiguousMatchErrorError(t *testing.T) {
	err := &AmbiguousMatchError{Matches: 5}
	assert.Contains(t, err.Error(), "multiple reminders match")
}

// =============================================================================
// Additional WebhookRepo Tests
// =============================================================================

func TestWebhookRepoGetByKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)

	webhook := model.NewWebhook("my-hook", model.WebhookTypeGeneric, "https://example.com/hook")
	err := repo.Create(webhook)
	require.NoError(t, err)

	// Get by full key
	found, err := repo.GetByKey(webhook.GetKey())
	require.NoError(t, err)
	assert.Equal(t, "my-hook", found.Name)

	// Non-existent key
	_, err = repo.GetByKey("webhook:nonexistent")
	assert.Error(t, err)
}

func TestWebhookRepoDisable(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)

	webhook := model.NewWebhook("my-hook", model.WebhookTypeGeneric, "https://example.com/hook")
	webhook.Enabled = true
	err := repo.Create(webhook)
	require.NoError(t, err)

	// Disable
	err = repo.Disable("my-hook")
	require.NoError(t, err)

	found, err := repo.Get("my-hook")
	require.NoError(t, err)
	assert.False(t, found.Enabled)
}

func TestWebhookRepoUpdateLastUsed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWebhookRepo(db)

	webhook := model.NewWebhook("my-hook", model.WebhookTypeGeneric, "https://example.com/hook")
	err := repo.Create(webhook)
	require.NoError(t, err)

	// Update last used with success
	err = repo.UpdateLastUsed("my-hook", nil)
	require.NoError(t, err)

	found, err := repo.Get("my-hook")
	require.NoError(t, err)
	assert.False(t, found.LastUsed.IsZero())
	assert.Empty(t, found.LastError)

	// Update last used with error
	testErr := fmt.Errorf("connection refused")
	err = repo.UpdateLastUsed("my-hook", testErr)
	require.NoError(t, err)

	found, err = repo.Get("my-hook")
	require.NoError(t, err)
	assert.Contains(t, found.LastError, "connection refused")
}
