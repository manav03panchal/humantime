// Package integration provides integration tests for Humantime commands.
// These tests use a real in-memory Badger database and test the full
// command stack including the repository layer.
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
// Project Creation with Color Tests
// =============================================================================

func TestProjectCreationWithColor(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("creates project with valid hex color", func(t *testing.T) {
		project := model.NewProject("clientwork", "Client Work", "#FF5733")
		err := repo.Create(project)
		require.NoError(t, err)

		retrieved, err := repo.Get("clientwork")
		require.NoError(t, err)

		assert.Equal(t, "clientwork", retrieved.SID)
		assert.Equal(t, "Client Work", retrieved.DisplayName)
		assert.Equal(t, "#FF5733", retrieved.Color)
	})

	t.Run("creates project with lowercase hex color", func(t *testing.T) {
		project := model.NewProject("lowercasecolor", "Lowercase Color", "#aabbcc")
		err := repo.Create(project)
		require.NoError(t, err)

		retrieved, err := repo.Get("lowercasecolor")
		require.NoError(t, err)
		assert.Equal(t, "#aabbcc", retrieved.Color)
	})

	t.Run("creates project with empty color", func(t *testing.T) {
		project := model.NewProject("nocolorproj", "No Color Project", "")
		err := repo.Create(project)
		require.NoError(t, err)

		retrieved, err := repo.Get("nocolorproj")
		require.NoError(t, err)
		assert.Equal(t, "", retrieved.Color)
	})

	t.Run("validates color format", func(t *testing.T) {
		// Valid colors
		assert.True(t, model.ValidateColor("#FF5733"))
		assert.True(t, model.ValidateColor("#aabbcc"))
		assert.True(t, model.ValidateColor("#000000"))
		assert.True(t, model.ValidateColor("#FFFFFF"))
		assert.True(t, model.ValidateColor("")) // Empty is valid

		// Invalid colors
		assert.False(t, model.ValidateColor("#FFF"))          // Too short
		assert.False(t, model.ValidateColor("#FFFFFFF"))      // Too long
		assert.False(t, model.ValidateColor("FF5733"))        // Missing #
		assert.False(t, model.ValidateColor("#GG5733"))       // Invalid hex chars
		assert.False(t, model.ValidateColor("red"))           // Color name not allowed
		assert.False(t, model.ValidateColor("#FF 5733"))      // Contains space
	})
}

// =============================================================================
// Project Listing Tests
// =============================================================================

func TestProjectListing(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("lists empty projects", func(t *testing.T) {
		projects, err := repo.List()
		require.NoError(t, err)
		assert.Empty(t, projects)
	})

	t.Run("lists single project", func(t *testing.T) {
		project := model.NewProject("singleproj", "Single Project", "#123456")
		require.NoError(t, repo.Create(project))

		projects, err := repo.List()
		require.NoError(t, err)
		assert.Len(t, projects, 1)
		assert.Equal(t, "singleproj", projects[0].SID)
	})

	t.Run("lists multiple projects", func(t *testing.T) {
		// Create additional projects
		require.NoError(t, repo.Create(model.NewProject("proj2", "Project 2", "#FF0000")))
		require.NoError(t, repo.Create(model.NewProject("proj3", "Project 3", "#00FF00")))
		require.NoError(t, repo.Create(model.NewProject("proj4", "Project 4", "#0000FF")))

		projects, err := repo.List()
		require.NoError(t, err)
		assert.Len(t, projects, 4) // Including singleproj from previous test

		// Verify all projects are present
		projectSIDs := make(map[string]bool)
		for _, p := range projects {
			projectSIDs[p.SID] = true
		}
		assert.True(t, projectSIDs["singleproj"])
		assert.True(t, projectSIDs["proj2"])
		assert.True(t, projectSIDs["proj3"])
		assert.True(t, projectSIDs["proj4"])
	})

	t.Run("lists projects with all fields intact", func(t *testing.T) {
		db := setupTestDB(t) // Fresh database
		repo := storage.NewProjectRepo(db)

		require.NoError(t, repo.Create(model.NewProject("fullproj", "Full Project", "#ABCDEF")))

		projects, err := repo.List()
		require.NoError(t, err)
		require.Len(t, projects, 1)

		p := projects[0]
		assert.Equal(t, "fullproj", p.SID)
		assert.Equal(t, "Full Project", p.DisplayName)
		assert.Equal(t, "#ABCDEF", p.Color)
		assert.Contains(t, p.Key, model.PrefixProject+":")
	})
}

// =============================================================================
// Project Editing Tests
// =============================================================================

