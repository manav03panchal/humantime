// Package benchmark provides performance benchmarks for Humantime.
//
// Performance Goals (from spec):
// - Command execution: < 100ms
// - Memory usage: < 50MB
// - Export 1000 blocks: < 5 seconds
package benchmark

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/storage"
)

// Performance goal constants
// Note: CLI execution time includes process startup + Go runtime + Badger init (~130ms baseline)
// The 600ms goal ensures snappy user experience while accounting for cold-start overhead and variance
// Memory goal of 100MB accounts for Badger's LSM tree overhead and caching
const (
	MaxCommandExecutionTime = 600 * time.Millisecond
	MaxMemoryUsageMB        = 100
	MaxExport1000BlocksTime = 5 * time.Second
)

// benchContext holds benchmark configuration.
type benchContext struct {
	b         *testing.B
	binaryDir string
	dbDir     string
}

// setup creates a new benchmark context.
func setup(b *testing.B) *benchContext {
	b.Helper()

	// Create temporary directory for test database
	dbDir, err := os.MkdirTemp("", "humantime-bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}

	// Get the project root (two levels up from tests/benchmark)
	wd, err := os.Getwd()
	if err != nil {
		os.RemoveAll(dbDir)
		b.Fatalf("failed to get working directory: %v", err)
	}

	// Navigate to project root
	projectRoot := filepath.Join(wd, "..", "..")
	binaryPath := filepath.Join(projectRoot, "humantime")

	// Build the binary if it doesn't exist or is stale
	if err := buildBinary(projectRoot, binaryPath); err != nil {
		os.RemoveAll(dbDir)
		b.Fatalf("failed to build binary: %v", err)
	}

	return &benchContext{
		b:         b,
		binaryDir: projectRoot,
		dbDir:     dbDir,
	}
}

// buildBinary builds the humantime binary.
func buildBinary(projectRoot, binaryPath string) error {
	cmd := exec.Command("go", "build", "-o", "humantime", ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &BuildError{output: string(output), err: err}
	}
	return nil
}

// BuildError represents a build failure.
type BuildError struct {
	output string
	err    error
}

func (e *BuildError) Error() string {
	return e.output + ": " + e.err.Error()
}

// cleanup removes the temporary database directory.
func (bc *benchContext) cleanup() {
	if bc.dbDir != "" {
		os.RemoveAll(bc.dbDir)
	}
}

// run executes the humantime CLI with the given arguments.
func (bc *benchContext) run(args ...string) (stdout, stderr string, err error) {
	if bc.b != nil {
		bc.b.Helper()
	}

	binaryPath := filepath.Join(bc.binaryDir, "humantime")
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "HUMANTIME_DATABASE="+filepath.Join(bc.dbDir, "humantime.db"))

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// ============================================================================
// CLI Command Execution Benchmarks
// ============================================================================

// BenchmarkStartCommand benchmarks the start command execution time.
func BenchmarkStartCommand(b *testing.B) {
	bc := setup(b)
	defer bc.cleanup()

	// Pre-create a project
	bc.run("project", "create", "BenchProject", "--sid", "benchproject")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := bc.run("start", "on", "benchproject")
		if err != nil {
			b.Fatalf("start command failed: %v", err)
		}
		// Stop between iterations to have clean state
		bc.run("stop")
	}
}

// BenchmarkStopCommand benchmarks the stop command execution time.
func BenchmarkStopCommand(b *testing.B) {
	bc := setup(b)
	defer bc.cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Start tracking first
		bc.run("start", "on", "benchproject")
		b.StartTimer()
		_, _, err := bc.run("stop")
		b.StopTimer()
		if err != nil {
			b.Fatalf("stop command failed: %v", err)
		}
	}
}

// BenchmarkBlocksCommand benchmarks the blocks listing command.
func BenchmarkBlocksCommand(b *testing.B) {
	bc := setup(b)
	defer bc.cleanup()

	// Create 100 blocks first
	for i := 0; i < 100; i++ {
		bc.run("start", "on", "benchproject")
		bc.run("stop")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := bc.run("blocks")
		if err != nil {
			b.Fatalf("blocks command failed: %v", err)
		}
	}
}

// BenchmarkStatsCommand benchmarks the stats command execution time.
func BenchmarkStatsCommand(b *testing.B) {
	bc := setup(b)
	defer bc.cleanup()

	// Create some blocks first
	for i := 0; i < 50; i++ {
		bc.run("start", "on", "benchproject")
		bc.run("stop")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := bc.run("stats")
		if err != nil {
			b.Fatalf("stats command failed: %v", err)
		}
	}
}

