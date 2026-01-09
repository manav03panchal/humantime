// Package contract provides contract tests for the storage layer.
// These tests verify that the storage implementation correctly handles
// all CRUD operations and behaves according to its contract.
package contract

import (
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// setupTestDB creates a new in-memory database for testing.
func setupTestDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err, "failed to open in-memory database")
	t.Cleanup(func() {
		err := db.Close()
		assert.NoError(t, err, "failed to close database")
	})
	return db
}

// =============================================================================
// Database Connection Tests
// =============================================================================

func TestDB_Open_InMemory(t *testing.T) {
	t.Run("opens with InMemory flag", func(t *testing.T) {
		db, err := storage.Open(storage.Options{InMemory: true})
		require.NoError(t, err)
		assert.NotNil(t, db)
		require.NoError(t, db.Close())
	})

	t.Run("opens with empty Path (defaults to in-memory)", func(t *testing.T) {
		db, err := storage.Open(storage.Options{Path: ""})
		require.NoError(t, err)
		assert.NotNil(t, db)
		require.NoError(t, db.Close())
	})

	t.Run("InMemory flag overrides Path", func(t *testing.T) {
		db, err := storage.Open(storage.Options{
			Path:     "/nonexistent/path",
			InMemory: true,
		})
		require.NoError(t, err)
		assert.NotNil(t, db)
		require.NoError(t, db.Close())
	})
}

func TestDB_Close(t *testing.T) {
	t.Run("closes successfully", func(t *testing.T) {
		db, err := storage.Open(storage.Options{InMemory: true})
		require.NoError(t, err)
		err = db.Close()
		assert.NoError(t, err)
	})

	t.Run("Badger returns underlying database", func(t *testing.T) {
		db, err := storage.Open(storage.Options{InMemory: true})
		require.NoError(t, err)
		defer db.Close()

		badgerDB := db.Badger()
		assert.NotNil(t, badgerDB)
	})
}

func TestDB_DefaultPath(t *testing.T) {
	t.Run("returns non-empty path", func(t *testing.T) {
		path := storage.DefaultPath()
		assert.NotEmpty(t, path)
		assert.Contains(t, path, storage.AppName)
	})
}

// =============================================================================
// Block Repository Tests
// =============================================================================

func TestBlockRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)

	t.Run("creates block with generated key", func(t *testing.T) {
		block := model.NewBlock("user-123", "myproject", "task1", "working on feature", time.Now())
		err := repo.Create(block)
		require.NoError(t, err)

		assert.NotEmpty(t, block.Key)
		assert.Contains(t, block.Key, model.PrefixBlock+":")
	})

	t.Run("creates multiple blocks with unique keys", func(t *testing.T) {
		block1 := model.NewBlock("user-123", "proj1", "", "", time.Now())
		block2 := model.NewBlock("user-123", "proj2", "", "", time.Now())

		require.NoError(t, repo.Create(block1))
		require.NoError(t, repo.Create(block2))

		assert.NotEqual(t, block1.Key, block2.Key)
	})
}

func TestBlockRepo_Get(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)

	t.Run("retrieves existing block", func(t *testing.T) {
		original := model.NewBlock("user-123", "myproject", "task1", "test note", time.Now().Add(-1*time.Hour))
		original.TimestampEnd = time.Now()
		require.NoError(t, repo.Create(original))

		retrieved, err := repo.Get(original.Key)
		require.NoError(t, err)

		assert.Equal(t, original.Key, retrieved.Key)
		assert.Equal(t, original.OwnerKey, retrieved.OwnerKey)
		assert.Equal(t, original.ProjectSID, retrieved.ProjectSID)
		assert.Equal(t, original.TaskSID, retrieved.TaskSID)
		assert.Equal(t, original.Note, retrieved.Note)
		assert.WithinDuration(t, original.TimestampStart, retrieved.TimestampStart, time.Second)
		assert.WithinDuration(t, original.TimestampEnd, retrieved.TimestampEnd, time.Second)
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		_, err := repo.Get("block:nonexistent")
		assert.Error(t, err)
		assert.True(t, storage.IsErrKeyNotFound(err))
	})
}