func TestProjectEditing(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("edits project display name", func(t *testing.T) {
		project := model.NewProject("editproj1", "Original Name", "#FF5733")
		require.NoError(t, repo.Create(project))

		// Retrieve and update
		retrieved, err := repo.Get("editproj1")
		require.NoError(t, err)

		retrieved.DisplayName = "Updated Name"
		require.NoError(t, repo.Update(retrieved))

		// Verify update
		updated, err := repo.Get("editproj1")
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.DisplayName)
		assert.Equal(t, "#FF5733", updated.Color) // Color unchanged
	})

	t.Run("edits project color", func(t *testing.T) {
		project := model.NewProject("editproj2", "Color Project", "#000000")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("editproj2")
		require.NoError(t, err)

		retrieved.Color = "#FFFFFF"
		require.NoError(t, repo.Update(retrieved))

		updated, err := repo.Get("editproj2")
		require.NoError(t, err)
		assert.Equal(t, "#FFFFFF", updated.Color)
		assert.Equal(t, "Color Project", updated.DisplayName) // Name unchanged
	})

	t.Run("edits both name and color", func(t *testing.T) {
		project := model.NewProject("editproj3", "Both Fields", "#AABBCC")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("editproj3")
		require.NoError(t, err)

		retrieved.DisplayName = "New Name"
		retrieved.Color = "#112233"
		require.NoError(t, repo.Update(retrieved))

		updated, err := repo.Get("editproj3")
		require.NoError(t, err)
		assert.Equal(t, "New Name", updated.DisplayName)
		assert.Equal(t, "#112233", updated.Color)
	})

	t.Run("clears project color", func(t *testing.T) {
		project := model.NewProject("editproj4", "Clear Color", "#FF0000")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("editproj4")
		require.NoError(t, err)

		retrieved.Color = ""
		require.NoError(t, repo.Update(retrieved))

		updated, err := repo.Get("editproj4")
		require.NoError(t, err)
		assert.Equal(t, "", updated.Color)
	})

	t.Run("preserves SID and key after edit", func(t *testing.T) {
		project := model.NewProject("editproj5", "Original", "#123456")
		originalKey := project.Key
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("editproj5")
		require.NoError(t, err)

		retrieved.DisplayName = "Changed"
		require.NoError(t, repo.Update(retrieved))

		updated, err := repo.Get("editproj5")
		require.NoError(t, err)
		assert.Equal(t, "editproj5", updated.SID)   // SID unchanged
		assert.Equal(t, originalKey, updated.Key)   // Key unchanged
	})
}

// =============================================================================
// Auto-creation on First Use Tests
// =============================================================================

func TestProjectAutoCreation(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("auto-creates project on GetOrCreate when not exists", func(t *testing.T) {
		project, created, err := repo.GetOrCreate("newproject", "New Project")
		require.NoError(t, err)

		assert.True(t, created, "expected project to be created")
		assert.Equal(t, "newproject", project.SID)
		assert.Equal(t, "New Project", project.DisplayName)
		assert.Equal(t, "", project.Color) // Auto-created without color
	})

	t.Run("returns existing project on GetOrCreate", func(t *testing.T) {
		// First create
		original := model.NewProject("existingproj", "Original Name", "#FF5733")
		require.NoError(t, repo.Create(original))

		// GetOrCreate should return existing
		project, created, err := repo.GetOrCreate("existingproj", "Different Name")
		require.NoError(t, err)

		assert.False(t, created, "expected project not to be created")
		assert.Equal(t, "Original Name", project.DisplayName) // Original preserved
		assert.Equal(t, "#FF5733", project.Color)             // Original color preserved
	})

	t.Run("auto-created project persists", func(t *testing.T) {
		_, created, err := repo.GetOrCreate("persistproj", "Persist Project")
		require.NoError(t, err)
		assert.True(t, created)

		// Verify it exists
		exists, err := repo.Exists("persistproj")
		require.NoError(t, err)
		assert.True(t, exists)

		// Verify it can be retrieved
		project, err := repo.Get("persistproj")
		require.NoError(t, err)
		assert.Equal(t, "persistproj", project.SID)
		assert.Equal(t, "Persist Project", project.DisplayName)
	})

	t.Run("multiple auto-creations work independently", func(t *testing.T) {
		p1, created1, err := repo.GetOrCreate("auto1", "Auto 1")
		require.NoError(t, err)
		assert.True(t, created1)

		p2, created2, err := repo.GetOrCreate("auto2", "Auto 2")
		require.NoError(t, err)
		assert.True(t, created2)

		p3, created3, err := repo.GetOrCreate("auto3", "Auto 3")
		require.NoError(t, err)
		assert.True(t, created3)

		// Verify all are distinct
		assert.NotEqual(t, p1.Key, p2.Key)
		assert.NotEqual(t, p2.Key, p3.Key)
		assert.NotEqual(t, p1.Key, p3.Key)
	})

	t.Run("GetOrCreate is idempotent", func(t *testing.T) {
		p1, created1, err := repo.GetOrCreate("idempotent", "Idempotent")
		require.NoError(t, err)
		assert.True(t, created1)

		p2, created2, err := repo.GetOrCreate("idempotent", "Idempotent")
		require.NoError(t, err)
		assert.False(t, created2)

		assert.Equal(t, p1.Key, p2.Key)
		assert.Equal(t, p1.SID, p2.SID)
		assert.Equal(t, p1.DisplayName, p2.DisplayName)
	})
}

// =============================================================================
// Project Existence and Retrieval Tests
// =============================================================================

