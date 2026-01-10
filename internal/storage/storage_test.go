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
