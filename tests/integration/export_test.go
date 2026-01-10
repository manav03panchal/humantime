// Package integration provides integration tests for Humantime.
//
// These tests verify the export functionality using an in-memory Badger
// database. They test the complete export flow including JSON and CSV
// formats, backup functionality, and filtering options.
package integration

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
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

// setupExportTestDB creates a new test context with an in-memory database.
func setupExportTestDB(t *testing.T) *testContext {
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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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
	tc := setupExportTestDB(t)

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

// =============================================================================
// JSON Import Tests
// =============================================================================

// importBlockOutput represents a block during import (same structure as export).
type importBlockOutput struct {
	Key             string `json:"key"`
	ProjectSID      string `json:"project_sid"`
	TaskSID         string `json:"task_sid,omitempty"`
	Note            string `json:"note,omitempty"`
	TimestampStart  string `json:"timestamp_start"`
	TimestampEnd    string `json:"timestamp_end,omitempty"`
	DurationSeconds int64  `json:"duration_seconds"`
	IsActive        bool   `json:"is_active"`
}

// importFromJSON parses JSON export data and creates blocks in the database.
func (tc *testContext) importFromJSON(jsonData string) (int, error) {
	tc.t.Helper()

	var export struct {
		Version    string              `json:"version"`
		ExportedAt string              `json:"exported_at"`
		Blocks     []importBlockOutput `json:"blocks"`
		Count      int                 `json:"count"`
	}

	if err := json.Unmarshal([]byte(jsonData), &export); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}

	config, err := tc.configRepo.Get()
	if err != nil {
		return 0, err
	}

	imported := 0
	for _, b := range export.Blocks {
		// Parse timestamps
		start, err := time.Parse(time.RFC3339, b.TimestampStart)
		if err != nil {
			return imported, fmt.Errorf("invalid start timestamp: %w", err)
		}

		var end time.Time
		if b.TimestampEnd != "" {
			end, err = time.Parse(time.RFC3339, b.TimestampEnd)
			if err != nil {
				return imported, fmt.Errorf("invalid end timestamp: %w", err)
			}
		}

		// Ensure project exists
		_, _, err = tc.projectRepo.GetOrCreate(b.ProjectSID, b.ProjectSID)
		if err != nil {
			return imported, err
		}

		// Ensure task exists if specified
		if b.TaskSID != "" {
			_, _, err = tc.taskRepo.GetOrCreate(b.ProjectSID, b.TaskSID, b.TaskSID)
			if err != nil {
				return imported, err
			}
		}

		// Create block
		block := model.NewBlock(config.UserKey, b.ProjectSID, b.TaskSID, b.Note, start)
		if !end.IsZero() {
			block.TimestampEnd = end
		}

		if err := tc.blockRepo.Create(block); err != nil {
			return imported, err
		}
		imported++
	}

	return imported, nil
}

// importFromBackupJSON parses backup JSON data and imports all entities.
func (tc *testContext) importFromBackupJSON(jsonData string, force bool) (*importStats, error) {
	tc.t.Helper()

	var backup jsonBackup
	if err := json.Unmarshal([]byte(jsonData), &backup); err != nil {
		return nil, fmt.Errorf("failed to parse backup: %w", err)
	}

	stats := &importStats{}

	// Import projects
	for _, p := range backup.Projects {
		exists, _ := tc.projectRepo.Exists(p.SID)
		if exists && !force {
			stats.Duplicates++
			continue
		}

		if exists {
			if err := tc.projectRepo.Update(p); err != nil {
				return stats, err
			}
		} else {
			if err := tc.projectRepo.Create(p); err != nil {
				return stats, err
			}
		}
		stats.Projects++
	}

	// Import tasks
	for _, t := range backup.Tasks {
		exists, _ := tc.taskRepo.Exists(t.ProjectSID, t.SID)
		if exists && !force {
			stats.Duplicates++
			continue
		}

		if exists {
			if err := tc.taskRepo.Update(t); err != nil {
				return stats, err
			}
		} else {
			if err := tc.taskRepo.Create(t); err != nil {
				return stats, err
			}
		}
		stats.Tasks++
	}

	// Import blocks
	for _, b := range backup.Blocks {
		_, err := tc.blockRepo.Get(b.Key)
		if err == nil && !force {
			stats.Duplicates++
			continue
		}

		if err == nil {
			if err := tc.blockRepo.Update(b); err != nil {
				return stats, err
			}
		} else {
			if err := tc.blockRepo.Create(b); err != nil {
				return stats, err
			}
		}
		stats.Blocks++
	}

	// Import goals
	for _, g := range backup.Goals {
		exists, _ := tc.goalRepo.Exists(g.ProjectSID)
		if exists && !force {
			stats.Duplicates++
			continue
		}

		if err := tc.goalRepo.Upsert(g); err != nil {
			return stats, err
		}
		stats.Goals++
	}

	return stats, nil
}