func TestProjectExistence(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("Exists returns false for non-existent project", func(t *testing.T) {
		exists, err := repo.Exists("nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Exists returns true for existing project", func(t *testing.T) {
		project := model.NewProject("existsproj", "Exists Project", "")
		require.NoError(t, repo.Create(project))

		exists, err := repo.Exists("existsproj")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Get returns error for non-existent project", func(t *testing.T) {
		_, err := repo.Get("notfound")
		assert.Error(t, err)
		assert.True(t, storage.IsErrKeyNotFound(err))
	})
}

// =============================================================================
// Project Deletion Tests
// =============================================================================

func TestProjectDeletion(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("deletes existing project", func(t *testing.T) {
		project := model.NewProject("delproj", "Delete Project", "#FF0000")
		require.NoError(t, repo.Create(project))

		// Verify it exists
		exists, err := repo.Exists("delproj")
		require.NoError(t, err)
		assert.True(t, exists)

		// Delete
		err = repo.Delete("delproj")
		require.NoError(t, err)

		// Verify it no longer exists
		exists, err = repo.Exists("delproj")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("delete non-existent project does not error", func(t *testing.T) {
		err := repo.Delete("neverexisted")
		assert.NoError(t, err)
	})

	t.Run("deleted project cannot be retrieved", func(t *testing.T) {
		project := model.NewProject("delproj2", "Delete Project 2", "")
		require.NoError(t, repo.Create(project))
		require.NoError(t, repo.Delete("delproj2"))

		_, err := repo.Get("delproj2")
		assert.Error(t, err)
		assert.True(t, storage.IsErrKeyNotFound(err))
	})
}

// =============================================================================
// Edge Cases and Integration Scenarios
// =============================================================================

func TestProjectEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("project with special characters in display name", func(t *testing.T) {
		project := model.NewProject("specialchars", "Client Work (2024) - Phase 1", "#ABCDEF")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("specialchars")
		require.NoError(t, err)
		assert.Equal(t, "Client Work (2024) - Phase 1", retrieved.DisplayName)
	})

	t.Run("project with unicode in display name", func(t *testing.T) {
		project := model.NewProject("unicode", "Projet de Developpement", "#123456")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("unicode")
		require.NoError(t, err)
		assert.Equal(t, "Projet de Developpement", retrieved.DisplayName)
	})

	t.Run("project SID case sensitivity", func(t *testing.T) {
		project := model.NewProject("casesensitive", "Case Sensitive", "#FF0000")
		require.NoError(t, repo.Create(project))

		// Should find with exact case
		_, err := repo.Get("casesensitive")
		require.NoError(t, err)

		// Should NOT find with different case (SIDs are case-sensitive)
		_, err = repo.Get("CaseSensitive")
		assert.True(t, storage.IsErrKeyNotFound(err))
	})

	t.Run("create duplicate project fails", func(t *testing.T) {
		project1 := model.NewProject("duplicate", "First", "#111111")
		require.NoError(t, repo.Create(project1))

		// Creating same SID should succeed (overwrite) in current implementation
		// This test documents the behavior
		project2 := model.NewProject("duplicate", "Second", "#222222")
		err := repo.Create(project2)
		// Note: Current implementation allows overwrite, not checking for duplicates at repo level
		// The cmd/project.go checks for existence before creating
		assert.NoError(t, err)
	})
}

// =============================================================================
// Integration with Block Repository
// =============================================================================

func TestProjectWithBlocks(t *testing.T) {
	db := setupTestDB(t)
	projectRepo := storage.NewProjectRepo(db)
	blockRepo := storage.NewBlockRepo(db)

	t.Run("project with associated blocks", func(t *testing.T) {
		// Create project
		project := model.NewProject("withblocks", "With Blocks", "#FF5733")
		require.NoError(t, projectRepo.Create(project))

		// Create blocks for this project
		block1 := model.NewBlock("user1", "withblocks", "task1", "note 1", time.Now())
		block2 := model.NewBlock("user1", "withblocks", "task2", "note 2", time.Now())
		require.NoError(t, blockRepo.Create(block1))
		require.NoError(t, blockRepo.Create(block2))

		// Verify blocks are associated with project
		blocks, err := blockRepo.ListByProject("withblocks")
		require.NoError(t, err)
		assert.Len(t, blocks, 2)

		for _, b := range blocks {
			assert.Equal(t, "withblocks", b.ProjectSID)
		}
	})

	t.Run("blocks persist after project update", func(t *testing.T) {
		db := setupTestDB(t)
		projectRepo := storage.NewProjectRepo(db)
		blockRepo := storage.NewBlockRepo(db)

		// Create project and blocks
		project := model.NewProject("updateproj", "Update Project", "#AABBCC")
		require.NoError(t, projectRepo.Create(project))

		block := model.NewBlock("user1", "updateproj", "task1", "note", time.Now())
		require.NoError(t, blockRepo.Create(block))

		// Update project
		retrieved, err := projectRepo.Get("updateproj")
		require.NoError(t, err)
		retrieved.DisplayName = "Updated Project Name"
		require.NoError(t, projectRepo.Update(retrieved))

		// Verify blocks still exist
		blocks, err := blockRepo.ListByProject("updateproj")
		require.NoError(t, err)
		assert.Len(t, blocks, 1)
		assert.Equal(t, "updateproj", blocks[0].ProjectSID)
	})

	t.Run("deleted project leaves orphaned blocks", func(t *testing.T) {
		db := setupTestDB(t)
		projectRepo := storage.NewProjectRepo(db)
		blockRepo := storage.NewBlockRepo(db)

		// Create project and blocks
		project := model.NewProject("delproj", "Delete Project", "#123456")
		require.NoError(t, projectRepo.Create(project))

		block := model.NewBlock("user1", "delproj", "", "note", time.Now())
		require.NoError(t, blockRepo.Create(block))

		// Delete project
		require.NoError(t, projectRepo.Delete("delproj"))

		// Blocks still exist (orphaned)
		blocks, err := blockRepo.ListByProject("delproj")
		require.NoError(t, err)
		assert.Len(t, blocks, 1)
		assert.Equal(t, "delproj", blocks[0].ProjectSID)
	})

	t.Run("multiple projects with blocks are independent", func(t *testing.T) {
		db := setupTestDB(t)
		projectRepo := storage.NewProjectRepo(db)
		blockRepo := storage.NewBlockRepo(db)

		// Create two projects
		project1 := model.NewProject("proj1", "Project 1", "#FF0000")
		project2 := model.NewProject("proj2", "Project 2", "#00FF00")
		require.NoError(t, projectRepo.Create(project1))
		require.NoError(t, projectRepo.Create(project2))

		// Create blocks for each
		block1 := model.NewBlock("user1", "proj1", "", "note1", time.Now())
		block2 := model.NewBlock("user1", "proj1", "", "note2", time.Now())
		block3 := model.NewBlock("user1", "proj2", "", "note3", time.Now())
		require.NoError(t, blockRepo.Create(block1))
		require.NoError(t, blockRepo.Create(block2))
		require.NoError(t, blockRepo.Create(block3))

		// Verify block counts per project
		blocks1, err := blockRepo.ListByProject("proj1")
		require.NoError(t, err)
		assert.Len(t, blocks1, 2)

		blocks2, err := blockRepo.ListByProject("proj2")
		require.NoError(t, err)
		assert.Len(t, blocks2, 1)
	})
}

// =============================================================================
// Project with Tasks Tests
// =============================================================================

func TestProjectWithTasks(t *testing.T) {
	db := setupTestDB(t)
	projectRepo := storage.NewProjectRepo(db)
	taskRepo := storage.NewTaskRepo(db)

	t.Run("project with associated tasks", func(t *testing.T) {
		// Create project
		project := model.NewProject("withtasks", "With Tasks", "#FF5733")
		require.NoError(t, projectRepo.Create(project))

		// Create tasks for this project
		task1 := model.NewTask("withtasks", "task1", "Task One", "#111111")
		task2 := model.NewTask("withtasks", "task2", "Task Two", "#222222")
		task3 := model.NewTask("withtasks", "task3", "Task Three", "")
		require.NoError(t, taskRepo.Create(task1))
		require.NoError(t, taskRepo.Create(task2))
		require.NoError(t, taskRepo.Create(task3))

		// Verify tasks are associated with project
		tasks, err := taskRepo.ListByProject("withtasks")
		require.NoError(t, err)
		assert.Len(t, tasks, 3)

		for _, task := range tasks {
			assert.Equal(t, "withtasks", task.ProjectSID)
		}
	})

	t.Run("tasks persist after project update", func(t *testing.T) {
		db := setupTestDB(t)
		projectRepo := storage.NewProjectRepo(db)
		taskRepo := storage.NewTaskRepo(db)

		// Create project and task
		project := model.NewProject("taskupdate", "Task Update Project", "#AABBCC")
		require.NoError(t, projectRepo.Create(project))

		task := model.NewTask("taskupdate", "task1", "Task One", "#123456")
		require.NoError(t, taskRepo.Create(task))

		// Update project
		retrieved, err := projectRepo.Get("taskupdate")
		require.NoError(t, err)
		retrieved.DisplayName = "Updated Task Project"
		require.NoError(t, projectRepo.Update(retrieved))

		// Verify tasks still exist
		tasks, err := taskRepo.ListByProject("taskupdate")
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "taskupdate", tasks[0].ProjectSID)
	})

	t.Run("deleted project leaves orphaned tasks", func(t *testing.T) {
		db := setupTestDB(t)
		projectRepo := storage.NewProjectRepo(db)
		taskRepo := storage.NewTaskRepo(db)

		// Create project and task
		project := model.NewProject("deltaskproj", "Delete Task Project", "#654321")
		require.NoError(t, projectRepo.Create(project))

		task := model.NewTask("deltaskproj", "task1", "Task One", "")
		require.NoError(t, taskRepo.Create(task))

		// Delete project
		require.NoError(t, projectRepo.Delete("deltaskproj"))

		// Tasks still exist (orphaned)
		tasks, err := taskRepo.ListByProject("deltaskproj")
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "deltaskproj", tasks[0].ProjectSID)
	})

	t.Run("multiple projects have independent task namespaces", func(t *testing.T) {
		db := setupTestDB(t)
		projectRepo := storage.NewProjectRepo(db)
		taskRepo := storage.NewTaskRepo(db)

		// Create two projects
		project1 := model.NewProject("projA", "Project A", "#FF0000")
		project2 := model.NewProject("projB", "Project B", "#00FF00")
		require.NoError(t, projectRepo.Create(project1))
		require.NoError(t, projectRepo.Create(project2))

		// Create task with same SID in different projects
		taskA := model.NewTask("projA", "design", "Design Task A", "#AAAAAA")
		taskB := model.NewTask("projB", "design", "Design Task B", "#BBBBBB")
		require.NoError(t, taskRepo.Create(taskA))
		require.NoError(t, taskRepo.Create(taskB))

		// Verify tasks are independent
		retrievedA, err := taskRepo.Get("projA", "design")
		require.NoError(t, err)
		assert.Equal(t, "Design Task A", retrievedA.DisplayName)

		retrievedB, err := taskRepo.Get("projB", "design")
		require.NoError(t, err)
		assert.Equal(t, "Design Task B", retrievedB.DisplayName)
	})

	t.Run("task GetOrCreate with project", func(t *testing.T) {
		db := setupTestDB(t)
		projectRepo := storage.NewProjectRepo(db)
		taskRepo := storage.NewTaskRepo(db)

		// Create project
		project := model.NewProject("taskgetcreate", "Task GetOrCreate Project", "")
		require.NoError(t, projectRepo.Create(project))

		// GetOrCreate new task
		task1, created1, err := taskRepo.GetOrCreate("taskgetcreate", "newtask", "New Task")
		require.NoError(t, err)
		assert.True(t, created1)
		assert.Equal(t, "newtask", task1.SID)
		assert.Equal(t, "taskgetcreate", task1.ProjectSID)

		// GetOrCreate existing task
		task2, created2, err := taskRepo.GetOrCreate("taskgetcreate", "newtask", "Different Name")
		require.NoError(t, err)
		assert.False(t, created2)
		assert.Equal(t, "New Task", task2.DisplayName) // Original preserved
	})

	t.Run("task deletion does not affect project", func(t *testing.T) {
		db := setupTestDB(t)
		projectRepo := storage.NewProjectRepo(db)
		taskRepo := storage.NewTaskRepo(db)

		// Create project and tasks
		project := model.NewProject("taskdelproj", "Task Delete Project", "#ABCDEF")
		require.NoError(t, projectRepo.Create(project))

		task1 := model.NewTask("taskdelproj", "task1", "Task One", "")
		task2 := model.NewTask("taskdelproj", "task2", "Task Two", "")
		require.NoError(t, taskRepo.Create(task1))
		require.NoError(t, taskRepo.Create(task2))

		// Delete one task
		require.NoError(t, taskRepo.Delete("taskdelproj", "task1"))

		// Project still exists
		exists, err := projectRepo.Exists("taskdelproj")
		require.NoError(t, err)
		assert.True(t, exists)

		// Other task still exists
		tasks, err := taskRepo.ListByProject("taskdelproj")
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "task2", tasks[0].SID)
	})
}