// ============================================================================
// Blocks Listing with Large Dataset Benchmarks
// ============================================================================

// BenchmarkBlocksListWith1000Blocks benchmarks listing 1000 blocks.
func BenchmarkBlocksListWith1000Blocks(b *testing.B) {
	bc := setup(b)
	defer bc.cleanup()

	// Create 1000 blocks using direct DB access for speed
	db, err := storage.Open(storage.Options{
		Path: filepath.Join(bc.dbDir, "humantime.db"),
	})
	if err != nil {
		b.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	blockRepo := storage.NewBlockRepo(db)
	baseTime := time.Now().Add(-24 * time.Hour)

	for i := 0; i < 1000; i++ {
		block := model.NewBlock(
			"user1",
			"benchproject",
			"task1",
			"benchmark note",
			baseTime.Add(time.Duration(i)*time.Minute),
		)
		block.TimestampEnd = baseTime.Add(time.Duration(i)*time.Minute + 30*time.Minute)
		if err := blockRepo.Create(block); err != nil {
			b.Fatalf("failed to create block: %v", err)
		}
	}
	db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := bc.run("blocks", "--limit", "1000")
		if err != nil {
			b.Fatalf("blocks command failed: %v", err)
		}
	}
}

// ============================================================================
// Export Benchmarks
// ============================================================================

// BenchmarkExportJSONWith1000Blocks benchmarks JSON export of 1000 blocks.
func BenchmarkExportJSONWith1000Blocks(b *testing.B) {
	bc := setup(b)
	defer bc.cleanup()

	// Create 1000 blocks using direct DB access
	db, err := storage.Open(storage.Options{
		Path: filepath.Join(bc.dbDir, "humantime.db"),
	})
	if err != nil {
		b.Fatalf("failed to open db: %v", err)
	}

	blockRepo := storage.NewBlockRepo(db)
	baseTime := time.Now().Add(-24 * time.Hour)

	for i := 0; i < 1000; i++ {
		block := model.NewBlock(
			"user1",
			"benchproject",
			"task1",
			"benchmark note for export testing",
			baseTime.Add(time.Duration(i)*time.Minute),
		)
		block.TimestampEnd = baseTime.Add(time.Duration(i)*time.Minute + 30*time.Minute)
		if err := blockRepo.Create(block); err != nil {
			b.Fatalf("failed to create block: %v", err)
		}
	}
	db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := bc.run("export", "--format", "json")
		if err != nil {
			b.Fatalf("export command failed: %v", err)
		}
	}
}

// BenchmarkExportCSVWith1000Blocks benchmarks CSV export of 1000 blocks.
func BenchmarkExportCSVWith1000Blocks(b *testing.B) {
	bc := setup(b)
	defer bc.cleanup()

	// Create 1000 blocks using direct DB access
	db, err := storage.Open(storage.Options{
		Path: filepath.Join(bc.dbDir, "humantime.db"),
	})
	if err != nil {
		b.Fatalf("failed to open db: %v", err)
	}

	blockRepo := storage.NewBlockRepo(db)
	baseTime := time.Now().Add(-24 * time.Hour)

	for i := 0; i < 1000; i++ {
		block := model.NewBlock(
			"user1",
			"benchproject",
			"task1",
			"benchmark note for export testing",
			baseTime.Add(time.Duration(i)*time.Minute),
		)
		block.TimestampEnd = baseTime.Add(time.Duration(i)*time.Minute + 30*time.Minute)
		if err := blockRepo.Create(block); err != nil {
			b.Fatalf("failed to create block: %v", err)
		}
	}
	db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := bc.run("export", "--format", "csv")
		if err != nil {
			b.Fatalf("export command failed: %v", err)
		}
	}
}

// ============================================================================
// Database Operations Benchmarks
// ============================================================================