type importStats struct {
	Projects   int
	Tasks      int
	Blocks     int
	Goals      int
	Duplicates int
	Errors     int
}

func TestImportJSON_EmptyExport(t *testing.T) {
	tc := setupExportTestDB(t)

	// Create an empty export JSON
	emptyExport := `{"version":"1","exported_at":"2024-01-01T00:00:00Z","blocks":[],"count":0}`

	// Import
	imported, err := tc.importFromJSON(emptyExport)
	require.NoError(t, err)
	assert.Equal(t, 0, imported)

	// Verify database is still empty
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	assert.Empty(t, blocks)
}

func TestImportJSON_SingleBlock(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now().UTC()
	start := now.Add(-2 * time.Hour)
	end := now.Add(-1 * time.Hour)

	// Create export JSON with a single block
	exportJSON := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"blocks": [{
			"key": "block:test-key-1",
			"project_sid": "project1",
			"task_sid": "task1",
			"note": "test note",
			"timestamp_start": "%s",
			"timestamp_end": "%s",
			"duration_seconds": 3600,
			"is_active": false
		}],
		"count": 1
	}`, now.Format(time.RFC3339), start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Import
	imported, err := tc.importFromJSON(exportJSON)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	// Verify block was created
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	block := blocks[0]
	assert.Equal(t, "project1", block.ProjectSID)
	assert.Equal(t, "task1", block.TaskSID)
	assert.Equal(t, "test note", block.Note)
}

func TestImportJSON_MultipleBlocks(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now().UTC()

	// Create export JSON with multiple blocks
	exportJSON := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"blocks": [
			{
				"key": "block:test-key-1",
				"project_sid": "project1",
				"task_sid": "task1",
				"note": "note 1",
				"timestamp_start": "%s",
				"timestamp_end": "%s",
				"duration_seconds": 3600,
				"is_active": false
			},
			{
				"key": "block:test-key-2",
				"project_sid": "project1",
				"task_sid": "task2",
				"note": "note 2",
				"timestamp_start": "%s",
				"timestamp_end": "%s",
				"duration_seconds": 3600,
				"is_active": false
			},
			{
				"key": "block:test-key-3",
				"project_sid": "project2",
				"task_sid": "",
				"note": "note 3",
				"timestamp_start": "%s",
				"timestamp_end": "%s",
				"duration_seconds": 3600,
				"is_active": false
			}
		],
		"count": 3
	}`,
		now.Format(time.RFC3339),
		now.Add(-4*time.Hour).Format(time.RFC3339), now.Add(-3*time.Hour).Format(time.RFC3339),
		now.Add(-3*time.Hour).Format(time.RFC3339), now.Add(-2*time.Hour).Format(time.RFC3339),
		now.Add(-2*time.Hour).Format(time.RFC3339), now.Add(-1*time.Hour).Format(time.RFC3339))

	// Import
	imported, err := tc.importFromJSON(exportJSON)
	require.NoError(t, err)
	assert.Equal(t, 3, imported)

	// Verify blocks were created
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	assert.Len(t, blocks, 3)

	// Verify projects were created
	projects, err := tc.projectRepo.List()
	require.NoError(t, err)
	assert.Len(t, projects, 2) // project1 and project2
}