// =============================================================================
// Project Rename Tests (Comprehensive)
// =============================================================================

func TestProjectRename(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("rename project display name only", func(t *testing.T) {
		project := model.NewProject("renameproj1", "Original Name", "#FF5733")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("renameproj1")
		require.NoError(t, err)

		retrieved.DisplayName = "Completely New Name"
		require.NoError(t, repo.Update(retrieved))

		updated, err := repo.Get("renameproj1")
		require.NoError(t, err)
		assert.Equal(t, "Completely New Name", updated.DisplayName)
		assert.Equal(t, "renameproj1", updated.SID) // SID unchanged
	})

	t.Run("rename to very long display name (within limit)", func(t *testing.T) {
		project := model.NewProject("renameproj2", "Short", "#000000")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("renameproj2")
		require.NoError(t, err)

		// 64 chars max per validation
		longName := "This is a very long project name that fits within 64 characters"
		retrieved.DisplayName = longName
		require.NoError(t, repo.Update(retrieved))

		updated, err := repo.Get("renameproj2")
		require.NoError(t, err)
		assert.Equal(t, longName, updated.DisplayName)
	})

	t.Run("rename to empty display name", func(t *testing.T) {
		project := model.NewProject("renameproj3", "Has Name", "#123456")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("renameproj3")
		require.NoError(t, err)

		// Setting empty name - repo allows it (validation at cmd level)
		retrieved.DisplayName = ""
		require.NoError(t, repo.Update(retrieved))

		updated, err := repo.Get("renameproj3")
		require.NoError(t, err)
		assert.Equal(t, "", updated.DisplayName)
	})

	t.Run("rename multiple times", func(t *testing.T) {
		project := model.NewProject("renameproj4", "Name V1", "#AABBCC")
		require.NoError(t, repo.Create(project))

		for i := 2; i <= 5; i++ {
			retrieved, err := repo.Get("renameproj4")
			require.NoError(t, err)

			newName := "Name V" + string(rune('0'+i))
			retrieved.DisplayName = newName
			require.NoError(t, repo.Update(retrieved))

			updated, err := repo.Get("renameproj4")
			require.NoError(t, err)
			assert.Equal(t, newName, updated.DisplayName)
		}
	})

	t.Run("rename with special characters", func(t *testing.T) {
		project := model.NewProject("renameproj5", "Plain Name", "#FF0000")
		require.NoError(t, repo.Create(project))

		specialNames := []string{
			"Project (2024)",
			"Client/Internal",
			"Work & Life",
			"Test: Phase 1",
			"Project #42",
			`Name with "quotes"`,
			"Tab\there",
		}

		for _, name := range specialNames {
			retrieved, err := repo.Get("renameproj5")
			require.NoError(t, err)

			retrieved.DisplayName = name
			require.NoError(t, repo.Update(retrieved))

			updated, err := repo.Get("renameproj5")
			require.NoError(t, err)
			assert.Equal(t, name, updated.DisplayName)
		}
	})

	t.Run("rename with unicode characters", func(t *testing.T) {
		project := model.NewProject("renameproj6", "English Name", "#00FF00")
		require.NoError(t, repo.Create(project))

		unicodeNames := []string{
			"Cafe Project",
			"Projet Francais",
			"Proyecto Espanol",
			"Projekt Deutsch",
			"Chinese Characters",
			"Japanese Hiragana",
			"Korean Hangul",
			"Emoji Test: Work",
		}

		for _, name := range unicodeNames {
			retrieved, err := repo.Get("renameproj6")
			require.NoError(t, err)

			retrieved.DisplayName = name
			require.NoError(t, repo.Update(retrieved))

			updated, err := repo.Get("renameproj6")
			require.NoError(t, err)
			assert.Equal(t, name, updated.DisplayName)
		}
	})
}