func TestBlockRepo_List(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)

	t.Run("returns empty list when no blocks exist", func(t *testing.T) {
		blocks, err := repo.List()
		require.NoError(t, err)
		assert.Empty(t, blocks)
	})

	t.Run("returns all blocks", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			block := model.NewBlock("user-123", "project", "", "", time.Now())
			require.NoError(t, repo.Create(block))
		}

		blocks, err := repo.List()
		require.NoError(t, err)
		assert.Len(t, blocks, 3)
	})
}

func TestBlockRepo_ListFiltered(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)

	// Create test data
	now := time.Now()
	blocks := []*model.Block{
		{OwnerKey: "user1", ProjectSID: "proj1", TaskSID: "task1", TimestampStart: now.Add(-3 * time.Hour), TimestampEnd: now.Add(-2 * time.Hour)},
		{OwnerKey: "user1", ProjectSID: "proj1", TaskSID: "task2", TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},
		{OwnerKey: "user1", ProjectSID: "proj2", TaskSID: "task1", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
	}
	for _, b := range blocks {
		require.NoError(t, repo.Create(b))
	}

	t.Run("filters by project", func(t *testing.T) {
		result, err := repo.ListFiltered(storage.BlockFilter{ProjectSID: "proj1"})
		require.NoError(t, err)
		assert.Len(t, result, 2)
		for _, b := range result {
			assert.Equal(t, "proj1", b.ProjectSID)
		}
	})

	t.Run("filters by task", func(t *testing.T) {
		result, err := repo.ListFiltered(storage.BlockFilter{TaskSID: "task1"})
		require.NoError(t, err)
		assert.Len(t, result, 2)
		for _, b := range result {
			assert.Equal(t, "task1", b.TaskSID)
		}
	})

	t.Run("filters by project and task", func(t *testing.T) {
		result, err := repo.ListFiltered(storage.BlockFilter{ProjectSID: "proj1", TaskSID: "task1"})
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "proj1", result[0].ProjectSID)
		assert.Equal(t, "task1", result[0].TaskSID)
	})

	t.Run("filters by start time", func(t *testing.T) {
		// StartAfter excludes blocks that start BEFORE the given time
		// Block at -3h: excluded (starts before -1.5h)
		// Block at -2h: excluded (starts before -1.5h)
		// Block at -1h: kept (starts after -1.5h)
		result, err := repo.ListFiltered(storage.BlockFilter{StartAfter: now.Add(-90 * time.Minute)})
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("applies limit", func(t *testing.T) {
		result, err := repo.ListFiltered(storage.BlockFilter{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("results are sorted by start time descending", func(t *testing.T) {
		result, err := repo.ListFiltered(storage.BlockFilter{})
		require.NoError(t, err)
		require.Len(t, result, 3)
		for i := 0; i < len(result)-1; i++ {
			assert.True(t, result[i].TimestampStart.After(result[i+1].TimestampStart) ||
				result[i].TimestampStart.Equal(result[i+1].TimestampStart))
		}
	})
}

func TestBlockRepo_ListByProject(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)

	// Create blocks for different projects
	require.NoError(t, repo.Create(model.NewBlock("user1", "proj1", "", "", time.Now())))
	require.NoError(t, repo.Create(model.NewBlock("user1", "proj1", "", "", time.Now())))
	require.NoError(t, repo.Create(model.NewBlock("user1", "proj2", "", "", time.Now())))

	t.Run("returns blocks for specific project", func(t *testing.T) {
		blocks, err := repo.ListByProject("proj1")
		require.NoError(t, err)
		assert.Len(t, blocks, 2)
		for _, b := range blocks {
			assert.Equal(t, "proj1", b.ProjectSID)
		}
	})

	t.Run("returns empty for non-existent project", func(t *testing.T) {
		blocks, err := repo.ListByProject("nonexistent")
		require.NoError(t, err)
		assert.Empty(t, blocks)
	})
}

func TestBlockRepo_ListByProjectAndTask(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)

	require.NoError(t, repo.Create(model.NewBlock("user1", "proj1", "task1", "", time.Now())))
	require.NoError(t, repo.Create(model.NewBlock("user1", "proj1", "task2", "", time.Now())))
	require.NoError(t, repo.Create(model.NewBlock("user1", "proj1", "task1", "", time.Now())))

	t.Run("returns blocks for specific project and task", func(t *testing.T) {
		blocks, err := repo.ListByProjectAndTask("proj1", "task1")
		require.NoError(t, err)
		assert.Len(t, blocks, 2)
	})
}

func TestBlockRepo_ListByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)

	now := time.Now()

	// Block that ends before range
	b1 := model.NewBlock("user1", "proj1", "", "", now.Add(-5*time.Hour))
	b1.TimestampEnd = now.Add(-4 * time.Hour)
	require.NoError(t, repo.Create(b1))

	// Block within range
	b2 := model.NewBlock("user1", "proj1", "", "", now.Add(-2*time.Hour))
	b2.TimestampEnd = now.Add(-1 * time.Hour)
	require.NoError(t, repo.Create(b2))

	// Block that starts after range
	b3 := model.NewBlock("user1", "proj1", "", "", now.Add(1*time.Hour))
	b3.TimestampEnd = now.Add(2 * time.Hour)
	require.NoError(t, repo.Create(b3))

	t.Run("returns blocks within time range", func(t *testing.T) {
		blocks, err := repo.ListByTimeRange(now.Add(-3*time.Hour), now)
		require.NoError(t, err)
		assert.Len(t, blocks, 1)
	})

	t.Run("includes active blocks in range", func(t *testing.T) {
		// Create active block (no end time)
		active := model.NewBlock("user1", "proj1", "", "", now.Add(-30*time.Minute))
		require.NoError(t, repo.Create(active))

		blocks, err := repo.ListByTimeRange(now.Add(-1*time.Hour), now.Add(1*time.Hour))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(blocks), 1)
	})
}

func TestBlockRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)

	t.Run("updates existing block", func(t *testing.T) {
		block := model.NewBlock("user-123", "myproject", "", "", time.Now().Add(-1*time.Hour))
		require.NoError(t, repo.Create(block))

		// Update the block
		block.Note = "updated note"
		block.TimestampEnd = time.Now()
		require.NoError(t, repo.Update(block))

		// Verify update
		retrieved, err := repo.Get(block.Key)
		require.NoError(t, err)
		assert.Equal(t, "updated note", retrieved.Note)
		assert.False(t, retrieved.TimestampEnd.IsZero())
	})
}

func TestBlockRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewBlockRepo(db)

	t.Run("deletes existing block", func(t *testing.T) {
		block := model.NewBlock("user-123", "myproject", "", "", time.Now())
		require.NoError(t, repo.Create(block))

		err := repo.Delete(block.Key)
		require.NoError(t, err)

		_, err = repo.Get(block.Key)
		assert.True(t, storage.IsErrKeyNotFound(err))
	})

	t.Run("delete non-existent key does not error", func(t *testing.T) {
		err := repo.Delete("block:nonexistent")
		assert.NoError(t, err)
	})
}

func TestBlockRepo_TotalDuration(t *testing.T) {
	now := time.Now()

	t.Run("calculates total duration of completed blocks", func(t *testing.T) {
		blocks := []*model.Block{
			{TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},   // 1 hour
			{TimestampStart: now.Add(-4 * time.Hour), TimestampEnd: now.Add(-3 * time.Hour)},   // 1 hour
			{TimestampStart: now.Add(-30 * time.Minute), TimestampEnd: now},                    // 30 min
		}

		total := storage.TotalDuration(blocks)
		assert.InDelta(t, 2.5*float64(time.Hour), float64(total), float64(time.Second))
	})

	t.Run("returns zero for empty list", func(t *testing.T) {
		total := storage.TotalDuration([]*model.Block{})
		assert.Equal(t, time.Duration(0), total)
	})
}