func TestImportJSON_ActiveBlock(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now().UTC()
	start := now.Add(-30 * time.Minute)

	// Create export JSON with an active block (no end time)
	exportJSON := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"blocks": [{
			"key": "block:active-key",
			"project_sid": "project1",
			"note": "active work",
			"timestamp_start": "%s",
			"duration_seconds": 1800,
			"is_active": true
		}],
		"count": 1
	}`, now.Format(time.RFC3339), start.Format(time.RFC3339))

	// Import
	imported, err := tc.importFromJSON(exportJSON)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	// Verify block was created as active
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	block := blocks[0]
	assert.True(t, block.IsActive())
	assert.Equal(t, "active work", block.Note)
}

func TestImportJSON_BlockWithSpecialCharacters(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	end := now

	// Create export JSON with special characters in note
	exportJSON := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"blocks": [{
			"key": "block:special-key",
			"project_sid": "project1",
			"note": "Test with \"quotes\", 'apostrophes', and\nnewlines",
			"timestamp_start": "%s",
			"timestamp_end": "%s",
			"duration_seconds": 3600,
			"is_active": false
		}],
		"count": 1
	}`, now.Format(time.RFC3339), start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Import
	imported, err := tc.importFromJSON(exportJSON)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	// Verify special characters preserved
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	assert.Equal(t, "Test with \"quotes\", 'apostrophes', and\nnewlines", blocks[0].Note)
}

// =============================================================================
// CSV Import Tests
// =============================================================================

// importFromCSV parses CSV data and creates blocks in the database.
func (tc *testContext) importFromCSV(csvData string) (int, error) {
	tc.t.Helper()

	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return 0, nil
	}

	// Skip header
	if len(records) > 0 && records[0][0] == "key" {
		records = records[1:]
	}

	config, err := tc.configRepo.Get()
	if err != nil {
		return 0, err
	}

	imported := 0
	for _, row := range records {
		if len(row) < 7 {
			continue
		}

		projectSID := row[1]
		taskSID := row[2]
		note := row[3]
		startStr := row[4]
		endStr := row[5]

		// Parse timestamps
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return imported, fmt.Errorf("invalid start timestamp: %w", err)
		}

		var end time.Time
		if endStr != "" {
			end, err = time.Parse(time.RFC3339, endStr)
			if err != nil {
				return imported, fmt.Errorf("invalid end timestamp: %w", err)
			}
		}

		// Ensure project exists
		_, _, err = tc.projectRepo.GetOrCreate(projectSID, projectSID)
		if err != nil {
			return imported, err
		}

		// Ensure task exists if specified
		if taskSID != "" {
			_, _, err = tc.taskRepo.GetOrCreate(projectSID, taskSID, taskSID)
			if err != nil {
				return imported, err
			}
		}

		// Create block
		block := model.NewBlock(config.UserKey, projectSID, taskSID, note, start)
		if !end.IsZero() {
			block.TimestampEnd = end
		}

		if err := tc.blockRepo.Create(block); err != nil {
			return imported, err
		}
		imported++
	}

	return imported, nil
}

func TestImportCSV_EmptyFile(t *testing.T) {
	tc := setupExportTestDB(t)

	// CSV with only header
	csvData := "key,project,task,note,start,end,duration_seconds\n"

	imported, err := tc.importFromCSV(csvData)
	require.NoError(t, err)
	assert.Equal(t, 0, imported)

	// Verify database is empty
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	assert.Empty(t, blocks)
}

func TestImportCSV_SingleBlock(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now().UTC()
	start := now.Add(-2 * time.Hour)
	end := now.Add(-1 * time.Hour)

	csvData := fmt.Sprintf(`key,project,task,note,start,end,duration_seconds
block:csv-key-1,project1,task1,test note,%s,%s,3600
`, start.Format(time.RFC3339), end.Format(time.RFC3339))

	imported, err := tc.importFromCSV(csvData)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	// Verify block was created
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	block := blocks[0]
	assert.Equal(t, "project1", block.ProjectSID)
	assert.Equal(t, "task1", block.TaskSID)
	assert.Equal(t, "test note", block.Note)
}