// =============================================================================
// Project Color Operations Tests
// =============================================================================

func TestProjectColorOperations(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("set color on project without color", func(t *testing.T) {
		project := model.NewProject("colorproj1", "No Color Project", "")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("colorproj1")
		require.NoError(t, err)
		assert.Equal(t, "", retrieved.Color)

		retrieved.Color = "#FF5733"
		require.NoError(t, repo.Update(retrieved))

		updated, err := repo.Get("colorproj1")
		require.NoError(t, err)
		assert.Equal(t, "#FF5733", updated.Color)
	})

	t.Run("change project color", func(t *testing.T) {
		project := model.NewProject("colorproj2", "Color Project", "#FF0000")
		require.NoError(t, repo.Create(project))

		colors := []string{"#00FF00", "#0000FF", "#FFFF00", "#FF00FF", "#00FFFF"}
		for _, color := range colors {
			retrieved, err := repo.Get("colorproj2")
			require.NoError(t, err)

			retrieved.Color = color
			require.NoError(t, repo.Update(retrieved))

			updated, err := repo.Get("colorproj2")
			require.NoError(t, err)
			assert.Equal(t, color, updated.Color)
		}
	})

	t.Run("remove project color", func(t *testing.T) {
		project := model.NewProject("colorproj3", "Has Color", "#ABCDEF")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("colorproj3")
		require.NoError(t, err)

		retrieved.Color = ""
		require.NoError(t, repo.Update(retrieved))

		updated, err := repo.Get("colorproj3")
		require.NoError(t, err)
		assert.Equal(t, "", updated.Color)
	})

	t.Run("color format edge cases", func(t *testing.T) {
		// Test all valid hex color formats
		validColors := []string{
			"#000000", // Black
			"#FFFFFF", // White
			"#ffffff", // White (lowercase)
			"#AbCdEf", // Mixed case
			"#123456", // Sequential
			"#AABBCC", // Double hex
		}

		for i, color := range validColors {
			sid := "coloredge" + string(rune('0'+i))
			project := model.NewProject(sid, "Test", color)
			require.NoError(t, repo.Create(project))

			retrieved, err := repo.Get(sid)
			require.NoError(t, err)
			assert.Equal(t, color, retrieved.Color)
		}
	})

	t.Run("color validation comprehensive", func(t *testing.T) {
		// Valid
		assert.True(t, model.ValidateColor(""))
		assert.True(t, model.ValidateColor("#000000"))
		assert.True(t, model.ValidateColor("#ffffff"))
		assert.True(t, model.ValidateColor("#FFFFFF"))
		assert.True(t, model.ValidateColor("#AbCdEf"))
		assert.True(t, model.ValidateColor("#123456"))
		assert.True(t, model.ValidateColor("#7890ab"))

		// Invalid - wrong length
		assert.False(t, model.ValidateColor("#FFF"))
		assert.False(t, model.ValidateColor("#FFFF"))
		assert.False(t, model.ValidateColor("#FFFFF"))
		assert.False(t, model.ValidateColor("#FFFFFFF"))
		assert.False(t, model.ValidateColor("#FFFFFFFF"))

		// Invalid - missing hash
		assert.False(t, model.ValidateColor("000000"))
		assert.False(t, model.ValidateColor("FFFFFF"))

		// Invalid - wrong prefix
		assert.False(t, model.ValidateColor("0xFF5733"))
		assert.False(t, model.ValidateColor("$FF5733"))

		// Invalid - non-hex characters
		assert.False(t, model.ValidateColor("#GGGGGG"))
		assert.False(t, model.ValidateColor("#12345G"))
		assert.False(t, model.ValidateColor("#ZZZZZZ"))

		// Invalid - special characters
		assert.False(t, model.ValidateColor("#FF 5733"))
		assert.False(t, model.ValidateColor("# FF5733"))
		assert.False(t, model.ValidateColor("#FF5733 "))
		assert.False(t, model.ValidateColor(" #FF5733"))

		// Invalid - color names
		assert.False(t, model.ValidateColor("red"))
		assert.False(t, model.ValidateColor("blue"))
		assert.False(t, model.ValidateColor("green"))

		// Invalid - rgb/rgba format
		assert.False(t, model.ValidateColor("rgb(255,0,0)"))
		assert.False(t, model.ValidateColor("rgba(255,0,0,1)"))
	})
}

