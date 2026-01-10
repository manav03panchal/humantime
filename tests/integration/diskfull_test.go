package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/manav03panchal/humantime/internal/errors"
	"github.com/manav03panchal/humantime/internal/storage"
)

// TestDiskSpaceCheck verifies that disk space checking works correctly.
func TestDiskSpaceCheck(t *testing.T) {
	// Get disk space for temp directory
	tmpDir := os.TempDir()
	info, err := storage.GetDiskSpace(tmpDir)
	if err != nil {
		t.Fatalf("GetDiskSpace failed: %v", err)
	}

	// Verify we got valid info
	if info.TotalBytes == 0 {
		t.Error("TotalBytes should not be zero")
	}
	if info.FreeBytes == 0 {
		t.Error("FreeBytes should not be zero (unless disk is actually full)")
	}
	if info.Path == "" {
		t.Error("Path should not be empty")
	}

	// Check that free percent is reasonable
	freePercent := info.FreePercent()
	if freePercent < 0 || freePercent > 100 {
		t.Errorf("FreePercent %f is out of range", freePercent)
	}

	t.Logf("Disk space: %.2f%% free (%d MB / %d MB)",
		freePercent,
		info.FreeBytes/(1024*1024),
		info.TotalBytes/(1024*1024))
}

// TestCheckDiskSpaceWarning verifies low disk space warnings.
func TestCheckDiskSpaceWarning(t *testing.T) {
	tmpDir := os.TempDir()
	warning := storage.CheckDiskSpaceWarning(tmpDir)

	// We can't guarantee low disk space, but the function should not panic
	t.Logf("Disk space warning: %q", warning)
}

// TestSafeWriteCreatesFile verifies SafeWrite creates files correctly.
func TestSafeWriteCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("hello world")

	err := storage.SafeWrite(testFile, testData, 0600)
	if err != nil {
		t.Fatalf("SafeWrite failed: %v", err)
	}

	// Verify file exists and has correct content
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("File content mismatch: got %q, want %q", data, testData)
	}

	// Verify permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("File permissions: got %o, want 0600", info.Mode().Perm())
	}
}

// TestSafeWriteAtomic verifies SafeWrite is atomic (no partial writes).
func TestSafeWriteAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "atomic.txt")

	// Write initial content
	initial := []byte("initial content")
	if err := storage.SafeWrite(testFile, initial, 0600); err != nil {
		t.Fatalf("Initial SafeWrite failed: %v", err)
	}

	// Write new content (should replace atomically)
	newContent := []byte("new content that is longer")
	if err := storage.SafeWrite(testFile, newContent, 0600); err != nil {
		t.Fatalf("Second SafeWrite failed: %v", err)
	}

	// Verify new content
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != string(newContent) {
		t.Errorf("File content: got %q, want %q", data, newContent)
	}
}

// TestSafeWriteOverwritesExisting verifies SafeWrite correctly overwrites.
func TestSafeWriteOverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "overwrite.txt")

	// Create initial file with different content
	if err := os.WriteFile(testFile, []byte("old"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Overwrite with SafeWrite
	newContent := []byte("new")
	if err := storage.SafeWrite(testFile, newContent, 0600); err != nil {
		t.Fatalf("SafeWrite failed: %v", err)
	}

	// Verify new content
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != "new" {
		t.Errorf("File content: got %q, want %q", data, "new")
	}
}

// TestEnsureDirectory verifies directory creation with proper permissions.
func TestEnsureDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "subdir", "nested")

	err := storage.EnsureDirectory(newDir)
	if err != nil {
		t.Fatalf("EnsureDirectory failed: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("Directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}

	// Verify permissions (0700)
	if info.Mode().Perm() != 0700 {
		t.Errorf("Directory permissions: got %o, want 0700", info.Mode().Perm())
	}
}

// TestDiskFullErrorDetection verifies disk full error classification.
func TestDiskFullErrorDetection(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantFull bool
	}{
		{"nil error", nil, false},
		{"disk full sentinel", errors.ErrDiskFull, true},
		{"wrapped disk full", errors.Wrap(errors.ErrDiskFull, "context"), true},
		{"random error", errors.NewUserError("test", "suggestion"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errors.Is(tt.err, errors.ErrDiskFull)
			if got != tt.wantFull {
				t.Errorf("IsDiskFull = %v, want %v", got, tt.wantFull)
			}
		})
	}
}