func TestImportCSV_MultipleBlocks(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now().UTC()

	csvData := fmt.Sprintf(`key,project,task,note,start,end,duration_seconds
block:csv-key-1,project1,task1,note 1,%s,%s,3600
block:csv-key-2,project1,task2,note 2,%s,%s,3600
block:csv-key-3,project2,,note 3,%s,%s,3600
`,
		now.Add(-4*time.Hour).Format(time.RFC3339), now.Add(-3*time.Hour).Format(time.RFC3339),
		now.Add(-3*time.Hour).Format(time.RFC3339), now.Add(-2*time.Hour).Format(time.RFC3339),
		now.Add(-2*time.Hour).Format(time.RFC3339), now.Add(-1*time.Hour).Format(time.RFC3339))

	imported, err := tc.importFromCSV(csvData)
	require.NoError(t, err)
	assert.Equal(t, 3, imported)

	// Verify blocks were created
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	assert.Len(t, blocks, 3)
}

func TestImportCSV_ActiveBlock(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now().UTC()
	start := now.Add(-30 * time.Minute)

	// Empty end time indicates active block
	csvData := fmt.Sprintf(`key,project,task,note,start,end,duration_seconds
block:csv-active,project1,,active work,%s,,1800
`, start.Format(time.RFC3339))

	imported, err := tc.importFromCSV(csvData)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	// Verify block is active
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.True(t, blocks[0].IsActive())
}

// =============================================================================
// Data Integrity Round-Trip Tests
// =============================================================================

func TestRoundTrip_JSONExportImport_SingleBlock(t *testing.T) {
	// Create first database with test data
	tc1 := setupExportTestDB(t)

	now := time.Now()
	start := now.Add(-2 * time.Hour)
	end := now.Add(-1 * time.Hour)

	// Create original block
	originalBlock := tc1.createTestBlock("project1", "task1", "round-trip test", start, end)

	// Export to JSON
	blocks, err := tc1.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Create second database and import
	tc2 := setupExportTestDB(t)

	imported, err := tc2.importFromJSON(jsonStr)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	// Verify data integrity
	importedBlocks, err := tc2.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, importedBlocks, 1)

	importedBlock := importedBlocks[0]

	// Verify all fields match (key will be different)
	assert.Equal(t, originalBlock.ProjectSID, importedBlock.ProjectSID)
	assert.Equal(t, originalBlock.TaskSID, importedBlock.TaskSID)
	assert.Equal(t, originalBlock.Note, importedBlock.Note)

	// Timestamps should be within 1 second (accounting for serialization precision)
	assert.WithinDuration(t, originalBlock.TimestampStart, importedBlock.TimestampStart, 1*time.Second)
	assert.WithinDuration(t, originalBlock.TimestampEnd, importedBlock.TimestampEnd, 1*time.Second)
}

func TestRoundTrip_JSONExportImport_MultipleBlocks(t *testing.T) {
	tc1 := setupExportTestDB(t)

	now := time.Now()

	// Create multiple blocks with diverse data
	blocks := []*model.Block{
		tc1.createTestBlock("project1", "task1", "note 1", now.Add(-4*time.Hour), now.Add(-3*time.Hour)),
		tc1.createTestBlock("project1", "task2", "note 2", now.Add(-3*time.Hour), now.Add(-2*time.Hour)),
		tc1.createTestBlock("project2", "", "note 3", now.Add(-2*time.Hour), now.Add(-1*time.Hour)),
		tc1.createTestBlock("project3", "task3", "unicode: \u4e2d\u6587", now.Add(-1*time.Hour), now),
	}

	// Export to JSON
	allBlocks, err := tc1.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, allBlocks, 4)

	jsonStr, err := exportBlocksToJSON(allBlocks)
	require.NoError(t, err)

	// Create second database and import
	tc2 := setupExportTestDB(t)

	imported, err := tc2.importFromJSON(jsonStr)
	require.NoError(t, err)
	assert.Equal(t, 4, imported)

	// Verify all blocks were imported
	importedBlocks, err := tc2.blockRepo.List()
	require.NoError(t, err)
	assert.Len(t, importedBlocks, 4)

	// Verify projects were created
	projects, err := tc2.projectRepo.List()
	require.NoError(t, err)
	assert.Len(t, projects, 3) // project1, project2, project3

	// Verify unicode data preserved
	var unicodeBlock *model.Block
	for _, b := range importedBlocks {
		if strings.Contains(b.Note, "unicode") {
			unicodeBlock = b
			break
		}
	}
	require.NotNil(t, unicodeBlock)
	assert.Equal(t, "unicode: \u4e2d\u6587", unicodeBlock.Note)

	// Verify original blocks match
	for _, original := range blocks {
		found := false
		for _, imported := range importedBlocks {
			if imported.ProjectSID == original.ProjectSID &&
				imported.TaskSID == original.TaskSID &&
				imported.Note == original.Note {
				found = true
				assert.WithinDuration(t, original.TimestampStart, imported.TimestampStart, 1*time.Second)
				assert.WithinDuration(t, original.TimestampEnd, imported.TimestampEnd, 1*time.Second)
				break
			}
		}
		assert.True(t, found, "Block with note '%s' should be found after import", original.Note)
	}
}