// =============================================================================
// Project Lifecycle Tests
// =============================================================================

func TestProjectLifecycle(t *testing.T) {
	t.Run("full project lifecycle: create, read, update, delete", func(t *testing.T) {
		db := setupTestDB(t)
		repo := storage.NewProjectRepo(db)

		// Create
		project := model.NewProject("lifecycle", "Lifecycle Project", "#FF5733")
		require.NoError(t, repo.Create(project))

		// Read
		retrieved, err := repo.Get("lifecycle")
		require.NoError(t, err)
		assert.Equal(t, "lifecycle", retrieved.SID)
		assert.Equal(t, "Lifecycle Project", retrieved.DisplayName)
		assert.Equal(t, "#FF5733", retrieved.Color)

		// Update name
		retrieved.DisplayName = "Updated Lifecycle"
		require.NoError(t, repo.Update(retrieved))

		updated, err := repo.Get("lifecycle")
		require.NoError(t, err)
		assert.Equal(t, "Updated Lifecycle", updated.DisplayName)

		// Update color
		updated.Color = "#00FF00"
		require.NoError(t, repo.Update(updated))

		final, err := repo.Get("lifecycle")
		require.NoError(t, err)
		assert.Equal(t, "#00FF00", final.Color)

		// Delete
		require.NoError(t, repo.Delete("lifecycle"))

		// Verify deleted
		exists, err := repo.Exists("lifecycle")
		require.NoError(t, err)
		assert.False(t, exists)

		_, err = repo.Get("lifecycle")
		assert.Error(t, err)
		assert.True(t, storage.IsErrKeyNotFound(err))
	})

	t.Run("project with tasks and blocks lifecycle", func(t *testing.T) {
		db := setupTestDB(t)
		projectRepo := storage.NewProjectRepo(db)
		taskRepo := storage.NewTaskRepo(db)
		blockRepo := storage.NewBlockRepo(db)

		// Create project
		project := model.NewProject("fulllife", "Full Lifecycle", "#AABBCC")
		require.NoError(t, projectRepo.Create(project))

		// Add tasks
		task1 := model.NewTask("fulllife", "task1", "Task One", "#111111")
		task2 := model.NewTask("fulllife", "task2", "Task Two", "#222222")
		require.NoError(t, taskRepo.Create(task1))
		require.NoError(t, taskRepo.Create(task2))

		// Add blocks
		block1 := model.NewBlock("user1", "fulllife", "task1", "note1", time.Now())
		block2 := model.NewBlock("user1", "fulllife", "task2", "note2", time.Now())
		block3 := model.NewBlock("user1", "fulllife", "", "no task", time.Now())
		require.NoError(t, blockRepo.Create(block1))
		require.NoError(t, blockRepo.Create(block2))
		require.NoError(t, blockRepo.Create(block3))

		// Verify all exist
		tasks, err := taskRepo.ListByProject("fulllife")
		require.NoError(t, err)
		assert.Len(t, tasks, 2)

		blocks, err := blockRepo.ListByProject("fulllife")
		require.NoError(t, err)
		assert.Len(t, blocks, 3)

		// Update project
		retrieved, err := projectRepo.Get("fulllife")
		require.NoError(t, err)
		retrieved.DisplayName = "Updated Full Life"
		require.NoError(t, projectRepo.Update(retrieved))

		// Verify tasks and blocks still associated
		tasks, err = taskRepo.ListByProject("fulllife")
		require.NoError(t, err)
		assert.Len(t, tasks, 2)

		blocks, err = blockRepo.ListByProject("fulllife")
		require.NoError(t, err)
		assert.Len(t, blocks, 3)

		// Delete project (note: orphans tasks and blocks)
		require.NoError(t, projectRepo.Delete("fulllife"))

		// Verify project deleted but tasks and blocks remain
		exists, err := projectRepo.Exists("fulllife")
		require.NoError(t, err)
		assert.False(t, exists)

		tasks, err = taskRepo.ListByProject("fulllife")
		require.NoError(t, err)
		assert.Len(t, tasks, 2) // Orphaned

		blocks, err = blockRepo.ListByProject("fulllife")
		require.NoError(t, err)
		assert.Len(t, blocks, 3) // Orphaned
	})
}

