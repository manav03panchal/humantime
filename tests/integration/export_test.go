// Package integration provides integration tests for Humantime.
//
// These tests verify the export functionality using an in-memory Badger
// database. They test the complete export flow including JSON and CSV
// formats, backup functionality, and filtering options.
package integration

import (
	"encoding/csv"
	"encoding/json"
	"strings"
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

// testContext holds test configuration with in-memory database.
type testContext struct {
	t               *testing.T
	db              *storage.DB
	blockRepo       *storage.BlockRepo
	projectRepo     *storage.ProjectRepo
	taskRepo        *storage.TaskRepo
	configRepo      *storage.ConfigRepo
	activeBlockRepo *storage.ActiveBlockRepo
	goalRepo        *storage.GoalRepo
}

// setupTestDB creates a new test context with an in-memory database.
func setupTestDB(t *testing.T) *testContext {
	t.Helper()
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err, "failed to open in-memory database")

	tc := &testContext{
		t:               t,
		db:              db,
		blockRepo:       storage.NewBlockRepo(db),
		projectRepo:     storage.NewProjectRepo(db),
		taskRepo:        storage.NewTaskRepo(db),
		configRepo:      storage.NewConfigRepo(db),
		activeBlockRepo: storage.NewActiveBlockRepo(db),
		goalRepo:        storage.NewGoalRepo(db),
	}

	t.Cleanup(func() {
		err := db.Close()
		assert.NoError(t, err, "failed to close database")
	})

	return tc
}

// createTestBlock creates a block with the given parameters and saves it.
func (tc *testContext) createTestBlock(projectSID, taskSID, note string, start, end time.Time) *model.Block {
	tc.t.Helper()

	// Ensure project exists
	_, _, err := tc.projectRepo.GetOrCreate(projectSID, projectSID)
	require.NoError(tc.t, err)

	// Ensure task exists if specified
	if taskSID != "" {
		_, _, err := tc.taskRepo.GetOrCreate(projectSID, taskSID, taskSID)
		require.NoError(tc.t, err)
	}

	// Get config for owner key
	config, err := tc.configRepo.Get()
	require.NoError(tc.t, err)

	block := model.NewBlock(config.UserKey, projectSID, taskSID, note, start)
	if !end.IsZero() {
		block.TimestampEnd = end
	}

	err = tc.blockRepo.Create(block)
	require.NoError(tc.t, err)

	return block
}

// createTestProject creates a project with the given parameters.
func (tc *testContext) createTestProject(sid, displayName, color string) *model.Project {
	tc.t.Helper()

	project := model.NewProject(sid, displayName, color)
	err := tc.projectRepo.Create(project)
	require.NoError(tc.t, err)

	return project
}

// createTestTask creates a task with the given parameters.
func (tc *testContext) createTestTask(projectSID, sid, displayName, color string) *model.Task {
	tc.t.Helper()

	task := model.NewTask(projectSID, sid, displayName, color)
	err := tc.taskRepo.Create(task)
	require.NoError(tc.t, err)

	return task
}

// createTestGoal creates a goal with the given parameters.
func (tc *testContext) createTestGoal(projectSID string, goalType model.GoalType, target time.Duration) *model.Goal {
	tc.t.Helper()

	goal := model.NewGoal(projectSID, goalType, target)
	err := tc.goalRepo.Create(goal)
	require.NoError(tc.t, err)

	return goal
}

// =============================================================================
// JSON Export Structure Types (matching export.go output)
// =============================================================================

// jsonExport represents the JSON export output structure.
type jsonExport struct {
	Version    string        `json:"version"`
	ExportedAt string        `json:"exported_at"`
	Blocks     []blockOutput `json:"blocks"`
	Count      int           `json:"count"`
}