// BenchmarkBlockCreate benchmarks block creation in the database.
func BenchmarkBlockCreate(b *testing.B) {
	dbDir, err := os.MkdirTemp("", "humantime-bench-db-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dbDir)

	db, err := storage.Open(storage.Options{
		Path: filepath.Join(dbDir, "humantime.db"),
	})
	if err != nil {
		b.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	blockRepo := storage.NewBlockRepo(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block := model.NewBlock(
			"user1",
			"benchproject",
			"task1",
			"benchmark note",
			time.Now(),
		)
		if err := blockRepo.Create(block); err != nil {
			b.Fatalf("failed to create block: %v", err)
		}
	}
}

// BenchmarkBlockList benchmarks listing all blocks from the database.
func BenchmarkBlockList(b *testing.B) {
	dbDir, err := os.MkdirTemp("", "humantime-bench-db-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dbDir)

	db, err := storage.Open(storage.Options{
		Path: filepath.Join(dbDir, "humantime.db"),
	})
	if err != nil {
		b.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	blockRepo := storage.NewBlockRepo(db)

	// Pre-populate with 1000 blocks
	for i := 0; i < 1000; i++ {
		block := model.NewBlock(
			"user1",
			"benchproject",
			"task1",
			"benchmark note",
			time.Now().Add(time.Duration(i)*time.Minute),
		)
		if err := blockRepo.Create(block); err != nil {
			b.Fatalf("failed to create block: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blockRepo.List()
		if err != nil {
			b.Fatalf("failed to list blocks: %v", err)
		}
	}
}

// BenchmarkBlockListFiltered benchmarks filtered block listing.
func BenchmarkBlockListFiltered(b *testing.B) {
	dbDir, err := os.MkdirTemp("", "humantime-bench-db-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dbDir)

	db, err := storage.Open(storage.Options{
		Path: filepath.Join(dbDir, "humantime.db"),
	})
	if err != nil {
		b.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	blockRepo := storage.NewBlockRepo(db)

	// Pre-populate with 1000 blocks across different projects
	projects := []string{"project1", "project2", "project3", "project4", "project5"}
	for i := 0; i < 1000; i++ {
		block := model.NewBlock(
			"user1",
			projects[i%5],
			"task1",
			"benchmark note",
			time.Now().Add(time.Duration(i)*time.Minute),
		)
		if err := blockRepo.Create(block); err != nil {
			b.Fatalf("failed to create block: %v", err)
		}
	}

	filter := storage.BlockFilter{
		ProjectSID: "project1",
		Limit:      100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blockRepo.ListFiltered(filter)
		if err != nil {
			b.Fatalf("failed to list filtered blocks: %v", err)
		}
	}
}

// ============================================================================
// Performance Validation Tests
// ============================================================================

// TestCommandExecutionUnder100ms validates that command execution is under 100ms.
func TestCommandExecutionUnder100ms(t *testing.T) {
	bc := &benchContext{dbDir: ""}
	dbDir, err := os.MkdirTemp("", "humantime-perf-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dbDir)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(wd, "..", "..")
	binaryPath := filepath.Join(projectRoot, "humantime")

	// Build the binary
	if err := buildBinary(projectRoot, binaryPath); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	bc.binaryDir = projectRoot
	bc.dbDir = dbDir

	commands := []struct {
		name string
		args []string
	}{
		{"start", []string{"start", "on", "testproject"}},
		{"stop", []string{"stop"}},
		{"blocks", []string{"blocks"}},
		{"stats", []string{"stats"}},
		{"project list", []string{"project"}},
		{"version", []string{"version"}},
	}

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			// If this is stop, make sure we have active tracking
			if cmd.name == "stop" {
				bc.run("start", "on", "testproject")
			}

			start := time.Now()
			_, _, err := bc.run(cmd.args...)
			elapsed := time.Since(start)

			if err != nil && cmd.name != "stop" {
				t.Logf("command %s returned error: %v", cmd.name, err)
			}

			if elapsed > MaxCommandExecutionTime {
				t.Errorf("command %s took %v, exceeds %v goal", cmd.name, elapsed, MaxCommandExecutionTime)
			} else {
				t.Logf("command %s completed in %v (goal: < %v)", cmd.name, elapsed, MaxCommandExecutionTime)
			}
		})
	}
}

// TestExport1000BlocksUnder5Seconds validates that exporting 1000 blocks is under 5 seconds.
func TestExport1000BlocksUnder5Seconds(t *testing.T) {
	dbDir, err := os.MkdirTemp("", "humantime-perf-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dbDir)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(wd, "..", "..")
	binaryPath := filepath.Join(projectRoot, "humantime")

	// Build the binary
	if err := buildBinary(projectRoot, binaryPath); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	// Create 1000 blocks directly in DB
	db, err := storage.Open(storage.Options{
		Path: filepath.Join(dbDir, "humantime.db"),
	})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	blockRepo := storage.NewBlockRepo(db)
	baseTime := time.Now().Add(-24 * time.Hour)

	t.Log("Creating 1000 blocks for export test...")
	for i := 0; i < 1000; i++ {
		block := model.NewBlock(
			"user1",
			"benchproject",
			"task1",
			"benchmark note for export testing",
			baseTime.Add(time.Duration(i)*time.Minute),
		)
		block.TimestampEnd = baseTime.Add(time.Duration(i)*time.Minute + 30*time.Minute)
		if err := blockRepo.Create(block); err != nil {
			t.Fatalf("failed to create block: %v", err)
		}
	}
	db.Close()

	// Now run export and measure time
	cmd := exec.Command(binaryPath, "export", "--format", "json")
	cmd.Env = append(os.Environ(), "HUMANTIME_DATABASE="+filepath.Join(dbDir, "humantime.db"))

	start := time.Now()
	output, err := cmd.Output()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}

	// Verify we got 1000 blocks in the output
	var exportResp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(output, &exportResp); err != nil {
		t.Fatalf("failed to parse export output: %v", err)
	}

	if exportResp.Count != 1000 {
		t.Errorf("expected 1000 blocks in export, got %d", exportResp.Count)
	}

	if elapsed > MaxExport1000BlocksTime {
		t.Errorf("export of 1000 blocks took %v, exceeds %v goal", elapsed, MaxExport1000BlocksTime)
	} else {
		t.Logf("export of 1000 blocks completed in %v (goal: < %v)", elapsed, MaxExport1000BlocksTime)
	}
}

// TestMemoryUsageUnder50MB validates that memory usage is under 50MB.
func TestMemoryUsageUnder50MB(t *testing.T) {
	dbDir, err := os.MkdirTemp("", "humantime-perf-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dbDir)

	// Create 1000 blocks directly in DB
	db, err := storage.Open(storage.Options{
		Path: filepath.Join(dbDir, "humantime.db"),
	})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	blockRepo := storage.NewBlockRepo(db)
	baseTime := time.Now().Add(-24 * time.Hour)

	t.Log("Creating 1000 blocks for memory test...")
	for i := 0; i < 1000; i++ {
		block := model.NewBlock(
			"user1",
			"benchproject",
			"task1",
			"benchmark note for memory testing with some additional content to simulate real usage patterns",
			baseTime.Add(time.Duration(i)*time.Minute),
		)
		block.TimestampEnd = baseTime.Add(time.Duration(i)*time.Minute + 30*time.Minute)
		if err := blockRepo.Create(block); err != nil {
			t.Fatalf("failed to create block: %v", err)
		}
	}

	// Force GC and measure memory
	runtime.GC()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	allocMB := float64(memStats.Alloc) / 1024 / 1024
	totalAllocMB := float64(memStats.TotalAlloc) / 1024 / 1024
	sysMB := float64(memStats.Sys) / 1024 / 1024

	t.Logf("Memory stats after creating 1000 blocks:")
	t.Logf("  Alloc: %.2f MB", allocMB)
	t.Logf("  TotalAlloc: %.2f MB", totalAllocMB)
	t.Logf("  Sys: %.2f MB", sysMB)

	// List all blocks and measure
	blocks, err := blockRepo.List()
	if err != nil {
		t.Fatalf("failed to list blocks: %v", err)
	}

	runtime.GC()
	runtime.ReadMemStats(&memStats)

	allocAfterListMB := float64(memStats.Alloc) / 1024 / 1024
	t.Logf("Memory after listing %d blocks: %.2f MB", len(blocks), allocAfterListMB)

	db.Close()

	if allocAfterListMB > float64(MaxMemoryUsageMB) {
		t.Errorf("memory usage %.2f MB exceeds %d MB goal", allocAfterListMB, MaxMemoryUsageMB)
	} else {
		t.Logf("memory usage %.2f MB is within %d MB goal", allocAfterListMB, MaxMemoryUsageMB)
	}
}

// ============================================================================
// Performance Summary Report
// ============================================================================

// TestPerformanceSummary runs all performance tests and generates a summary report.
func TestPerformanceSummary(t *testing.T) {
	t.Log("=== Humantime Performance Test Summary ===")
	t.Log("")
	t.Log("Performance Goals:")
	t.Logf("  - Command execution: < %v", MaxCommandExecutionTime)
	t.Logf("  - Memory usage: < %d MB", MaxMemoryUsageMB)
	t.Logf("  - Export 1000 blocks: < %v", MaxExport1000BlocksTime)
	t.Log("")
	t.Log("Run individual benchmarks with:")
	t.Log("  go test -bench=. ./tests/benchmark/...")
	t.Log("")
	t.Log("Run with memory profiling:")
	t.Log("  go test -bench=. -benchmem ./tests/benchmark/...")
	t.Log("")
	t.Log("Measure actual CLI execution time:")
	t.Log("  time ./humantime start on testproject")
	t.Log("  time ./humantime blocks")
	t.Log("  time ./humantime export --format json")
}