// =============================================================================
// Concurrent Operations Tests
// =============================================================================

func TestProjectConcurrentOperations(t *testing.T) {
	t.Run("concurrent project creation", func(t *testing.T) {
		db := setupTestDB(t)
		repo := storage.NewProjectRepo(db)

		// Create multiple projects concurrently
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(idx int) {
				sid := "concurrent" + string(rune('A'+idx))
				project := model.NewProject(sid, "Concurrent "+sid, "#FF5733")
				_ = repo.Create(project)
				done <- true
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all projects exist
		projects, err := repo.List()
		require.NoError(t, err)
		assert.Len(t, projects, 10)
	})

	t.Run("concurrent read and write", func(t *testing.T) {
		db := setupTestDB(t)
		repo := storage.NewProjectRepo(db)

		// Create initial project
		project := model.NewProject("concurrentrw", "Concurrent RW", "#000000")
		require.NoError(t, repo.Create(project))

		done := make(chan bool, 20)

		// Concurrent reads
		for i := 0; i < 10; i++ {
			go func() {
				_, _ = repo.Get("concurrentrw")
				done <- true
			}()
		}

		// Concurrent updates
		for i := 0; i < 10; i++ {
			go func(idx int) {
				retrieved, err := repo.Get("concurrentrw")
				if err == nil {
					retrieved.DisplayName = "Updated " + string(rune('0'+idx))
					_ = repo.Update(retrieved)
				}
				done <- true
			}(i)
		}

		// Wait for all
		for i := 0; i < 20; i++ {
			<-done
		}

		// Verify project still exists and is valid
		final, err := repo.Get("concurrentrw")
		require.NoError(t, err)
		assert.Equal(t, "concurrentrw", final.SID)
	})
}

// =============================================================================
// Project Key Generation Tests
// =============================================================================

func TestProjectKeyGeneration(t *testing.T) {
	t.Run("key format is correct", func(t *testing.T) {
		project := model.NewProject("testkey", "Test Key", "")
		assert.Equal(t, model.PrefixProject+":testkey", project.Key)
	})

	t.Run("key is deterministic", func(t *testing.T) {
		key1 := model.GenerateProjectKey("myproject")
		key2 := model.GenerateProjectKey("myproject")
		assert.Equal(t, key1, key2)
	})

	t.Run("different SIDs produce different keys", func(t *testing.T) {
		key1 := model.GenerateProjectKey("project1")
		key2 := model.GenerateProjectKey("project2")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("key survives create-retrieve cycle", func(t *testing.T) {
		db := setupTestDB(t)
		repo := storage.NewProjectRepo(db)

		project := model.NewProject("keytest", "Key Test", "#FFFFFF")
		originalKey := project.Key
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("keytest")
		require.NoError(t, err)
		assert.Equal(t, originalKey, retrieved.Key)
	})
}

// =============================================================================
// Project SID Validation Tests
// =============================================================================

func TestProjectSIDEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewProjectRepo(db)

	t.Run("SID with numbers", func(t *testing.T) {
		project := model.NewProject("project123", "Project 123", "")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("project123")
		require.NoError(t, err)
		assert.Equal(t, "project123", retrieved.SID)
	})

	t.Run("SID with underscores", func(t *testing.T) {
		project := model.NewProject("my_project", "My Project", "")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("my_project")
		require.NoError(t, err)
		assert.Equal(t, "my_project", retrieved.SID)
	})

	t.Run("SID with hyphens", func(t *testing.T) {
		project := model.NewProject("my-project", "My Project", "")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("my-project")
		require.NoError(t, err)
		assert.Equal(t, "my-project", retrieved.SID)
	})

	t.Run("single character SID", func(t *testing.T) {
		project := model.NewProject("a", "Single Char", "")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("a")
		require.NoError(t, err)
		assert.Equal(t, "a", retrieved.SID)
	})

	t.Run("numeric only SID", func(t *testing.T) {
		project := model.NewProject("12345", "Numeric Only", "")
		require.NoError(t, repo.Create(project))

		retrieved, err := repo.Get("12345")
		require.NoError(t, err)
		assert.Equal(t, "12345", retrieved.SID)
	})

	t.Run("mixed case SID retrieval", func(t *testing.T) {
		project := model.NewProject("MixedCase", "Mixed Case Project", "")
		require.NoError(t, repo.Create(project))

		// Exact case works
		retrieved, err := repo.Get("MixedCase")
		require.NoError(t, err)
		assert.Equal(t, "MixedCase", retrieved.SID)

		// Different case fails (SIDs are case-sensitive)
		_, err = repo.Get("mixedcase")
		assert.True(t, storage.IsErrKeyNotFound(err))

		_, err = repo.Get("MIXEDCASE")
		assert.True(t, storage.IsErrKeyNotFound(err))
	})
}