func TestBlockRepo_AggregateByProject(t *testing.T) {
	now := time.Now()

	blocks := []*model.Block{
		{ProjectSID: "proj1", TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},
		{ProjectSID: "proj1", TimestampStart: now.Add(-3 * time.Hour), TimestampEnd: now.Add(-2 * time.Hour)},
		{ProjectSID: "proj2", TimestampStart: now.Add(-30 * time.Minute), TimestampEnd: now},
	}

	t.Run("aggregates blocks by project", func(t *testing.T) {
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

	t.Run("results sorted by duration descending", func(t *testing.T) {
		agg := storage.AggregateByProject(blocks)
		require.Len(t, agg, 2)
		assert.GreaterOrEqual(t, agg[0].Duration, agg[1].Duration)
	})
}

// =============================================================================
// Project Repository Tests
// =============================================================================

func TestProjectRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("creates project with correct key", func(t *testing.T) {
		project := model.NewProject("myproj", "My Project", "#FF5733")
		err := repo.Create(project)
		require.NoError(t, err)

		assert.Equal(t, model.GenerateProjectKey("myproj"), project.Key)
	})

	t.Run("project persists with all fields", func(t *testing.T) {
		project := model.NewProject("testproj", "Test Project", "#00FF00")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("testproj")
		require.NoError(t, err)
		assert.Equal(t, "testproj", retrieved.SID)
		assert.Equal(t, "Test Project", retrieved.DisplayName)
		assert.Equal(t, "#00FF00", retrieved.Color)
	})
}

func TestProjectRepo_Get(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("retrieves existing project", func(t *testing.T) {
		original := model.NewProject("myproj", "My Project", "#AABBCC")
		require.NoError(t, repo.Create(original))

		retrieved, err := repo.Get("myproj")
		require.NoError(t, err)
		assert.Equal(t, original.SID, retrieved.SID)
		assert.Equal(t, original.DisplayName, retrieved.DisplayName)
		assert.Equal(t, original.Color, retrieved.Color)
	})

	t.Run("returns error for non-existent project", func(t *testing.T) {
		_, err := repo.Get("nonexistent")
		assert.Error(t, err)
		assert.True(t, storage.IsErrKeyNotFound(err))
	})
}

func TestProjectRepo_GetOrCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("creates new project when not exists", func(t *testing.T) {
		project, created, err := repo.GetOrCreate("newproj", "New Project")
		require.NoError(t, err)
		assert.True(t, created)
		assert.Equal(t, "newproj", project.SID)
		assert.Equal(t, "New Project", project.DisplayName)
	})

	t.Run("returns existing project without creating", func(t *testing.T) {
		// First create
		original := model.NewProject("existingproj", "Existing Project", "#123456")
		require.NoError(t, repo.Create(original))

		// GetOrCreate should return existing
		project, created, err := repo.GetOrCreate("existingproj", "Different Name")
		require.NoError(t, err)
		assert.False(t, created)
		assert.Equal(t, "Existing Project", project.DisplayName) // Original name preserved
	})
}

func TestProjectRepo_List(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("returns empty list when no projects exist", func(t *testing.T) {
		projects, err := repo.List()
		require.NoError(t, err)
		assert.Empty(t, projects)
	})

	t.Run("returns all projects", func(t *testing.T) {
		require.NoError(t, repo.Create(model.NewProject("proj1", "Project 1", "")))
		require.NoError(t, repo.Create(model.NewProject("proj2", "Project 2", "")))
		require.NoError(t, repo.Create(model.NewProject("proj3", "Project 3", "")))

		projects, err := repo.List()
		require.NoError(t, err)
		assert.Len(t, projects, 3)
	})
}

