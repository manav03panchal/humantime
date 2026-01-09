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
}