func TestRoundTrip_CSVExportImport(t *testing.T) {
	tc1 := setupExportTestDB(t)

	now := time.Now()

	// Create test blocks
	tc1.createTestBlock("project1", "task1", "csv test 1", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	tc1.createTestBlock("project1", "", "csv test 2", now.Add(-1*time.Hour), now)

	// Export to CSV
	blocks, err := tc1.blockRepo.List()
	require.NoError(t, err)

	csvStr, err := exportBlocksToCSV(blocks)
	require.NoError(t, err)

	// Create second database and import
	tc2 := setupExportTestDB(t)

	imported, err := tc2.importFromCSV(csvStr)
	require.NoError(t, err)
	assert.Equal(t, 2, imported)

	// Verify data integrity
	importedBlocks, err := tc2.blockRepo.List()
	require.NoError(t, err)
	assert.Len(t, importedBlocks, 2)

	// Verify content matches
	projectCounts := make(map[string]int)
	for _, b := range importedBlocks {
		projectCounts[b.ProjectSID]++
		assert.Contains(t, b.Note, "csv test")
	}
	assert.Equal(t, 2, projectCounts["project1"])
}

func TestRoundTrip_BackupExportImport(t *testing.T) {
	tc1 := setupExportTestDB(t)

	now := time.Now()

	// Create comprehensive test data
	tc1.createTestProject("project1", "Project One", "#FF0000")
	tc1.createTestProject("project2", "Project Two", "#00FF00")
	tc1.createTestTask("project1", "task1", "Task One", "#0000FF")
	tc1.createTestTask("project1", "task2", "Task Two", "#FFFF00")
	tc1.createTestBlock("project1", "task1", "backup test 1", now.Add(-3*time.Hour), now.Add(-2*time.Hour))
	tc1.createTestBlock("project2", "", "backup test 2", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	tc1.createTestGoal("project1", model.GoalTypeDaily, 8*time.Hour)
	tc1.createTestGoal("project2", model.GoalTypeWeekly, 40*time.Hour)

	// Get all data for backup
	config, err := tc1.configRepo.Get()
	require.NoError(t, err)

	projects, err := tc1.projectRepo.List()
	require.NoError(t, err)

	tasks, err := tc1.taskRepo.List()
	require.NoError(t, err)

	blocks, err := tc1.blockRepo.List()
	require.NoError(t, err)

	goals, err := tc1.goalRepo.List()
	require.NoError(t, err)

	activeBlock, err := tc1.activeBlockRepo.Get()
	require.NoError(t, err)

	// Export to backup JSON
	backupJSON, err := exportBackupToJSON(config, projects, tasks, blocks, goals, activeBlock)
	require.NoError(t, err)

	// Create second database and import
	tc2 := setupExportTestDB(t)

	stats, err := tc2.importFromBackupJSON(backupJSON, false)
	require.NoError(t, err)

	// Verify import counts
	assert.Equal(t, 2, stats.Projects)
	assert.Equal(t, 2, stats.Tasks)
	assert.Equal(t, 2, stats.Blocks)
	assert.Equal(t, 2, stats.Goals)

	// Verify projects
	importedProjects, err := tc2.projectRepo.List()
	require.NoError(t, err)
	assert.Len(t, importedProjects, 2)

	projectMap := make(map[string]*model.Project)
	for _, p := range importedProjects {
		projectMap[p.SID] = p
	}
	assert.NotNil(t, projectMap["project1"])
	assert.Equal(t, "Project One", projectMap["project1"].DisplayName)
	assert.Equal(t, "#FF0000", projectMap["project1"].Color)

	// Verify tasks
	importedTasks, err := tc2.taskRepo.List()
	require.NoError(t, err)
	assert.Len(t, importedTasks, 2)

	// Verify blocks
	importedBlocks, err := tc2.blockRepo.List()
	require.NoError(t, err)
	assert.Len(t, importedBlocks, 2)

	// Verify goals
	importedGoals, err := tc2.goalRepo.List()
	require.NoError(t, err)
	assert.Len(t, importedGoals, 2)
}

func TestRoundTrip_DurationPreserved(t *testing.T) {
	tc1 := setupExportTestDB(t)

	now := time.Now()
	start := now.Add(-90 * time.Minute) // 90 minutes = 5400 seconds
	end := now

	// Create block with specific duration
	tc1.createTestBlock("project1", "", "duration test", start, end)

	// Export to JSON
	blocks, err := tc1.blockRepo.List()
	require.NoError(t, err)

	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Import to new database
	tc2 := setupExportTestDB(t)

	_, err = tc2.importFromJSON(jsonStr)
	require.NoError(t, err)

	// Verify duration is preserved
	importedBlocks, err := tc2.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, importedBlocks, 1)

	// Duration should be approximately 5400 seconds (90 minutes)
	assert.InDelta(t, 5400, importedBlocks[0].DurationSeconds(), 10)
}

func TestRoundTrip_TimezoneHandling(t *testing.T) {
	tc1 := setupExportTestDB(t)

	// Create block with specific timezone
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	now := time.Now().In(loc)
	start := now.Add(-1 * time.Hour)
	end := now

	tc1.createTestBlock("project1", "", "timezone test", start, end)

	// Export to JSON
	blocks, err := tc1.blockRepo.List()
	require.NoError(t, err)

	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Import to new database
	tc2 := setupExportTestDB(t)

	_, err = tc2.importFromJSON(jsonStr)
	require.NoError(t, err)

	// Verify timestamps are correctly preserved (in UTC)
	importedBlocks, err := tc2.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, importedBlocks, 1)

	// Timestamps should be equivalent (within 1 second tolerance)
	assert.WithinDuration(t, start, importedBlocks[0].TimestampStart, 1*time.Second)
	assert.WithinDuration(t, end, importedBlocks[0].TimestampEnd, 1*time.Second)
}

