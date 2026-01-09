// Package e2e provides end-to-end tests for Humantime CLI commands.
//
// These tests verify the complete behavior of CLI commands by running
// the actual binary and validating outputs. Tests use a temporary
// database directory that is cleaned up after each test.
package e2e

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testContext holds test configuration and provides helper methods.
type testContext struct {
	t         *testing.T
	binaryDir string
	dbDir     string
}

// setup creates a new test context with a temporary database directory.
// It returns the context and ensures the binary is built.
func setup(t *testing.T) *testContext {
	t.Helper()

	// Create temporary directory for test database
	dbDir, err := os.MkdirTemp("", "humantime-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Get the project root (two levels up from tests/e2e)
	wd, err := os.Getwd()
	if err != nil {
		os.RemoveAll(dbDir)
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Navigate to project root
	projectRoot := filepath.Join(wd, "..", "..")
	binaryPath := filepath.Join(projectRoot, "humantime")

	// Build the binary if it doesn't exist
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "humantime", ".")
		cmd.Dir = projectRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			os.RemoveAll(dbDir)
			t.Fatalf("failed to build binary: %v\n%s", err, output)
		}
	}

	return &testContext{
		t:         t,
		binaryDir: projectRoot,
		dbDir:     dbDir,
	}
}

// cleanup removes the temporary database directory.
func (tc *testContext) cleanup() {
	if tc.dbDir != "" {
		os.RemoveAll(tc.dbDir)
	}
}

// run executes the humantime CLI with the given arguments.
// It sets HUMANTIME_DATABASE to use the test database directory.
func (tc *testContext) run(args ...string) (stdout, stderr string, err error) {
	tc.t.Helper()

	binaryPath := filepath.Join(tc.binaryDir, "humantime")
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "HUMANTIME_DATABASE="+filepath.Join(tc.dbDir, "humantime.db"))

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// runJSON executes the humantime CLI with JSON output format.
func (tc *testContext) runJSON(args ...string) (stdout, stderr string, err error) {
	tc.t.Helper()
	allArgs := append([]string{"--format", "json"}, args...)
	return tc.run(allArgs...)
}

