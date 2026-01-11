package storage

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/manav03panchal/humantime/internal/errors"
	"github.com/manav03panchal/humantime/internal/logging"
)

// RecoveryStatus represents the result of a database health check.
type RecoveryStatus struct {
	Healthy     bool      `json:"healthy"`
	Corrupted   bool      `json:"corrupted"`
	LastCheck   time.Time `json:"last_check"`
	ErrorCount  int       `json:"error_count"`
	Errors      []string  `json:"errors,omitempty"`
	Recoverable bool      `json:"recoverable"`
	BackupPath  string    `json:"backup_path,omitempty"`
}

// CheckDatabaseIntegrity performs a basic integrity check on the database.
// Returns a RecoveryStatus with details about the database health.
func CheckDatabaseIntegrity(db *DB) *RecoveryStatus {
	status := &RecoveryStatus{
		LastCheck: time.Now(),
		Healthy:   true,
	}

	if db == nil || db.db == nil {
		status.Healthy = false
		status.Corrupted = true
		status.Errors = append(status.Errors, "database not initialized")
		return status
	}

	// Try to iterate through a sample of keys to detect corruption
	err := db.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		count := 0
		for it.Rewind(); it.Valid() && count < 100; it.Next() {
			item := it.Item()
			// Try to read the value to detect corruption
			if err := item.Value(func(val []byte) error {
				return nil
			}); err != nil {
				status.Errors = append(status.Errors, fmt.Sprintf("corrupted value at key: %s", item.Key()))
				status.ErrorCount++
			}
			count++
		}
		return nil
	})

	if err != nil {
		status.Healthy = false
		status.Corrupted = true
		status.Errors = append(status.Errors, fmt.Sprintf("iteration error: %v", err))
		status.ErrorCount++
	}

	if status.ErrorCount > 0 {
		status.Healthy = false
		status.Corrupted = true
		// Check if we can recover by re-opening
		status.Recoverable = status.ErrorCount < 10
	}

	return status
}

// CreateBackup creates a backup of the database directory.
// Returns the path to the backup or an error.
func CreateBackup(dbPath string) (string, error) {
	if dbPath == "" {
		return "", fmt.Errorf("database path is empty")
	}

	// Create backup directory
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create timestamped backup name
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("db-backup-%s", timestamp))

	// Copy the database directory
	if err := copyDir(dbPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to copy database: %w", err)
	}

	logging.Info("database backup created", logging.KeyOperation, "backup", "path", backupPath)
	return backupPath, nil
}

// copyDir copies a directory recursively.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, srcInfo.Mode())
}

// ExportSalvageableData attempts to export readable data from a corrupted database.
// Returns the number of records exported and the export file path.
func ExportSalvageableData(db *DB, exportPath string) (int, error) {
	if db == nil || db.db == nil {
		return 0, fmt.Errorf("database not available")
	}

	// Create export file
	file, err := os.Create(exportPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	exported := 0
	exportData := make(map[string]interface{})

	err = db.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			err := item.Value(func(val []byte) error {
				// Try to decode as JSON
				var jsonVal interface{}
				if json.Unmarshal(val, &jsonVal) == nil {
					exportData[key] = jsonVal
				} else {
					// Store as raw string if not JSON
					exportData[key] = string(val)
				}
				exported++
				return nil
			})

			if err != nil {
				// Skip corrupted entries
				logging.Warn("skipping corrupted entry", "key", key, logging.KeyError, err)
				continue
			}
		}
		return nil
	})

	if err != nil {
		return exported, fmt.Errorf("export iteration error: %w", err)
	}

	// Write exported data
	if err := encoder.Encode(exportData); err != nil {
		return exported, fmt.Errorf("failed to write export data: %w", err)
	}

	logging.Info("salvageable data exported", logging.KeyCount, exported, "path", exportPath)
	return exported, nil
}

// AttemptRecovery attempts to recover a corrupted database.
// Returns nil if recovery was successful, or an error describing the failure.
func AttemptRecovery(dbPath string) error {
	// First, create a backup
	backupPath, err := CreateBackup(dbPath)
	if err != nil {
		logging.Warn("failed to create backup before recovery", logging.KeyError, err)
		// Continue anyway - recovery might still work
	}

	// Try to open with recovery options
	opts := badger.DefaultOptions(dbPath).
		WithLoggingLevel(badger.ERROR).
		WithNumVersionsToKeep(1) // Keep only latest version

	db, err := badger.Open(opts)
	if err != nil {
		return errors.NewSystemError("failed to open database for recovery", err)
	}
	defer db.Close()

	// Run garbage collection to clean up
	for {
		err := db.RunValueLogGC(0.5)
		if err != nil {
			break // GC complete when no more garbage
		}
	}

	logging.Info("database recovery attempted",
		"backup_path", backupPath,
		logging.KeyStatus, "completed")

	return nil
}

// IsDatabaseCorrupted checks if the given error indicates database corruption.
func IsDatabaseCorrupted(err error) bool {
	if err == nil {
		return false
	}

	// Check for our sentinel error
	if stderrors.Is(err, errors.ErrDatabaseCorrupted) {
		return true
	}

	// Check error message for corruption patterns
	errStr := err.Error()
	corruptionPatterns := []string{
		"checksum mismatch",
		"corrupt",
		"invalid",
		"unexpected eof",
		"bad magic",
		"truncated",
	}

	for _, pattern := range corruptionPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

// equalFold checks if two strings are equal (case-insensitive).
func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