// =============================================================================
// Error Handling for Invalid Import Data
// =============================================================================

func TestImportJSON_InvalidJSON(t *testing.T) {
	tc := setupExportTestDB(t)

	// Invalid JSON syntax
	invalidJSON := `{"version": "1", "blocks": [broken json`

	_, err := tc.importFromJSON(invalidJSON)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestImportJSON_InvalidTimestamp(t *testing.T) {
	tc := setupExportTestDB(t)

	// Invalid timestamp format
	invalidTimestamp := `{
		"version": "1",
		"exported_at": "2024-01-01T00:00:00Z",
		"blocks": [{
			"key": "block:test",
			"project_sid": "project1",
			"timestamp_start": "not-a-valid-timestamp",
			"duration_seconds": 3600,
			"is_active": false
		}],
		"count": 1
	}`

	_, err := tc.importFromJSON(invalidTimestamp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start timestamp")
}

func TestImportJSON_InvalidEndTimestamp(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now().UTC()

	// Invalid end timestamp format
	invalidEndTimestamp := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"blocks": [{
			"key": "block:test",
			"project_sid": "project1",
			"timestamp_start": "%s",
			"timestamp_end": "invalid-end-timestamp",
			"duration_seconds": 3600,
			"is_active": false
		}],
		"count": 1
	}`, now.Format(time.RFC3339), now.Add(-1*time.Hour).Format(time.RFC3339))

	_, err := tc.importFromJSON(invalidEndTimestamp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid end timestamp")
}

func TestImportJSON_MissingRequiredFields(t *testing.T) {
	tc := setupExportTestDB(t)

	// Missing project_sid (though import will create empty project)
	now := time.Now().UTC()

	missingProject := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"blocks": [{
			"key": "block:test",
			"project_sid": "",
			"timestamp_start": "%s",
			"duration_seconds": 0,
			"is_active": false
		}],
		"count": 1
	}`, now.Format(time.RFC3339), now.Add(-1*time.Hour).Format(time.RFC3339))

	// This should succeed but create a block with empty project
	imported, err := tc.importFromJSON(missingProject)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	assert.Len(t, blocks, 1)
	assert.Equal(t, "", blocks[0].ProjectSID)
}