func TestProjectRepo_Exists(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("returns true for existing project", func(t *testing.T) {
		require.NoError(t, repo.Create(model.NewProject("exists", "Exists", "")))

		exists, err := repo.Exists("exists")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent project", func(t *testing.T) {
		exists, err := repo.Exists("nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestProjectRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("updates existing project", func(t *testing.T) {
		project := model.NewProject("myproj", "Original Name", "#000000")
		require.NoError(t, repo.Create(project))

		project.DisplayName = "Updated Name"
		project.Color = "#FFFFFF"
		require.NoError(t, repo.Update(project))

		retrieved, err := repo.Get("myproj")
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", retrieved.DisplayName)
		assert.Equal(t, "#FFFFFF", retrieved.Color)
	})
}

func TestProjectRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("deletes existing project", func(t *testing.T) {
		require.NoError(t, repo.Create(model.NewProject("todelete", "To Delete", "")))

		err := repo.Delete("todelete")
		require.NoError(t, err)

		exists, err := repo.Exists("todelete")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// =============================================================================
// Task Repository Tests
// =============================================================================

func TestTaskRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewTaskRepo(db)

	t.Run("creates task with correct key", func(t *testing.T) {
		task := model.NewTask("myproj", "mytask", "My Task", "#FF0000")
		err := repo.Create(task)
		require.NoError(t, err)

		assert.Equal(t, model.GenerateTaskKey("myproj", "mytask"), task.Key)
	})

	t.Run("task persists with all fields", func(t *testing.T) {
		task := model.NewTask("proj", "task1", "Task One", "#00FF00")
		require.NoError(t, repo.Create(task))

		retrieved, err := repo.Get("proj", "task1")
		require.NoError(t, err)
		assert.Equal(t, "task1", retrieved.SID)
		assert.Equal(t, "proj", retrieved.ProjectSID)
		assert.Equal(t, "Task One", retrieved.DisplayName)
		assert.Equal(t, "#00FF00", retrieved.Color)
	})
}

func TestTaskRepo_Get(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewTaskRepo(db)

	t.Run("retrieves existing task", func(t *testing.T) {
		original := model.NewTask("proj", "task1", "Task One", "#AABBCC")
		require.NoError(t, repo.Create(original))

		retrieved, err := repo.Get("proj", "task1")
		require.NoError(t, err)
		assert.Equal(t, original.SID, retrieved.SID)
		assert.Equal(t, original.ProjectSID, retrieved.ProjectSID)
		assert.Equal(t, original.DisplayName, retrieved.DisplayName)
	})

	t.Run("returns error for non-existent task", func(t *testing.T) {
		_, err := repo.Get("proj", "nonexistent")
		assert.Error(t, err)
		assert.True(t, storage.IsErrKeyNotFound(err))
	})
}

func TestTaskRepo_GetOrCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewTaskRepo(db)

	t.Run("creates new task when not exists", func(t *testing.T) {
		task, created, err := repo.GetOrCreate("proj", "newtask", "New Task")
		require.NoError(t, err)
		assert.True(t, created)
		assert.Equal(t, "newtask", task.SID)
		assert.Equal(t, "proj", task.ProjectSID)
	})

	t.Run("returns existing task without creating", func(t *testing.T) {
		original := model.NewTask("proj", "existingtask", "Existing Task", "#123456")
		require.NoError(t, repo.Create(original))

		task, created, err := repo.GetOrCreate("proj", "existingtask", "Different Name")
		require.NoError(t, err)
		assert.False(t, created)
		assert.Equal(t, "Existing Task", task.DisplayName) // Original preserved
	})
}

func TestTaskRepo_ListByProject(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewTaskRepo(db)

	// Create tasks for multiple projects
	require.NoError(t, repo.Create(model.NewTask("proj1", "task1", "Task 1", "")))
	require.NoError(t, repo.Create(model.NewTask("proj1", "task2", "Task 2", "")))
	require.NoError(t, repo.Create(model.NewTask("proj2", "task1", "Task 1", "")))

	t.Run("returns tasks for specific project", func(t *testing.T) {
		tasks, err := repo.ListByProject("proj1")
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
		for _, task := range tasks {
			assert.Equal(t, "proj1", task.ProjectSID)
		}
	})

	t.Run("returns empty for non-existent project", func(t *testing.T) {
		tasks, err := repo.ListByProject("nonexistent")
		require.NoError(t, err)
		assert.Empty(t, tasks)
	})
}

func TestTaskRepo_List(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewTaskRepo(db)

	t.Run("returns all tasks", func(t *testing.T) {
		require.NoError(t, repo.Create(model.NewTask("proj1", "task1", "Task 1", "")))
		require.NoError(t, repo.Create(model.NewTask("proj1", "task2", "Task 2", "")))
		require.NoError(t, repo.Create(model.NewTask("proj2", "task1", "Task 1", "")))

		tasks, err := repo.List()
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
	})
}

func TestTaskRepo_Exists(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewTaskRepo(db)

	t.Run("returns true for existing task", func(t *testing.T) {
		require.NoError(t, repo.Create(model.NewTask("proj", "exists", "Exists", "")))

		exists, err := repo.Exists("proj", "exists")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent task", func(t *testing.T) {
		exists, err := repo.Exists("proj", "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestTaskRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewTaskRepo(db)

	t.Run("updates existing task", func(t *testing.T) {
		task := model.NewTask("proj", "mytask", "Original", "#000000")
		require.NoError(t, repo.Create(task))

		task.DisplayName = "Updated"
		task.Color = "#FFFFFF"
		require.NoError(t, repo.Update(task))

		retrieved, err := repo.Get("proj", "mytask")
		require.NoError(t, err)
		assert.Equal(t, "Updated", retrieved.DisplayName)
		assert.Equal(t, "#FFFFFF", retrieved.Color)
	})
}

func TestTaskRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewTaskRepo(db)

	t.Run("deletes existing task", func(t *testing.T) {
		require.NoError(t, repo.Create(model.NewTask("proj", "todelete", "To Delete", "")))

		err := repo.Delete("proj", "todelete")
		require.NoError(t, err)

		exists, err := repo.Exists("proj", "todelete")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// =============================================================================
// Config Repository Tests
// =============================================================================

func TestConfigRepo_Get(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewConfigRepo(db)

	t.Run("creates config on first access", func(t *testing.T) {
		config, err := repo.Get()
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, model.KeyConfig, config.Key)
		assert.NotEmpty(t, config.UserKey)
	})

	t.Run("returns same config on subsequent access", func(t *testing.T) {
		config1, err := repo.Get()
		require.NoError(t, err)

		config2, err := repo.Get()
		require.NoError(t, err)

		assert.Equal(t, config1.UserKey, config2.UserKey)
	})
}

func TestConfigRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewConfigRepo(db)

	t.Run("updates config", func(t *testing.T) {
		config, err := repo.Get()
		require.NoError(t, err)

		newUserKey := "custom-user-key"
		config.UserKey = newUserKey
		require.NoError(t, repo.Update(config))

		retrieved, err := repo.Get()
		require.NoError(t, err)
		assert.Equal(t, newUserKey, retrieved.UserKey)
	})
}

// =============================================================================
// ActiveBlock Repository Tests
// =============================================================================

func TestActiveBlockRepo_Get(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewActiveBlockRepo(db)

	t.Run("creates active block on first access", func(t *testing.T) {
		active, err := repo.Get()
		require.NoError(t, err)
		assert.NotNil(t, active)
		assert.Equal(t, model.KeyActiveBlock, active.Key)
		assert.False(t, active.IsTracking())
	})
}

func TestActiveBlockRepo_SetActive(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewActiveBlockRepo(db)

	t.Run("sets active block", func(t *testing.T) {
		err := repo.SetActive("block:123")
		require.NoError(t, err)

		active, err := repo.Get()
		require.NoError(t, err)
		assert.Equal(t, "block:123", active.ActiveBlockKey)
		assert.True(t, active.IsTracking())
	})

	t.Run("moves previous active to previous", func(t *testing.T) {
		require.NoError(t, repo.SetActive("block:first"))
		require.NoError(t, repo.SetActive("block:second"))

		active, err := repo.Get()
		require.NoError(t, err)
		assert.Equal(t, "block:second", active.ActiveBlockKey)
		assert.Equal(t, "block:first", active.PreviousBlockKey)
	})
}

func TestActiveBlockRepo_ClearActive(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewActiveBlockRepo(db)

	t.Run("clears active block and saves to previous", func(t *testing.T) {
		require.NoError(t, repo.SetActive("block:123"))
		require.NoError(t, repo.ClearActive())

		active, err := repo.Get()
		require.NoError(t, err)
		assert.Empty(t, active.ActiveBlockKey)
		assert.Equal(t, "block:123", active.PreviousBlockKey)
		assert.False(t, active.IsTracking())
	})
}

func TestActiveBlockRepo_GetActiveBlock(t *testing.T) {
	db := setupTestDB(t)
	activeRepo := storage.NewActiveBlockRepo(db)
	blockRepo := storage.NewBlockRepo(db)

	t.Run("returns nil when not tracking", func(t *testing.T) {
		block, err := activeRepo.GetActiveBlock(blockRepo)
		require.NoError(t, err)
		assert.Nil(t, block)
	})

	t.Run("returns active block when tracking", func(t *testing.T) {
		// Create a block
		newBlock := model.NewBlock("user1", "proj1", "", "", time.Now())
		require.NoError(t, blockRepo.Create(newBlock))

		// Set it as active
		require.NoError(t, activeRepo.SetActive(newBlock.Key))

		// Retrieve active block
		block, err := activeRepo.GetActiveBlock(blockRepo)
		require.NoError(t, err)
		assert.NotNil(t, block)
		assert.Equal(t, newBlock.Key, block.Key)
	})
}

func TestActiveBlockRepo_GetPreviousBlock(t *testing.T) {
	db := setupTestDB(t)
	activeRepo := storage.NewActiveBlockRepo(db)
	blockRepo := storage.NewBlockRepo(db)

	t.Run("returns nil when no previous", func(t *testing.T) {
		block, err := activeRepo.GetPreviousBlock(blockRepo)
		require.NoError(t, err)
		assert.Nil(t, block)
	})

	t.Run("returns previous block", func(t *testing.T) {
		// Create blocks
		block1 := model.NewBlock("user1", "proj1", "", "", time.Now())
		require.NoError(t, blockRepo.Create(block1))
		block2 := model.NewBlock("user1", "proj1", "", "", time.Now())
		require.NoError(t, blockRepo.Create(block2))

		// Set first as active, then second
		require.NoError(t, activeRepo.SetActive(block1.Key))
		require.NoError(t, activeRepo.SetActive(block2.Key))

		// Get previous
		prev, err := activeRepo.GetPreviousBlock(blockRepo)
		require.NoError(t, err)
		assert.NotNil(t, prev)
		assert.Equal(t, block1.Key, prev.Key)
	})
}

// =============================================================================
// Goal Repository Tests
// =============================================================================

func TestGoalRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewGoalRepo(db)

	t.Run("creates goal with correct key", func(t *testing.T) {
		goal := model.NewGoal("myproj", model.GoalTypeDaily, 8*time.Hour)
		err := repo.Create(goal)
		require.NoError(t, err)

		assert.Equal(t, model.GenerateGoalKey("myproj"), goal.Key)
	})

	t.Run("goal persists with all fields", func(t *testing.T) {
		goal := model.NewGoal("testproj", model.GoalTypeWeekly, 40*time.Hour)
		require.NoError(t, repo.Create(goal))

		retrieved, err := repo.Get("testproj")
		require.NoError(t, err)
		assert.Equal(t, "testproj", retrieved.ProjectSID)
		assert.Equal(t, model.GoalTypeWeekly, retrieved.Type)
		assert.Equal(t, 40*time.Hour, retrieved.Target)
	})
}

func TestGoalRepo_Get(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewGoalRepo(db)

	t.Run("retrieves existing goal", func(t *testing.T) {
		original := model.NewGoal("proj", model.GoalTypeDaily, 4*time.Hour)
		require.NoError(t, repo.Create(original))

		retrieved, err := repo.Get("proj")
		require.NoError(t, err)
		assert.Equal(t, original.ProjectSID, retrieved.ProjectSID)
		assert.Equal(t, original.Type, retrieved.Type)
		assert.Equal(t, original.Target, retrieved.Target)
	})

	t.Run("returns error for non-existent goal", func(t *testing.T) {
		_, err := repo.Get("nonexistent")
		assert.Error(t, err)
		assert.True(t, storage.IsErrKeyNotFound(err))
	})
}

func TestGoalRepo_List(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewGoalRepo(db)

	t.Run("returns empty list when no goals exist", func(t *testing.T) {
		goals, err := repo.List()
		require.NoError(t, err)
		assert.Empty(t, goals)
	})

	t.Run("returns all goals", func(t *testing.T) {
		require.NoError(t, repo.Create(model.NewGoal("proj1", model.GoalTypeDaily, 8*time.Hour)))
		require.NoError(t, repo.Create(model.NewGoal("proj2", model.GoalTypeWeekly, 40*time.Hour)))

		goals, err := repo.List()
		require.NoError(t, err)
		assert.Len(t, goals, 2)
	})
}

func TestGoalRepo_Exists(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewGoalRepo(db)

	t.Run("returns true for existing goal", func(t *testing.T) {
		require.NoError(t, repo.Create(model.NewGoal("exists", model.GoalTypeDaily, 8*time.Hour)))

		exists, err := repo.Exists("exists")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent goal", func(t *testing.T) {
		exists, err := repo.Exists("nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestGoalRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewGoalRepo(db)

	t.Run("updates existing goal", func(t *testing.T) {
		goal := model.NewGoal("proj", model.GoalTypeDaily, 4*time.Hour)
		require.NoError(t, repo.Create(goal))

		goal.Type = model.GoalTypeWeekly
		goal.Target = 20 * time.Hour
		require.NoError(t, repo.Update(goal))

		retrieved, err := repo.Get("proj")
		require.NoError(t, err)
		assert.Equal(t, model.GoalTypeWeekly, retrieved.Type)
		assert.Equal(t, 20*time.Hour, retrieved.Target)
	})
}

func TestGoalRepo_Upsert(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewGoalRepo(db)

	t.Run("creates goal if not exists", func(t *testing.T) {
		goal := model.NewGoal("newproj", model.GoalTypeDaily, 6*time.Hour)
		require.NoError(t, repo.Upsert(goal))

		retrieved, err := repo.Get("newproj")
		require.NoError(t, err)
		assert.Equal(t, 6*time.Hour, retrieved.Target)
	})

	t.Run("updates goal if exists", func(t *testing.T) {
		goal := model.NewGoal("existproj", model.GoalTypeDaily, 4*time.Hour)
		require.NoError(t, repo.Create(goal))

		updatedGoal := model.NewGoal("existproj", model.GoalTypeWeekly, 30*time.Hour)
		require.NoError(t, repo.Upsert(updatedGoal))

		retrieved, err := repo.Get("existproj")
		require.NoError(t, err)
		assert.Equal(t, model.GoalTypeWeekly, retrieved.Type)
		assert.Equal(t, 30*time.Hour, retrieved.Target)
	})
}

func TestGoalRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewGoalRepo(db)

	t.Run("deletes existing goal", func(t *testing.T) {
		require.NoError(t, repo.Create(model.NewGoal("todelete", model.GoalTypeDaily, 8*time.Hour)))

		err := repo.Delete("todelete")
		require.NoError(t, err)

		exists, err := repo.Exists("todelete")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("delete non-existent goal does not error", func(t *testing.T) {
		err := repo.Delete("nonexistent")
		assert.NoError(t, err)
	})
}

func TestGoal_CalculateProgress(t *testing.T) {
	goal := model.NewGoal("proj", model.GoalTypeDaily, 8*time.Hour)

	t.Run("calculates progress correctly", func(t *testing.T) {
		progress := goal.CalculateProgress(4 * time.Hour)
		assert.Equal(t, 4*time.Hour, progress.Current)
		assert.Equal(t, 4*time.Hour, progress.Remaining)
		assert.InDelta(t, 50.0, progress.Percentage, 0.1)
		assert.False(t, progress.IsComplete)
	})

	t.Run("handles completion", func(t *testing.T) {
		progress := goal.CalculateProgress(8 * time.Hour)
		assert.True(t, progress.IsComplete)
		assert.Equal(t, time.Duration(0), progress.Remaining)
	})

	t.Run("handles over-completion", func(t *testing.T) {
		progress := goal.CalculateProgress(10 * time.Hour)
		assert.True(t, progress.IsComplete)
		assert.Equal(t, time.Duration(0), progress.Remaining)
		assert.Greater(t, progress.Percentage, 100.0)
	})
}

// =============================================================================
// CRUD and Error Handling Tests
// =============================================================================

func TestDB_CRUD_ErrorHandling(t *testing.T) {
	db := setupTestDB(t)

	t.Run("IsErrKeyNotFound identifies key not found errors", func(t *testing.T) {
		project := &model.Project{}
		err := db.Get("nonexistent", project)
		assert.True(t, storage.IsErrKeyNotFound(err))
	})

	t.Run("Exists returns false for missing key", func(t *testing.T) {
		exists, err := db.Exists("nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Set and Get work with model interface", func(t *testing.T) {
		project := model.NewProject("testsid", "Test", "#000000")
		require.NoError(t, db.Set(project))

		retrieved := &model.Project{}
		require.NoError(t, db.Get(project.GetKey(), retrieved))
		assert.Equal(t, project.SID, retrieved.SID)
	})

	t.Run("SetBytes and GetBytes work with raw data", func(t *testing.T) {
		data := []byte("raw data content")
		require.NoError(t, db.SetBytes("raw:key", data))

		retrieved, err := db.GetBytes("raw:key")
		require.NoError(t, err)
		assert.Equal(t, data, retrieved)
	})

	t.Run("ListByPrefix returns keys with prefix", func(t *testing.T) {
		// Create some projects
		require.NoError(t, db.Set(model.NewProject("list1", "List 1", "")))
		require.NoError(t, db.Set(model.NewProject("list2", "List 2", "")))

		keys, err := db.ListByPrefix(model.PrefixProject + ":")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(keys), 2)
	})

	t.Run("Delete removes key", func(t *testing.T) {
		project := model.NewProject("delme", "Delete Me", "")
		require.NoError(t, db.Set(project))

		require.NoError(t, db.Delete(project.GetKey()))

		exists, err := db.Exists(project.GetKey())
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
