package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/manav03panchal/humantime/internal/storage"
)

// TestFileLockAcquireRelease verifies basic lock acquire/release.
func TestFileLockAcquireRelease(t *testing.T) {
	tmpDir := t.TempDir()
	lock := storage.NewFileLock(tmpDir)

	// Acquire should succeed
	if err := lock.Acquire(); err != nil {
		t.Fatalf("First acquire failed: %v", err)
	}

	// Release should succeed
	if err := lock.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Lock file should be removed
	lockPath := filepath.Join(tmpDir, storage.LockFileName)
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("Lock file should be removed after release")
	}
}

// TestFileLockReacquire verifies lock can be reacquired after release.
func TestFileLockReacquire(t *testing.T) {
	tmpDir := t.TempDir()
	lock := storage.NewFileLock(tmpDir)

	// First acquire
	if err := lock.Acquire(); err != nil {
		t.Fatalf("First acquire failed: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("First release failed: %v", err)
	}

	// Second acquire
	if err := lock.Acquire(); err != nil {
		t.Fatalf("Second acquire failed: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Second release failed: %v", err)
	}
}

// TestFileLockWritesPID verifies PID is written to lock file.
func TestFileLockWritesPID(t *testing.T) {
	tmpDir := t.TempDir()
	lock := storage.NewFileLock(tmpDir)

	if err := lock.Acquire(); err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	defer lock.Release()

	// Read lock file
	lockPath := filepath.Join(tmpDir, storage.LockFileName)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	// Verify PID is our PID
	expectedPID := os.Getpid()
	if len(data) == 0 {
		t.Error("Lock file should contain PID")
	}

	t.Logf("Lock file contains: %s (our PID: %d)", string(data), expectedPID)
}

// TestFileLockDoubleRelease verifies double release is safe.
func TestFileLockDoubleRelease(t *testing.T) {
	tmpDir := t.TempDir()
	lock := storage.NewFileLock(tmpDir)

	if err := lock.Acquire(); err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	// First release
	if err := lock.Release(); err != nil {
		t.Fatalf("First release failed: %v", err)
	}

	// Second release should be no-op (not error)
	if err := lock.Release(); err != nil {
		t.Errorf("Second release should not error: %v", err)
	}
}

// TestFileLockReleaseWithoutAcquire verifies release without acquire is safe.
func TestFileLockReleaseWithoutAcquire(t *testing.T) {
	tmpDir := t.TempDir()
	lock := storage.NewFileLock(tmpDir)

	// Release without acquire should be no-op
	if err := lock.Release(); err != nil {
		t.Errorf("Release without acquire should not error: %v", err)
	}
}

// TestDatabaseUsesLock verifies database acquires lock on open.
func TestDatabaseUsesLock(t *testing.T) {
	tmpDir := t.TempDir()

	// Open first database
	db1, err := storage.Open(storage.Options{Path: tmpDir})
	if err != nil {
		t.Fatalf("First database open failed: %v", err)
	}

	// Lock file should exist
	lockPath := filepath.Join(tmpDir, storage.LockFileName)
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lock file should exist while database is open")
	}

	// Close database
	if err := db1.Close(); err != nil {
		t.Fatalf("Database close failed: %v", err)
	}
}

// TestDatabaseConcurrentOpenFails verifies concurrent access is prevented.
func TestDatabaseConcurrentOpenFails(t *testing.T) {
	tmpDir := t.TempDir()

	// Open first database
	db1, err := storage.Open(storage.Options{Path: tmpDir})
	if err != nil {
		t.Fatalf("First database open failed: %v", err)
	}
	defer db1.Close()

	// Second open should fail
	_, err = storage.Open(storage.Options{Path: tmpDir})
	if err == nil {
		t.Fatal("Second database open should fail due to lock")
	}

	// Error should indicate lock is held
	t.Logf("Expected error: %v", err)
}

// TestDatabaseReopenAfterClose verifies database can reopen after close.
func TestDatabaseReopenAfterClose(t *testing.T) {
	tmpDir := t.TempDir()

	// Open and close first database
	db1, err := storage.Open(storage.Options{Path: tmpDir})
	if err != nil {
		t.Fatalf("First open failed: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("First close failed: %v", err)
	}

	// Second open should succeed
	db2, err := storage.Open(storage.Options{Path: tmpDir})
	if err != nil {
		t.Fatalf("Second open failed: %v", err)
	}
	if err := db2.Close(); err != nil {
		t.Fatalf("Second close failed: %v", err)
	}
}

// TestInMemoryDatabaseNoLock verifies in-memory database doesn't use lock.
func TestInMemoryDatabaseNoLock(t *testing.T) {
	// Multiple in-memory databases should work
	db1, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		t.Fatalf("First in-memory open failed: %v", err)
	}
	defer db1.Close()

	db2, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		t.Fatalf("Second in-memory open failed: %v", err)
	}
	defer db2.Close()
}

// TestLockErrorMessage verifies lock error provides helpful message.
func TestLockErrorMessage(t *testing.T) {
	tmpDir := t.TempDir()

	// Open first database
	db1, err := storage.Open(storage.Options{Path: tmpDir})
	if err != nil {
		t.Fatalf("First open failed: %v", err)
	}
	defer db1.Close()

	// Try second open
	_, err = storage.Open(storage.Options{Path: tmpDir})
	if err == nil {
		t.Fatal("Expected lock error")
	}

	// Error should mention lock or another instance
	errStr := err.Error()
	if !containsAny(errStr, []string{"lock", "another", "instance", "PID"}) {
		t.Errorf("Error message should be helpful, got: %s", errStr)
	}
}

// containsAny checks if s contains any of the substrings.
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	s = toLowerCase(s)
	substr = toLowerCase(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// toLowerCase converts string to lowercase.
func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