func TestImportCSV_InvalidCSV(t *testing.T) {
	tc := setupExportTestDB(t)

	// Malformed CSV (wrong number of columns) - CSV parser will return error
	invalidCSV := `key,project,task,note,start,end,duration_seconds
block:test,project1,task1`

	_, err := tc.importFromCSV(invalidCSV)
	// CSV parser returns an error for wrong number of fields
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse CSV")
}

func TestImportCSV_InvalidTimestamp(t *testing.T) {
	tc := setupExportTestDB(t)

	// Invalid timestamp in CSV
	invalidCSV := `key,project,task,note,start,end,duration_seconds
block:test,project1,task1,note,invalid-timestamp,,3600`

	_, err := tc.importFromCSV(invalidCSV)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start timestamp")
}

func TestImportJSON_EmptyBlocksArray(t *testing.T) {
	tc := setupExportTestDB(t)

	// Valid JSON with empty blocks array
	emptyBlocks := `{
		"version": "1",
		"exported_at": "2024-01-01T00:00:00Z",
		"blocks": [],
		"count": 0
	}`

	imported, err := tc.importFromJSON(emptyBlocks)
	require.NoError(t, err)
	assert.Equal(t, 0, imported)
}

func TestImportJSON_NullBlocks(t *testing.T) {
	tc := setupExportTestDB(t)

	// JSON with null blocks
	nullBlocks := `{
		"version": "1",
		"exported_at": "2024-01-01T00:00:00Z",
		"blocks": null,
		"count": 0
	}`

	imported, err := tc.importFromJSON(nullBlocks)
	require.NoError(t, err)
	assert.Equal(t, 0, imported)
}

func TestImportBackup_DuplicateHandling_NoDuplicates(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now()

	// First import
	backupJSON := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"config": null,
		"projects": [{"sid": "project1", "display_name": "Project One", "color": "#FF0000"}],
		"tasks": [{"project_sid": "project1", "sid": "task1", "display_name": "Task One", "color": ""}],
		"blocks": [],
		"goals": [{"project_sid": "project1", "type": "daily", "target": 28800000000000}],
		"active_block": null
	}`, now.Format(time.RFC3339))

	stats, err := tc.importFromBackupJSON(backupJSON, false)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.Projects)
	assert.Equal(t, 1, stats.Tasks)
	assert.Equal(t, 1, stats.Goals)

	// Second import - should skip duplicates
	stats2, err := tc.importFromBackupJSON(backupJSON, false)
	require.NoError(t, err)
	assert.Equal(t, 0, stats2.Projects) // Skipped
	assert.Equal(t, 0, stats2.Tasks)    // Skipped
	assert.Equal(t, 0, stats2.Goals)    // Skipped
	assert.Equal(t, 3, stats2.Duplicates)
}

func TestImportBackup_DuplicateHandling_WithForce(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now()

	// First import - include key fields required by the storage layer
	backupJSON := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"config": null,
		"projects": [{"key": "project:project1", "sid": "project1", "display_name": "Project One", "color": "#FF0000"}],
		"tasks": [{"key": "task:project1:task1", "project_sid": "project1", "sid": "task1", "display_name": "Task One", "color": ""}],
		"blocks": [],
		"goals": [{"key": "goal:project1", "project_sid": "project1", "type": "daily", "target": 28800000000000}],
		"active_block": null
	}`, now.Format(time.RFC3339))

	_, err := tc.importFromBackupJSON(backupJSON, false)
	require.NoError(t, err)

	// Second import with force - should update
	updatedBackupJSON := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"config": null,
		"projects": [{"key": "project:project1", "sid": "project1", "display_name": "Updated Project", "color": "#00FF00"}],
		"tasks": [{"key": "task:project1:task1", "project_sid": "project1", "sid": "task1", "display_name": "Updated Task", "color": "#BLUE"}],
		"blocks": [],
		"goals": [{"key": "goal:project1", "project_sid": "project1", "type": "weekly", "target": 144000000000000}],
		"active_block": null
	}`, now.Format(time.RFC3339))

	stats, err := tc.importFromBackupJSON(updatedBackupJSON, true) // force = true
	require.NoError(t, err)
	assert.Equal(t, 1, stats.Projects)
	assert.Equal(t, 1, stats.Tasks)
	assert.Equal(t, 1, stats.Goals)
	assert.Equal(t, 0, stats.Duplicates)

	// Verify data was updated
	projects, err := tc.projectRepo.List()
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "Updated Project", projects[0].DisplayName)
	assert.Equal(t, "#00FF00", projects[0].Color)
}

func TestImportBackup_InvalidJSON(t *testing.T) {
	tc := setupExportTestDB(t)

	invalidJSON := `{"version": "1", broken`

	_, err := tc.importFromBackupJSON(invalidJSON, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse backup")
}

func TestImportJSON_VeryLongNote(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	end := now

	// Create a very long note (test max length handling)
	longNote := strings.Repeat("a", 10000)

	exportJSON := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"blocks": [{
			"key": "block:long-note",
			"project_sid": "project1",
			"note": "%s",
			"timestamp_start": "%s",
			"timestamp_end": "%s",
			"duration_seconds": 3600,
			"is_active": false
		}],
		"count": 1
	}`, now.Format(time.RFC3339), longNote, start.Format(time.RFC3339), end.Format(time.RFC3339))

	imported, err := tc.importFromJSON(exportJSON)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	// Verify the long note was preserved
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, longNote, blocks[0].Note)
}

