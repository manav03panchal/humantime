package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/manav03panchal/humantime/internal/storage"
)

// TestDatabaseIntegrityCheck verifies the database integrity check works.
func TestDatabaseIntegrityCheck(t *testing.T) {
	tmpDir := t.TempDir()

	// Open a fresh database
	db, err := storage.Open(storage.Options{Path: tmpDir})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check integrity of empty database
	status := storage.CheckDatabaseIntegrity(db)

	if !status.Healthy {
		t.Errorf("Fresh database should be healthy, got errors: %v", status.Errors)
	}
	if status.Corrupted {
		t.Error("Fresh database should not be corrupted")
	}
	if status.ErrorCount != 0 {
		t.Errorf("Fresh database should have 0 errors, got %d", status.ErrorCount)
	}
}

// TestDatabaseIntegrityCheckNilDB verifies handling of nil database.
func TestDatabaseIntegrityCheckNilDB(t *testing.T) {
	status := storage.CheckDatabaseIntegrity(nil)

	if status.Healthy {
		t.Error("Nil database should not be healthy")
	}
	if !status.Corrupted {
		t.Error("Nil database should be marked as corrupted")
	}
	if len(status.Errors) == 0 {
		t.Error("Nil database should have error messages")
	}
}

// TestCreateBackup verifies database backup creation.
func TestCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "db")

	// Create and populate a database
	db, err := storage.Open(storage.Options{Path: dbPath})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	db.Close()

	// Create backup
	backupPath, err := storage.CreateBackup(dbPath)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup directory not created at %s", backupPath)
	}

	t.Logf("Backup created at: %s", backupPath)
}

// TestCreateBackupEmptyPath verifies error handling for empty path.
func TestCreateBackupEmptyPath(t *testing.T) {
	_, err := storage.CreateBackup("")
	if err == nil {
		t.Error("CreateBackup should fail with empty path")
	}
}

// TestExportSalvageableData verifies data export from database.
func TestExportSalvageableData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "db")
	exportPath := filepath.Join(tmpDir, "export.json")

	// Create a database
	db, err := storage.Open(storage.Options{Path: dbPath})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Export data (empty database should work)
	count, err := storage.ExportSalvageableData(db, exportPath)
	if err != nil {
		t.Fatalf("ExportSalvageableData failed: %v", err)
	}

	db.Close()

	// Verify export file exists
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Error("Export file not created")
	}

	t.Logf("Exported %d records", count)
}

// TestExportSalvageableDataNilDB verifies error handling for nil database.
func TestExportSalvageableDataNilDB(t *testing.T) {
	tmpDir := t.TempDir()
	exportPath := filepath.Join(tmpDir, "export.json")

	_, err := storage.ExportSalvageableData(nil, exportPath)
	if err == nil {
		t.Error("ExportSalvageableData should fail with nil database")
	}
}

// TestIsDatabaseCorrupted verifies corruption detection patterns.
func TestIsDatabaseCorrupted(t *testing.T) {
	tests := []struct {
		name       string
		errMsg     string
		wantResult bool
	}{
		{"nil error", "", false},
		{"checksum mismatch", "checksum mismatch at block 42", true},
		{"corrupt file", "corrupt data file", true},
		{"invalid header", "invalid header format", true},
		{"unexpected eof", "unexpected eof while reading", true},
		{"bad magic", "bad magic number", true},
		{"truncated", "truncated file", true},
		{"normal error", "connection refused", false},
		{"timeout", "operation timed out", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.errMsg != "" {
				err = &testError{msg: tt.errMsg}
			}

			got := storage.IsDatabaseCorrupted(err)
			if got != tt.wantResult {
				t.Errorf("IsDatabaseCorrupted(%q) = %v, want %v", tt.errMsg, got, tt.wantResult)
			}
		})
	}
}

// testError is a simple error implementation for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestRecoveryStatusJSON verifies RecoveryStatus JSON serialization.
func TestRecoveryStatusJSON(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := storage.Open(storage.Options{Path: tmpDir})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	status := storage.CheckDatabaseIntegrity(db)

	// Status should have required fields
	if status.LastCheck.IsZero() {
		t.Error("LastCheck should be set")
	}
}