// mustRun executes the CLI and fails the test if there's an error.
func (tc *testContext) mustRun(args ...string) string {
	tc.t.Helper()
	stdout, stderr, err := tc.run(args...)
	if err != nil {
		tc.t.Fatalf("command failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	return stdout
}

// mustRunJSON executes the CLI with JSON output and fails the test if there's an error.
func (tc *testContext) mustRunJSON(args ...string) string {
	tc.t.Helper()
	stdout, stderr, err := tc.runJSON(args...)
	if err != nil {
		tc.t.Fatalf("command failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	return stdout
}

// parseJSON parses JSON output into the provided structure.
func (tc *testContext) parseJSON(stdout string, v interface{}) {
	tc.t.Helper()
	if err := json.Unmarshal([]byte(stdout), v); err != nil {
		tc.t.Fatalf("failed to parse JSON: %v\noutput: %s", err, stdout)
	}
}

// ============================================================================
// Start Command Tests
// ============================================================================

func TestStartOnNewProject(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Start tracking on a new project
	stdout := tc.mustRun("start", "on", "testproject")

	// Verify output mentions the project
	if !strings.Contains(stdout, "testproject") {
		t.Errorf("expected output to mention project name, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Started") {
		t.Errorf("expected output to indicate tracking started, got: %s", stdout)
	}

	// Verify with JSON that project was created and block is active
	jsonOut := tc.mustRunJSON("blocks")

	var blocksResp struct {
		Blocks []struct {
			ProjectSID string `json:"project_sid"`
			IsActive   bool   `json:"is_active"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocksResp.Blocks))
	}
	if blocksResp.Blocks[0].ProjectSID != "testproject" {
		t.Errorf("expected project 'testproject', got '%s'", blocksResp.Blocks[0].ProjectSID)
	}
	if !blocksResp.Blocks[0].IsActive {
		t.Error("expected block to be active")
	}
}

func TestStartWithTask(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Start tracking on project with task
	stdout := tc.mustRun("start", "on", "myproject/bugfix")

	// Verify output mentions both project and task
	if !strings.Contains(stdout, "myproject") {
		t.Errorf("expected output to mention project name, got: %s", stdout)
	}

	// Verify with JSON
	jsonOut := tc.mustRunJSON("blocks")

	var blocksResp struct {
		Blocks []struct {
			ProjectSID string `json:"project_sid"`
			TaskSID    string `json:"task_sid"`
			IsActive   bool   `json:"is_active"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocksResp.Blocks))
	}
	if blocksResp.Blocks[0].ProjectSID != "myproject" {
		t.Errorf("expected project 'myproject', got '%s'", blocksResp.Blocks[0].ProjectSID)
	}
	if blocksResp.Blocks[0].TaskSID != "bugfix" {
		t.Errorf("expected task 'bugfix', got '%s'", blocksResp.Blocks[0].TaskSID)
	}
}

func TestStartWithNote(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Start tracking with a note
	tc.mustRun("start", "on", "project1", "--note", "fixing login issue")

	// Verify with JSON
	jsonOut := tc.mustRunJSON("blocks")

	var blocksResp struct {
		Blocks []struct {
			ProjectSID string `json:"project_sid"`
			Note       string `json:"note"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocksResp.Blocks))
	}
	if blocksResp.Blocks[0].Note != "fixing login issue" {
		t.Errorf("expected note 'fixing login issue', got '%s'", blocksResp.Blocks[0].Note)
	}
}

func TestStartWhileTrackingSwitchesProjects(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Start tracking on first project
	tc.mustRun("start", "on", "project1")

	// Wait briefly to ensure time difference
	time.Sleep(100 * time.Millisecond)

	// Start tracking on second project (should switch)
	stdout := tc.mustRun("start", "on", "project2")

	// Verify output mentions stopping previous tracking
	if !strings.Contains(stdout, "project1") || !strings.Contains(stdout, "project2") {
		t.Logf("output: %s", stdout)
	}

	// Verify with JSON - should have 2 blocks, only second is active
	jsonOut := tc.mustRunJSON("blocks")

	var blocksResp struct {
		Blocks []struct {
			ProjectSID string `json:"project_sid"`
			IsActive   bool   `json:"is_active"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(blocksResp.Blocks))
	}

	// Count active blocks
	activeCount := 0
	var activeProject string
	for _, b := range blocksResp.Blocks {
		if b.IsActive {
			activeCount++
			activeProject = b.ProjectSID
		}
	}

	if activeCount != 1 {
		t.Errorf("expected exactly 1 active block, got %d", activeCount)
	}
	if activeProject != "project2" {
		t.Errorf("expected active project 'project2', got '%s'", activeProject)
	}
}

// ============================================================================
// Stop Command Tests
// ============================================================================

func TestStopEndsActiveTracking(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Start tracking
	tc.mustRun("start", "on", "testproject")

	// Wait briefly
	time.Sleep(100 * time.Millisecond)

	// Stop tracking
	stdout := tc.mustRun("stop")

	// Verify output indicates stopping
	if !strings.Contains(stdout, "Stopped") {
		t.Errorf("expected output to indicate tracking stopped, got: %s", stdout)
	}

	// Verify with JSON - block should no longer be active
	jsonOut := tc.mustRunJSON("blocks")

	var blocksResp struct {
		Blocks []struct {
			ProjectSID   string `json:"project_sid"`
			IsActive     bool   `json:"is_active"`
			TimestampEnd string `json:"timestamp_end"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocksResp.Blocks))
	}
	if blocksResp.Blocks[0].IsActive {
		t.Error("expected block to not be active after stop")
	}
	if blocksResp.Blocks[0].TimestampEnd == "" {
		t.Error("expected block to have an end timestamp after stop")
	}
}

func TestStopWithNoActiveTracking(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Try to stop with no active tracking
	stdout, _, _ := tc.run("stop")

	// Verify output indicates no active tracking
	if !strings.Contains(stdout, "No active tracking") && !strings.Contains(stdout, "no active") {
		t.Errorf("expected message about no active tracking, got: %s", stdout)
	}
}

func TestStopWithNote(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Start tracking
	tc.mustRun("start", "on", "testproject")

	// Stop with a note
	tc.mustRun("stop", "--note", "completed the feature")

	// Verify with JSON
	jsonOut := tc.mustRunJSON("blocks")

	var blocksResp struct {
		Blocks []struct {
			Note string `json:"note"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocksResp.Blocks))
	}
	if !strings.Contains(blocksResp.Blocks[0].Note, "completed the feature") {
		t.Errorf("expected note to contain 'completed the feature', got '%s'", blocksResp.Blocks[0].Note)
	}
}

func TestStopAppendsNoteToExisting(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Start tracking with initial note
	tc.mustRun("start", "on", "testproject", "--note", "initial note")

	// Stop with additional note
	tc.mustRun("stop", "--note", "additional note")

	// Verify with JSON - note should contain both parts
	jsonOut := tc.mustRunJSON("blocks")

	var blocksResp struct {
		Blocks []struct {
			Note string `json:"note"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocksResp.Blocks))
	}
	if !strings.Contains(blocksResp.Blocks[0].Note, "initial note") {
		t.Errorf("expected note to contain 'initial note', got '%s'", blocksResp.Blocks[0].Note)
	}
	if !strings.Contains(blocksResp.Blocks[0].Note, "additional note") {
		t.Errorf("expected note to contain 'additional note', got '%s'", blocksResp.Blocks[0].Note)
	}
}

// ============================================================================
// Blocks Command Tests
// ============================================================================

func TestBlocksListShowsAllBlocks(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create multiple blocks
	tc.mustRun("start", "on", "project1")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	tc.mustRun("start", "on", "project2")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	tc.mustRun("start", "on", "project3")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	// List all blocks
	jsonOut := tc.mustRunJSON("blocks")

	var blocksResp struct {
		Blocks     []struct{ ProjectSID string } `json:"blocks"`
		TotalCount int                           `json:"total_count"`
		ShownCount int                           `json:"shown_count"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if blocksResp.TotalCount != 3 {
		t.Errorf("expected total_count 3, got %d", blocksResp.TotalCount)
	}
	if blocksResp.ShownCount != 3 {
		t.Errorf("expected shown_count 3, got %d", blocksResp.ShownCount)
	}
	if len(blocksResp.Blocks) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(blocksResp.Blocks))
	}
}

func TestBlocksFilterByProject(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create blocks for different projects
	tc.mustRun("start", "on", "project1")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	tc.mustRun("start", "on", "project2")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	tc.mustRun("start", "on", "project1")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	// Filter by project1
	jsonOut := tc.mustRunJSON("blocks", "on", "project1")

	var blocksResp struct {
		Blocks []struct {
			ProjectSID string `json:"project_sid"`
		} `json:"blocks"`
		ShownCount int `json:"shown_count"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if blocksResp.ShownCount != 2 {
		t.Errorf("expected 2 blocks for project1, got %d", blocksResp.ShownCount)
	}
	for _, b := range blocksResp.Blocks {
		if b.ProjectSID != "project1" {
			t.Errorf("expected all blocks to be for project1, got '%s'", b.ProjectSID)
		}
	}
}

func TestBlocksFilterByProjectFlag(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create blocks
	tc.mustRun("start", "on", "alpha")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	tc.mustRun("start", "on", "beta")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	// Filter using --project flag
	jsonOut := tc.mustRunJSON("blocks", "--project", "beta")

	var blocksResp struct {
		Blocks []struct {
			ProjectSID string `json:"project_sid"`
		} `json:"blocks"`
		ShownCount int `json:"shown_count"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if blocksResp.ShownCount != 1 {
		t.Errorf("expected 1 block for beta, got %d", blocksResp.ShownCount)
	}
	if len(blocksResp.Blocks) > 0 && blocksResp.Blocks[0].ProjectSID != "beta" {
		t.Errorf("expected block for beta, got '%s'", blocksResp.Blocks[0].ProjectSID)
	}
}

// ============================================================================
// Project Command Tests
// ============================================================================

func TestProjectListShowsProjects(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create some projects by starting blocks
	tc.mustRun("start", "on", "project1")
	tc.mustRun("stop")
	tc.mustRun("start", "on", "project2")
	tc.mustRun("stop")

	// List projects
	jsonOut := tc.mustRunJSON("project")

	var projectsResp struct {
		Projects []struct {
			SID         string `json:"sid"`
			DisplayName string `json:"display_name"`
		} `json:"projects"`
	}
	tc.parseJSON(jsonOut, &projectsResp)

	if len(projectsResp.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projectsResp.Projects))
	}

	// Verify project names
	projectNames := make(map[string]bool)
	for _, p := range projectsResp.Projects {
		projectNames[p.SID] = true
	}
	if !projectNames["project1"] {
		t.Error("expected project1 in list")
	}
	if !projectNames["project2"] {
		t.Error("expected project2 in list")
	}
}

func TestProjectCreateWithColor(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create project with color using a custom SID to ensure consistent naming
	stdout := tc.mustRun("project", "create", "My Project", "--sid", "myproject", "--color", "#FF5733")

	// Verify output
	if !strings.Contains(stdout, "Created") {
		t.Errorf("expected output to indicate project created, got: %s", stdout)
	}

	// Verify with JSON
	jsonOut := tc.mustRunJSON("project", "myproject")

	var projectResp struct {
		SID         string `json:"sid"`
		DisplayName string `json:"display_name"`
		Color       string `json:"color"`
	}
	tc.parseJSON(jsonOut, &projectResp)

	if projectResp.SID != "myproject" {
		t.Errorf("expected SID 'myproject', got '%s'", projectResp.SID)
	}
	if projectResp.DisplayName != "My Project" {
		t.Errorf("expected display name 'My Project', got '%s'", projectResp.DisplayName)
	}
	if projectResp.Color != "#FF5733" {
		t.Errorf("expected color '#FF5733', got '%s'", projectResp.Color)
	}
}

func TestProjectCreateWithCustomSID(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create project with custom SID
	tc.mustRun("project", "create", "Client Work", "--sid", "clientwork")

	// Verify with JSON
	jsonOut := tc.mustRunJSON("project", "clientwork")

	var projectResp struct {
		SID         string `json:"sid"`
		DisplayName string `json:"display_name"`
	}
	tc.parseJSON(jsonOut, &projectResp)

	if projectResp.SID != "clientwork" {
		t.Errorf("expected SID 'clientwork', got '%s'", projectResp.SID)
	}
	if projectResp.DisplayName != "Client Work" {
		t.Errorf("expected display name 'Client Work', got '%s'", projectResp.DisplayName)
	}
}

// ============================================================================
// Stats Command Tests
// ============================================================================

func TestStatsShowsStatisticsForToday(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create some blocks for today with longer duration to ensure non-zero seconds
	tc.mustRun("start", "on", "project1")
	time.Sleep(1100 * time.Millisecond) // Over 1 second to ensure duration > 0
	tc.mustRun("stop")

	tc.mustRun("start", "on", "project2")
	time.Sleep(1100 * time.Millisecond)
	tc.mustRun("stop")

	// Get stats
	jsonOut := tc.mustRunJSON("stats")

	var statsResp struct {
		Period struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"period"`
		Summary struct {
			TotalDurationSeconds int64 `json:"total_duration_seconds"`
			ByProject            []struct {
				ProjectSID      string  `json:"project_sid"`
				DurationSeconds int64   `json:"duration_seconds"`
				Percentage      float64 `json:"percentage"`
			} `json:"by_project"`
		} `json:"summary"`
	}
	tc.parseJSON(jsonOut, &statsResp)

	// Verify period is today
	if statsResp.Period.Start == "" || statsResp.Period.End == "" {
		t.Error("expected period start and end to be set")
	}

	// Verify summary has data (duration should be at least 1 second per block)
	if statsResp.Summary.TotalDurationSeconds < 2 {
		t.Errorf("expected total duration to be at least 2 seconds, got %d", statsResp.Summary.TotalDurationSeconds)
	}
	if len(statsResp.Summary.ByProject) != 2 {
		t.Errorf("expected 2 projects in summary, got %d", len(statsResp.Summary.ByProject))
	}

	// Verify percentages add up to ~100%
	var totalPercentage float64
	for _, p := range statsResp.Summary.ByProject {
		totalPercentage += p.Percentage
	}
	if totalPercentage < 99 || totalPercentage > 101 {
		t.Errorf("expected percentages to add up to ~100%%, got %.2f%%", totalPercentage)
	}
}

func TestStatsFilterByProject(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create blocks for different projects
	tc.mustRun("start", "on", "project1")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	tc.mustRun("start", "on", "project2")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	// Get stats for project1 only
	jsonOut := tc.mustRunJSON("stats", "on", "project1")

	var statsResp struct {
		Summary struct {
			ByProject []struct {
				ProjectSID string `json:"project_sid"`
			} `json:"by_project"`
		} `json:"summary"`
	}
	tc.parseJSON(jsonOut, &statsResp)

	if len(statsResp.Summary.ByProject) != 1 {
		t.Errorf("expected 1 project in filtered stats, got %d", len(statsResp.Summary.ByProject))
	}
	if len(statsResp.Summary.ByProject) > 0 && statsResp.Summary.ByProject[0].ProjectSID != "project1" {
		t.Errorf("expected project1, got '%s'", statsResp.Summary.ByProject[0].ProjectSID)
	}
}

// ============================================================================
// Export Command Tests
// ============================================================================

func TestExportJSON(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create some blocks
	tc.mustRun("start", "on", "project1", "--note", "test note")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	tc.mustRun("start", "on", "project2/task1")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	// Export as JSON
	stdout := tc.mustRun("export", "--format", "json")

	// Parse the export
	var exportResp struct {
		Version    string `json:"version"`
		ExportedAt string `json:"exported_at"`
		Blocks     []struct {
			Key        string `json:"key"`
			ProjectSID string `json:"project_sid"`
			TaskSID    string `json:"task_sid"`
			Note       string `json:"note"`
		} `json:"blocks"`
		Count int `json:"count"`
	}
	tc.parseJSON(stdout, &exportResp)

	if exportResp.Version != "1" {
		t.Errorf("expected version '1', got '%s'", exportResp.Version)
	}
	if exportResp.ExportedAt == "" {
		t.Error("expected exported_at to be set")
	}
	if exportResp.Count != 2 {
		t.Errorf("expected 2 blocks, got %d", exportResp.Count)
	}
	if len(exportResp.Blocks) != 2 {
		t.Errorf("expected 2 blocks in array, got %d", len(exportResp.Blocks))
	}

	// Verify block data
	foundNote := false
	foundTask := false
	for _, b := range exportResp.Blocks {
		if b.Note == "test note" {
			foundNote = true
		}
		if b.TaskSID == "task1" {
			foundTask = true
		}
	}
	if !foundNote {
		t.Error("expected to find block with note 'test note'")
	}
	if !foundTask {
		t.Error("expected to find block with task 'task1'")
	}
}

func TestExportCSV(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create some blocks
	tc.mustRun("start", "on", "project1", "--note", "test note")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	tc.mustRun("start", "on", "project2/task1")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	// Export as CSV
	stdout := tc.mustRun("export", "--format", "csv")

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(stdout))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	// Should have header + 2 rows
	if len(records) != 3 {
		t.Errorf("expected 3 rows (header + 2 data), got %d", len(records))
	}

	// Verify header
	expectedHeaders := []string{"key", "project", "task", "note", "start", "end", "duration_seconds"}
	header := records[0]
	for i, h := range expectedHeaders {
		if i >= len(header) || header[i] != h {
			t.Errorf("expected header[%d] = '%s', got '%s'", i, h, header[i])
		}
	}

	// Verify data rows exist
	if len(records) >= 2 {
		// Find project column (index 1)
		projects := make(map[string]bool)
		for _, row := range records[1:] {
			if len(row) > 1 {
				projects[row[1]] = true
			}
		}
		if !projects["project1"] {
			t.Error("expected project1 in CSV data")
		}
		if !projects["project2"] {
			t.Error("expected project2 in CSV data")
		}
	}
}

func TestExportToFile(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create a block
	tc.mustRun("start", "on", "testproject")
	tc.mustRun("stop")

	// Export to file
	outputFile := filepath.Join(tc.dbDir, "export.json")
	tc.mustRun("export", "--format", "json", "-o", outputFile)

	// Read and verify file
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}

	var exportResp struct {
		Blocks []struct{ ProjectSID string } `json:"blocks"`
	}
	if err := json.Unmarshal(data, &exportResp); err != nil {
		t.Fatalf("failed to parse export file: %v", err)
	}

	if len(exportResp.Blocks) != 1 {
		t.Errorf("expected 1 block in export file, got %d", len(exportResp.Blocks))
	}
}

func TestExportFilterByProject(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Create blocks for different projects
	tc.mustRun("start", "on", "project1")
	tc.mustRun("stop")
	tc.mustRun("start", "on", "project2")
	tc.mustRun("stop")
	tc.mustRun("start", "on", "project1")
	tc.mustRun("stop")

	// Export only project1
	stdout := tc.mustRun("export", "--format", "json", "--project", "project1")

	var exportResp struct {
		Blocks []struct {
			ProjectSID string `json:"project_sid"`
		} `json:"blocks"`
		Count int `json:"count"`
	}
	tc.parseJSON(stdout, &exportResp)

	if exportResp.Count != 2 {
		t.Errorf("expected 2 blocks for project1, got %d", exportResp.Count)
	}
	for _, b := range exportResp.Blocks {
		if b.ProjectSID != "project1" {
			t.Errorf("expected all blocks to be for project1, got '%s'", b.ProjectSID)
		}
	}
}

// ============================================================================
// Version Command Tests
// ============================================================================

func TestVersionCommand(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Version command output goes to stdout (not stderr) through cobra's cmd.Printf
	stdout, stderr, err := tc.run("version")
	if err != nil {
		t.Fatalf("version command failed: %v, stderr: %s", err, stderr)
	}

	// Combine stdout and stderr as some output may go to either
	output := stdout + stderr

	if !strings.Contains(output, "humantime") {
		t.Errorf("expected version output to contain 'humantime', got stdout: %s, stderr: %s", stdout, stderr)
	}
	if !strings.Contains(output, "commit") {
		t.Errorf("expected version output to contain 'commit', got stdout: %s, stderr: %s", stdout, stderr)
	}
}

// ============================================================================
// Status Command Tests (root command with no args)
// ============================================================================

func TestStatusShowsNoActiveTracking(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	stdout := tc.mustRun()

	if !strings.Contains(stdout, "No active tracking") && !strings.Contains(stdout, "no active") {
		t.Errorf("expected message about no active tracking, got: %s", stdout)
	}
}

func TestStatusShowsActiveTracking(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	tc.mustRun("start", "on", "testproject")

	stdout := tc.mustRun()

	if !strings.Contains(stdout, "testproject") {
		t.Errorf("expected status to show active project, got: %s", stdout)
	}
	if !strings.Contains(stdout, "tracking") && !strings.Contains(stdout, "Currently") {
		t.Errorf("expected status to indicate active tracking, got: %s", stdout)
	}
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestStartRequiresProject(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	_, stderr, err := tc.run("start")

	if err == nil {
		t.Error("expected start without project to fail")
	}
	if !strings.Contains(stderr, "project") && !strings.Contains(stderr, "required") {
		t.Logf("stderr: %s", stderr)
	}
}

func TestInvalidSIDRejected(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Try to start with invalid SID (contains spaces or special chars)
	_, stderr, err := tc.run("start", "on", "invalid project!")

	if err == nil {
		t.Error("expected invalid SID to be rejected")
	}
	// The error should indicate invalid SID
	if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "SID") {
		t.Logf("stderr: %s", stderr)
	}
}

// ============================================================================
// Resume Command Tests
// ============================================================================

func TestResumeResumesLastProject(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Start and stop on a project
	tc.mustRun("start", "on", "myproject/mytask")
	time.Sleep(50 * time.Millisecond)
	tc.mustRun("stop")

	// Resume
	stdout := tc.mustRun("start", "resume")

	if !strings.Contains(stdout, "myproject") {
		t.Errorf("expected resume to show previous project, got: %s", stdout)
	}

	// Verify with JSON - should have new active block for same project
	jsonOut := tc.mustRunJSON("blocks")

	var blocksResp struct {
		Blocks []struct {
			ProjectSID string `json:"project_sid"`
			TaskSID    string `json:"task_sid"`
			IsActive   bool   `json:"is_active"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(blocksResp.Blocks))
	}

	// Find active block
	var activeBlock *struct {
		ProjectSID string `json:"project_sid"`
		TaskSID    string `json:"task_sid"`
		IsActive   bool   `json:"is_active"`
	}
	for i := range blocksResp.Blocks {
		if blocksResp.Blocks[i].IsActive {
			activeBlock = &blocksResp.Blocks[i]
			break
		}
	}

	if activeBlock == nil {
		t.Fatal("expected an active block after resume")
	}
	if activeBlock.ProjectSID != "myproject" {
		t.Errorf("expected active block project 'myproject', got '%s'", activeBlock.ProjectSID)
	}
	if activeBlock.TaskSID != "mytask" {
		t.Errorf("expected active block task 'mytask', got '%s'", activeBlock.TaskSID)
	}
}

// ============================================================================
// Alias Tests
// ============================================================================

func TestStartAliases(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Test 'sta' alias (more explicit than 's' which could conflict)
	stdout, stderr, err := tc.run("sta", "on", "testproject")
	if err != nil {
		t.Fatalf("'sta' alias failed: %v, stdout: %s, stderr: %s", err, stdout, stderr)
	}

	jsonOut := tc.mustRunJSON("blocks")
	var blocksResp struct {
		Blocks []struct {
			IsActive bool `json:"is_active"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocksResp.Blocks))
	}

	// Check that at least one block is active (blocks may not be sorted)
	hasActiveBlock := false
	for _, b := range blocksResp.Blocks {
		if b.IsActive {
			hasActiveBlock = true
			break
		}
	}
	if !hasActiveBlock {
		t.Error("expected at least one active block")
	}
}

func TestStopAliases(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	tc.mustRun("start", "on", "testproject")

	// Test 'e' alias (end)
	tc.mustRun("e")

	jsonOut := tc.mustRunJSON("blocks")
	var blocksResp struct {
		Blocks []struct {
			IsActive bool `json:"is_active"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 1 || blocksResp.Blocks[0].IsActive {
		t.Error("expected 'e' alias to stop tracking")
	}
}

// ============================================================================
// Concurrent Operations Tests
// ============================================================================

func TestRapidStartStop(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Rapidly start and stop multiple times
	for i := 0; i < 5; i++ {
		tc.mustRun("start", "on", "rapidtest")
		tc.mustRun("stop")
	}

	// Verify all blocks were created
	jsonOut := tc.mustRunJSON("blocks")
	var blocksResp struct {
		TotalCount int `json:"total_count"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if blocksResp.TotalCount != 5 {
		t.Errorf("expected 5 blocks, got %d", blocksResp.TotalCount)
	}
}

// ============================================================================
// Edge Cases Tests
// ============================================================================

func TestEmptyBlocksList(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	jsonOut := tc.mustRunJSON("blocks")

	var blocksResp struct {
		Blocks     []interface{} `json:"blocks"`
		TotalCount int           `json:"total_count"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if blocksResp.TotalCount != 0 {
		t.Errorf("expected 0 total_count, got %d", blocksResp.TotalCount)
	}
	if len(blocksResp.Blocks) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(blocksResp.Blocks))
	}
}

func TestEmptyProjectsList(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	jsonOut := tc.mustRunJSON("project")

	var projectsResp struct {
		Projects []interface{} `json:"projects"`
	}
	tc.parseJSON(jsonOut, &projectsResp)

	if len(projectsResp.Projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projectsResp.Projects))
	}
}

func TestLongProjectAndTaskNames(t *testing.T) {
	tc := setup(t)
	defer tc.cleanup()

	// Use reasonably long but valid SIDs
	longProject := "verylongprojectname"
	longTask := "verylongtaskname"

	tc.mustRun("start", "on", longProject+"/"+longTask)
	tc.mustRun("stop")

	jsonOut := tc.mustRunJSON("blocks")
	var blocksResp struct {
		Blocks []struct {
			ProjectSID string `json:"project_sid"`
			TaskSID    string `json:"task_sid"`
		} `json:"blocks"`
	}
	tc.parseJSON(jsonOut, &blocksResp)

	if len(blocksResp.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocksResp.Blocks))
	}
	if blocksResp.Blocks[0].ProjectSID != longProject {
		t.Errorf("expected project '%s', got '%s'", longProject, blocksResp.Blocks[0].ProjectSID)
	}
	if blocksResp.Blocks[0].TaskSID != longTask {
		t.Errorf("expected task '%s', got '%s'", longTask, blocksResp.Blocks[0].TaskSID)
	}
}