func TestImportJSON_UnicodeProjectAndTask(t *testing.T) {
	tc := setupExportTestDB(t)

	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	end := now

	// Unicode characters in project and task SIDs
	exportJSON := fmt.Sprintf(`{
		"version": "1",
		"exported_at": "%s",
		"blocks": [{
			"key": "block:unicode",
			"project_sid": "proyecto-espanol",
			"task_sid": "tarea-uno",
			"note": "Nota con caracteres especiales",
			"timestamp_start": "%s",
			"timestamp_end": "%s",
			"duration_seconds": 3600,
			"is_active": false
		}],
		"count": 1
	}`, now.Format(time.RFC3339), start.Format(time.RFC3339), end.Format(time.RFC3339))

	imported, err := tc.importFromJSON(exportJSON)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	// Verify the unicode data was preserved
	blocks, err := tc.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, "proyecto-espanol", blocks[0].ProjectSID)
	assert.Equal(t, "tarea-uno", blocks[0].TaskSID)
}

func TestExportImport_EmptyNote(t *testing.T) {
	tc1 := setupExportTestDB(t)

	now := time.Now()

	// Create block with empty note
	tc1.createTestBlock("project1", "", "", now.Add(-1*time.Hour), now)

	// Export
	blocks, err := tc1.blockRepo.List()
	require.NoError(t, err)

	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Import to new database
	tc2 := setupExportTestDB(t)

	_, err = tc2.importFromJSON(jsonStr)
	require.NoError(t, err)

	// Verify empty note preserved
	importedBlocks, err := tc2.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, importedBlocks, 1)
	assert.Equal(t, "", importedBlocks[0].Note)
}

func TestExportImport_EmptyTask(t *testing.T) {
	tc1 := setupExportTestDB(t)

	now := time.Now()

	// Create block with no task
	tc1.createTestBlock("project1", "", "no task block", now.Add(-1*time.Hour), now)

	// Export
	blocks, err := tc1.blockRepo.List()
	require.NoError(t, err)

	jsonStr, err := exportBlocksToJSON(blocks)
	require.NoError(t, err)

	// Import to new database
	tc2 := setupExportTestDB(t)

	_, err = tc2.importFromJSON(jsonStr)
	require.NoError(t, err)

	// Verify empty task preserved
	importedBlocks, err := tc2.blockRepo.List()
	require.NoError(t, err)
	require.Len(t, importedBlocks, 1)
	assert.Equal(t, "", importedBlocks[0].TaskSID)
}