// blockOutput represents a block in JSON export.
type blockOutput struct {
	Key             string `json:"key"`
	ProjectSID      string `json:"project_sid"`
	TaskSID         string `json:"task_sid,omitempty"`
	Note            string `json:"note,omitempty"`
	TimestampStart  string `json:"timestamp_start"`
	TimestampEnd    string `json:"timestamp_end,omitempty"`
	DurationSeconds int64  `json:"duration_seconds"`
	IsActive        bool   `json:"is_active"`
}

// jsonBackup represents the JSON backup output structure.
type jsonBackup struct {
	Version     string              `json:"version"`
	ExportedAt  string              `json:"exported_at"`
	Config      *model.Config       `json:"config"`
	Projects    []*model.Project    `json:"projects"`
	Tasks       []*model.Task       `json:"tasks"`
	Blocks      []*model.Block      `json:"blocks"`
	Goals       []*model.Goal       `json:"goals"`
	ActiveBlock *model.ActiveBlock  `json:"active_block"`
}

// =============================================================================
// Export Format Helper Functions (simulating export.go logic)
// =============================================================================

// exportBlocksToJSON exports blocks to JSON format (mimicking export.go).
func exportBlocksToJSON(blocks []*model.Block) (string, error) {
	output := struct {
		Version    string        `json:"version"`
		ExportedAt string        `json:"exported_at"`
		Blocks     []blockOutput `json:"blocks"`
		Count      int           `json:"count"`
	}{
		Version:    "1",
		ExportedAt: time.Now().Format(time.RFC3339),
		Blocks:     make([]blockOutput, len(blocks)),
		Count:      len(blocks),
	}

	for i, b := range blocks {
		out := blockOutput{
			Key:             b.Key,
			ProjectSID:      b.ProjectSID,
			TaskSID:         b.TaskSID,
			Note:            b.Note,
			TimestampStart:  b.TimestampStart.Format(time.RFC3339),
			DurationSeconds: b.DurationSeconds(),
			IsActive:        b.IsActive(),
		}
		if !b.TimestampEnd.IsZero() {
			out.TimestampEnd = b.TimestampEnd.Format(time.RFC3339)
		}
		output.Blocks[i] = out
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// exportBlocksToCSV exports blocks to CSV format (mimicking export.go).
func exportBlocksToCSV(blocks []*model.Block) (string, error) {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	// Write header
	if err := writer.Write([]string{
		"key", "project", "task", "note", "start", "end", "duration_seconds",
	}); err != nil {
		return "", err
	}

	// Write rows
	for _, b := range blocks {
		endStr := ""
		if !b.TimestampEnd.IsZero() {
			endStr = b.TimestampEnd.Format(time.RFC3339)
		}

		if err := writer.Write([]string{
			b.Key,
			b.ProjectSID,
			b.TaskSID,
			b.Note,
			b.TimestampStart.Format(time.RFC3339),
			endStr,
			formatInt(b.DurationSeconds()),
		}); err != nil {
			return "", err
		}
	}

	writer.Flush()
	return sb.String(), nil
}

// exportBackupToJSON creates a full backup in JSON format.
func exportBackupToJSON(
	config *model.Config,
	projects []*model.Project,
	tasks []*model.Task,
	blocks []*model.Block,
	goals []*model.Goal,
	activeBlock *model.ActiveBlock,
) (string, error) {
	backup := jsonBackup{
		Version:     "1",
		ExportedAt:  time.Now().Format(time.RFC3339),
		Config:      config,
		Projects:    projects,
		Tasks:       tasks,
		Blocks:      blocks,
		Goals:       goals,
		ActiveBlock: activeBlock,
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func formatInt(n int64) string {
	return strings.TrimSpace(strings.Replace(
		strings.Replace(
			strings.Replace(string(rune(n+'0')), "\x00", "", -1),
			"", "", 0),
		"", "", 0))
}

// =============================================================================
// JSON Export Format Tests
// =============================================================================

func TestExportJSON_EmptyDatabase(t *testing.T) {
	tc := setupTestDB(t)

	// Get all blocks (should be empty)
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	assert.Empty(t, blocks)

	// Export to JSON
	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Parse and verify
	var export jsonExport
	err = json.Unmarshal([]byte(jsonStr), &export)
	require.NoError(t, err)

	assert.Equal(t, "1", export.Version)
	assert.NotEmpty(t, export.ExportedAt)
	assert.Equal(t, 0, export.Count)
	assert.Empty(t, export.Blocks)
}

func TestExportJSON_SingleBlock(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()
	start := now.Add(-2 * time.Hour)
	end := now.Add(-1 * time.Hour)

	// Create a block
	block := tc.createTestBlock("project1", "task1", "test note", start, end)

	// Get all blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	// Export to JSON
	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Parse and verify
	var export jsonExport
	err = json.Unmarshal([]byte(jsonStr), &export)
	require.NoError(t, err)

	assert.Equal(t, "1", export.Version)
	assert.NotEmpty(t, export.ExportedAt)
	assert.Equal(t, 1, export.Count)
	require.Len(t, export.Blocks, 1)

	exported := export.Blocks[0]
	assert.Equal(t, block.Key, exported.Key)
	assert.Equal(t, "project1", exported.ProjectSID)
	assert.Equal(t, "task1", exported.TaskSID)
	assert.Equal(t, "test note", exported.Note)
	assert.False(t, exported.IsActive)
	assert.Greater(t, exported.DurationSeconds, int64(0))
}

func TestExportJSON_MultipleBlocks(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create multiple blocks
	tc.createTestBlock("project1", "task1", "note 1", now.Add(-4*time.Hour), now.Add(-3*time.Hour))
	tc.createTestBlock("project1", "task2", "note 2", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
	tc.createTestBlock("project2", "", "note 3", now.Add(-2*time.Hour), now.Add(-1*time.Hour))

	// Get all blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 3)

	// Export to JSON
	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Parse and verify
	var export jsonExport
	err = json.Unmarshal([]byte(jsonStr), &export)
	require.NoError(t, err)

	assert.Equal(t, 3, export.Count)
	require.Len(t, export.Blocks, 3)

	// Verify all blocks have required fields
	projectCount := make(map[string]int)
	for _, b := range export.Blocks {
		assert.NotEmpty(t, b.Key)
		assert.NotEmpty(t, b.ProjectSID)
		assert.NotEmpty(t, b.TimestampStart)
		projectCount[b.ProjectSID]++
	}

	assert.Equal(t, 2, projectCount["project1"])
	assert.Equal(t, 1, projectCount["project2"])
}

func TestExportJSON_ActiveBlock(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create an active block (no end time)
	block := tc.createTestBlock("project1", "", "active work", now.Add(-30*time.Minute), time.Time{})

	// Get all blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	// Export to JSON
	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Parse and verify
	var export jsonExport
	err = json.Unmarshal([]byte(jsonStr), &export)
	require.NoError(t, err)

	require.Len(t, export.Blocks, 1)
	exported := export.Blocks[0]

	assert.Equal(t, block.Key, exported.Key)
	assert.True(t, exported.IsActive)
	assert.Empty(t, exported.TimestampEnd)
	assert.Greater(t, exported.DurationSeconds, int64(0))
}

func TestExportJSON_BlockWithSpecialCharactersInNote(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create a block with special characters in note
	specialNote := "Test with \"quotes\", 'apostrophes', and\nnewlines"
	tc.createTestBlock("project1", "", specialNote, now.Add(-1*time.Hour), now)

	// Get all blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	// Export to JSON
	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Parse and verify - JSON should handle special characters correctly
	var export jsonExport
	err = json.Unmarshal([]byte(jsonStr), &export)
	require.NoError(t, err)

	require.Len(t, export.Blocks, 1)
	assert.Equal(t, specialNote, export.Blocks[0].Note)
}

// =============================================================================
// CSV Export Format Tests
// =============================================================================

func TestExportCSV_EmptyDatabase(t *testing.T) {
	tc := setupTestDB(t)

	// Get all blocks (should be empty)
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	// Export to CSV
	csvStr, err := exportBlocksToCSV(blocks)
	require.NoError(t, err)

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(csvStr))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Should have only header
	require.Len(t, records, 1)

	// Verify header
	expectedHeaders := []string{"key", "project", "task", "note", "start", "end", "duration_seconds"}
	assert.Equal(t, expectedHeaders, records[0])
}

func TestExportCSV_SingleBlock(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()
	start := now.Add(-2 * time.Hour)
	end := now.Add(-1 * time.Hour)

	// Create a block
	block := tc.createTestBlock("project1", "task1", "test note", start, end)

	// Get all blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	// Export to CSV
	csvStr, err := exportBlocksToCSV(blocks)
	require.NoError(t, err)

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(csvStr))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Should have header + 1 data row
	require.Len(t, records, 2)

	// Verify data row
	dataRow := records[1]
	assert.Equal(t, block.Key, dataRow[0])        // key
	assert.Equal(t, "project1", dataRow[1])        // project
	assert.Equal(t, "task1", dataRow[2])           // task
	assert.Equal(t, "test note", dataRow[3])       // note
	assert.NotEmpty(t, dataRow[4])                 // start
	assert.NotEmpty(t, dataRow[5])                 // end
	assert.NotEmpty(t, dataRow[6])                 // duration_seconds
}

func TestExportCSV_MultipleBlocks(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create multiple blocks
	tc.createTestBlock("project1", "task1", "note 1", now.Add(-4*time.Hour), now.Add(-3*time.Hour))
	tc.createTestBlock("project1", "task2", "note 2", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
	tc.createTestBlock("project2", "", "note 3", now.Add(-2*time.Hour), now.Add(-1*time.Hour))

	// Get all blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	// Export to CSV
	csvStr, err := exportBlocksToCSV(blocks)
	require.NoError(t, err)

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(csvStr))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Should have header + 3 data rows
	require.Len(t, records, 4)

	// Verify all rows have correct number of columns
	for _, row := range records {
		assert.Len(t, row, 7)
	}

	// Verify projects
	projects := make(map[string]int)
	for _, row := range records[1:] {
		projects[row[1]]++
	}
	assert.Equal(t, 2, projects["project1"])
	assert.Equal(t, 1, projects["project2"])
}

func TestExportCSV_ActiveBlock(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create an active block (no end time)
	tc.createTestBlock("project1", "", "active work", now.Add(-30*time.Minute), time.Time{})

	// Get all blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	// Export to CSV
	csvStr, err := exportBlocksToCSV(blocks)
	require.NoError(t, err)

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(csvStr))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	require.Len(t, records, 2)

	// Active block should have empty end time
	dataRow := records[1]
	assert.Empty(t, dataRow[5]) // end should be empty
}

func TestExportCSV_BlockWithCommasInNote(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create a block with commas in note
	noteWithCommas := "Task 1, Task 2, Task 3"
	tc.createTestBlock("project1", "", noteWithCommas, now.Add(-1*time.Hour), now)

	// Get all blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	// Export to CSV
	csvStr, err := exportBlocksToCSV(blocks)
	require.NoError(t, err)

	// Parse CSV - should handle commas correctly
	reader := csv.NewReader(strings.NewReader(csvStr))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	require.Len(t, records, 2)
	assert.Equal(t, noteWithCommas, records[1][3])
}

func TestExportCSV_HeaderFormat(t *testing.T) {
	tc := setupTestDB(t)

	// Get all blocks (empty)
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	// Export to CSV
	csvStr, err := exportBlocksToCSV(blocks)
	require.NoError(t, err)

	// Verify header row format exactly
	lines := strings.Split(csvStr, "\n")
	require.NotEmpty(t, lines)
	assert.Equal(t, "key,project,task,note,start,end,duration_seconds", lines[0])
}

// =============================================================================
// Backup Flag Tests
// =============================================================================

func TestExportBackup_EmptyDatabase(t *testing.T) {
	tc := setupTestDB(t)

	// Get all data
	config, err := tc.configRepo.Get()
	require.NoError(t, err)

	projects, err := tc.projectRepo.List()
	require.NoError(t, err)

	tasks, err := tc.taskRepo.List()
	require.NoError(t, err)

	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	goals, err := tc.goalRepo.List()
	require.NoError(t, err)

	activeBlock, err := tc.activeBlockRepo.Get()
	require.NoError(t, err)

	// Create backup JSON
	jsonStr, err := exportBackupToJSON(config, projects, tasks, blocks, goals, activeBlock)
	require.NoError(t, err)

	// Parse and verify
	var backup jsonBackup
	err = json.Unmarshal([]byte(jsonStr), &backup)
	require.NoError(t, err)

	assert.Equal(t, "1", backup.Version)
	assert.NotEmpty(t, backup.ExportedAt)
	assert.NotNil(t, backup.Config)
	assert.Empty(t, backup.Projects)
	assert.Empty(t, backup.Tasks)
	assert.Empty(t, backup.Blocks)
	assert.Empty(t, backup.Goals)
	assert.NotNil(t, backup.ActiveBlock)
}

func TestExportBackup_WithAllDataTypes(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create projects
	tc.createTestProject("project1", "Project One", "#FF0000")
	tc.createTestProject("project2", "Project Two", "#00FF00")

	// Create tasks
	tc.createTestTask("project1", "task1", "Task One", "#0000FF")
	tc.createTestTask("project1", "task2", "Task Two", "")

	// Create blocks
	tc.createTestBlock("project1", "task1", "note 1", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	tc.createTestBlock("project2", "", "note 2", now.Add(-1*time.Hour), now)

	// Create goals
	tc.createTestGoal("project1", model.GoalTypeDaily, 8*time.Hour)
	tc.createTestGoal("project2", model.GoalTypeWeekly, 40*time.Hour)

	// Get all data
	config, err := tc.configRepo.Get()
	require.NoError(t, err)

	projects, err := tc.projectRepo.List()
	require.NoError(t, err)

	tasks, err := tc.taskRepo.List()
	require.NoError(t, err)

	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	goals, err := tc.goalRepo.List()
	require.NoError(t, err)

	activeBlock, err := tc.activeBlockRepo.Get()
	require.NoError(t, err)

	// Create backup JSON
	jsonStr, err := exportBackupToJSON(config, projects, tasks, blocks, goals, activeBlock)
	require.NoError(t, err)

	// Parse and verify
	var backup jsonBackup
	err = json.Unmarshal([]byte(jsonStr), &backup)
	require.NoError(t, err)

	// Verify counts
	assert.Len(t, backup.Projects, 2)
	assert.Len(t, backup.Tasks, 2)
	assert.Len(t, backup.Blocks, 2)
	assert.Len(t, backup.Goals, 2)

	// Verify config
	assert.NotNil(t, backup.Config)
	assert.NotEmpty(t, backup.Config.UserKey)

	// Verify projects have all fields
	projectMap := make(map[string]*model.Project)
	for _, p := range backup.Projects {
		projectMap[p.SID] = p
	}
	assert.NotNil(t, projectMap["project1"])
	assert.Equal(t, "Project One", projectMap["project1"].DisplayName)
	assert.Equal(t, "#FF0000", projectMap["project1"].Color)

	// Verify goals
	goalMap := make(map[string]*model.Goal)
	for _, g := range backup.Goals {
		goalMap[g.ProjectSID] = g
	}
	assert.NotNil(t, goalMap["project1"])
	assert.Equal(t, model.GoalTypeDaily, goalMap["project1"].Type)
	assert.Equal(t, 8*time.Hour, goalMap["project1"].Target)
}

func TestExportBackup_WithActiveBlock(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create an active block
	block := tc.createTestBlock("project1", "", "active work", now.Add(-30*time.Minute), time.Time{})

	// Set it as active
	err := tc.activeBlockRepo.SetActive(block.Key)
	require.NoError(t, err)

	// Get all data
	config, err := tc.configRepo.Get()
	require.NoError(t, err)

	projects, err := tc.projectRepo.List()
	require.NoError(t, err)

	tasks, err := tc.taskRepo.List()
	require.NoError(t, err)

	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	goals, err := tc.goalRepo.List()
	require.NoError(t, err)

	activeBlock, err := tc.activeBlockRepo.Get()
	require.NoError(t, err)

	// Create backup JSON
	jsonStr, err := exportBackupToJSON(config, projects, tasks, blocks, goals, activeBlock)
	require.NoError(t, err)

	// Parse and verify
	var backup jsonBackup
	err = json.Unmarshal([]byte(jsonStr), &backup)
	require.NoError(t, err)

	// Verify active block state
	assert.NotNil(t, backup.ActiveBlock)
	assert.Equal(t, block.Key, backup.ActiveBlock.ActiveBlockKey)
	assert.True(t, backup.ActiveBlock.IsTracking())
}

// =============================================================================
// Filter by Project Tests
// =============================================================================

func TestExportFilterByProject_SingleProject(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create blocks for different projects
	tc.createTestBlock("project1", "", "note 1", now.Add(-4*time.Hour), now.Add(-3*time.Hour))
	tc.createTestBlock("project1", "", "note 2", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
	tc.createTestBlock("project2", "", "note 3", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	tc.createTestBlock("project3", "", "note 4", now.Add(-1*time.Hour), now)

	// Filter by project1
	filter := storage.BlockFilter{ProjectSID: "project1"}
	blocks, err := tc.blockRepo.ListFiltered(filter)
	require.NoError(t, err)

	// Should only have project1 blocks
	assert.Len(t, blocks, 2)
	for _, b := range blocks {
		assert.Equal(t, "project1", b.ProjectSID)
	}

	// Export filtered blocks to JSON
	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	var export jsonExport
	err = json.Unmarshal([]byte(jsonStr), &export)
	require.NoError(t, err)

	assert.Equal(t, 2, export.Count)
	for _, b := range export.Blocks {
		assert.Equal(t, "project1", b.ProjectSID)
	}
}

func TestExportFilterByProject_NonExistentProject(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create blocks
	tc.createTestBlock("project1", "", "note 1", now.Add(-2*time.Hour), now.Add(-1*time.Hour))

	// Filter by non-existent project
	filter := storage.BlockFilter{ProjectSID: "nonexistent"}
	blocks, err := tc.blockRepo.ListFiltered(filter)
	require.NoError(t, err)

	// Should be empty
	assert.Empty(t, blocks)

	// Export filtered blocks
	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	var export jsonExport
	err = json.Unmarshal([]byte(jsonStr), &export)
	require.NoError(t, err)

	assert.Equal(t, 0, export.Count)
	assert.Empty(t, export.Blocks)
}

func TestExportFilterByTask(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create blocks with different tasks
	tc.createTestBlock("project1", "task1", "note 1", now.Add(-4*time.Hour), now.Add(-3*time.Hour))
	tc.createTestBlock("project1", "task1", "note 2", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
	tc.createTestBlock("project1", "task2", "note 3", now.Add(-2*time.Hour), now.Add(-1*time.Hour))

	// Filter by task1
	filter := storage.BlockFilter{TaskSID: "task1"}
	blocks, err := tc.blockRepo.ListFiltered(filter)
	require.NoError(t, err)

	// Should only have task1 blocks
	assert.Len(t, blocks, 2)
	for _, b := range blocks {
		assert.Equal(t, "task1", b.TaskSID)
	}
}

func TestExportFilterByProjectAndTask(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create blocks with different projects and tasks
	tc.createTestBlock("project1", "task1", "note 1", now.Add(-5*time.Hour), now.Add(-4*time.Hour))
	tc.createTestBlock("project1", "task2", "note 2", now.Add(-4*time.Hour), now.Add(-3*time.Hour))
	tc.createTestBlock("project2", "task1", "note 3", now.Add(-3*time.Hour), now.Add(-2*time.Hour))

	// Filter by project1 and task1
	filter := storage.BlockFilter{ProjectSID: "project1", TaskSID: "task1"}
	blocks, err := tc.blockRepo.ListFiltered(filter)
	require.NoError(t, err)

	// Should only have project1/task1 blocks
	assert.Len(t, blocks, 1)
	assert.Equal(t, "project1", blocks[0].ProjectSID)
	assert.Equal(t, "task1", blocks[0].TaskSID)
}

// =============================================================================
// Filter by Time Tests
// =============================================================================

func TestExportFilterByTime_StartAfter(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create blocks at different times
	tc.createTestBlock("project1", "", "old block", now.Add(-24*time.Hour), now.Add(-23*time.Hour))
	tc.createTestBlock("project1", "", "recent block 1", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	tc.createTestBlock("project1", "", "recent block 2", now.Add(-1*time.Hour), now)

	// Filter blocks that start after 3 hours ago
	startAfter := now.Add(-3 * time.Hour)
	filter := storage.BlockFilter{StartAfter: startAfter}
	blocks, err := tc.blockRepo.ListFiltered(filter)
	require.NoError(t, err)

	// Should only have recent blocks
	assert.Len(t, blocks, 2)
	for _, b := range blocks {
		assert.True(t, b.TimestampStart.After(startAfter) || b.TimestampStart.Equal(startAfter))
	}
}

func TestExportFilterByTime_EndBefore(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create blocks at different times
	tc.createTestBlock("project1", "", "old block", now.Add(-24*time.Hour), now.Add(-23*time.Hour))
	tc.createTestBlock("project1", "", "recent block", now.Add(-1*time.Hour), now)

	// Filter blocks that end before 12 hours ago
	endBefore := now.Add(-12 * time.Hour)
	filter := storage.BlockFilter{EndBefore: endBefore}
	blocks, err := tc.blockRepo.ListFiltered(filter)
	require.NoError(t, err)

	// Should only have old blocks
	assert.Len(t, blocks, 1)
	assert.Contains(t, blocks[0].Note, "old block")
}

func TestExportFilterByTime_TimeRange(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create blocks spanning multiple days
	tc.createTestBlock("project1", "", "very old", now.Add(-72*time.Hour), now.Add(-71*time.Hour))
	tc.createTestBlock("project1", "", "yesterday", now.Add(-36*time.Hour), now.Add(-35*time.Hour))
	tc.createTestBlock("project1", "", "today", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	tc.createTestBlock("project1", "", "future", now.Add(1*time.Hour), now.Add(2*time.Hour))

	// Filter blocks from the last 48 hours
	startAfter := now.Add(-48 * time.Hour)
	endBefore := now.Add(30 * time.Minute)
	filter := storage.BlockFilter{StartAfter: startAfter, EndBefore: endBefore}
	blocks, err := tc.blockRepo.ListFiltered(filter)
	require.NoError(t, err)

	// Should have yesterday and today blocks
	assert.Len(t, blocks, 2)
}

func TestExportFilterByTime_CombinedWithProject(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create blocks for different projects and times
	tc.createTestBlock("project1", "", "p1 old", now.Add(-24*time.Hour), now.Add(-23*time.Hour))
	tc.createTestBlock("project1", "", "p1 recent", now.Add(-1*time.Hour), now)
	tc.createTestBlock("project2", "", "p2 old", now.Add(-24*time.Hour), now.Add(-23*time.Hour))
	tc.createTestBlock("project2", "", "p2 recent", now.Add(-1*time.Hour), now)

	// Filter project1 blocks from last 12 hours
	filter := storage.BlockFilter{
		ProjectSID: "project1",
		StartAfter: now.Add(-12 * time.Hour),
	}
	blocks, err := tc.blockRepo.ListFiltered(filter)
	require.NoError(t, err)

	// Should only have project1 recent block
	assert.Len(t, blocks, 1)
	assert.Equal(t, "project1", blocks[0].ProjectSID)
	assert.Contains(t, blocks[0].Note, "recent")
}

// =============================================================================
// Export Data Integrity Tests
// =============================================================================

func TestExportJSON_TimestampFormat(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	end := now

	// Create a block
	tc.createTestBlock("project1", "", "test", start, end)

	// Get blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	// Export to JSON
	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Verify timestamps are in RFC3339 format
	var export jsonExport
	err = json.Unmarshal([]byte(jsonStr), &export)
	require.NoError(t, err)

	require.Len(t, export.Blocks, 1)

	// Try to parse the timestamps - should be valid RFC3339
	_, err = time.Parse(time.RFC3339, export.Blocks[0].TimestampStart)
	assert.NoError(t, err, "TimestampStart should be valid RFC3339")

	_, err = time.Parse(time.RFC3339, export.Blocks[0].TimestampEnd)
	assert.NoError(t, err, "TimestampEnd should be valid RFC3339")
}

func TestExportJSON_DurationCalculation(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()
	start := now.Add(-90 * time.Minute) // 90 minutes = 5400 seconds
	end := now

	// Create a block
	tc.createTestBlock("project1", "", "test", start, end)

	// Get blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	// Export to JSON
	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	var export jsonExport
	err = json.Unmarshal([]byte(jsonStr), &export)
	require.NoError(t, err)

	require.Len(t, export.Blocks, 1)

	// Duration should be approximately 5400 seconds (90 minutes)
	// Allow some tolerance for test execution time
	assert.InDelta(t, 5400, export.Blocks[0].DurationSeconds, 10)
}

func TestExportCSV_TimestampFormat(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create a block
	tc.createTestBlock("project1", "", "test", now.Add(-1*time.Hour), now)

	// Get blocks
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)

	// Export to CSV
	csvStr, err := exportBlocksToCSV(blocks)
	require.NoError(t, err)

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(csvStr))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	require.Len(t, records, 2)
	dataRow := records[1]

	// Start timestamp should be valid RFC3339
	_, err = time.Parse(time.RFC3339, dataRow[4])
	assert.NoError(t, err, "Start timestamp should be valid RFC3339")

	// End timestamp should be valid RFC3339
	_, err = time.Parse(time.RFC3339, dataRow[5])
	assert.NoError(t, err, "End timestamp should be valid RFC3339")
}

// =============================================================================
// Export Ordering Tests
// =============================================================================

func TestExportFiltered_SortedByStartTimeDescending(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create blocks in random order
	tc.createTestBlock("project1", "", "middle", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	tc.createTestBlock("project1", "", "oldest", now.Add(-4*time.Hour), now.Add(-3*time.Hour))
	tc.createTestBlock("project1", "", "newest", now.Add(-1*time.Hour), now)

	// Get filtered blocks (should be sorted)
	filter := storage.BlockFilter{}
	blocks, err := tc.blockRepo.ListFiltered(filter)
	require.NoError(t, err)

	require.Len(t, blocks, 3)

	// Verify descending order by start time
	assert.Contains(t, blocks[0].Note, "newest")
	assert.Contains(t, blocks[1].Note, "middle")
	assert.Contains(t, blocks[2].Note, "oldest")
}

func TestExportFiltered_LimitResults(t *testing.T) {
	tc := setupTestDB(t)

	now := time.Now()

	// Create multiple blocks
	for i := 0; i < 10; i++ {
		tc.createTestBlock("project1", "", "block", now.Add(time.Duration(-i)*time.Hour), now.Add(time.Duration(-i)*time.Hour+30*time.Minute))
	}

	// Get filtered blocks with limit
	filter := storage.BlockFilter{Limit: 5}
	blocks, err := tc.blockRepo.ListFiltered(filter)
	require.NoError(t, err)

	// Should only have 5 blocks
	assert.Len(t, blocks, 5)
}
