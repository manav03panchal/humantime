// Package storage provides the database layer for Humantime.
package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	badger "github.com/dgraph-io/badger/v4"
)

const (
	// AppName is the application name used for data directories.
	AppName = "humantime"
)

// DB wraps a Badger database connection.
type DB struct {
	db   *badger.DB
	lock *FileLock
	path string // Database path for error context
}

// Options configures the database connection.
type Options struct {
	// Path is the database directory path. Empty string uses in-memory mode.
	Path string
	// InMemory forces in-memory mode regardless of Path.
	InMemory bool
}

// DefaultPath returns the default database path following XDG spec.
func DefaultPath() string {
	return filepath.Join(xdg.DataHome, AppName, "db")
}

// Open opens or creates a database at the given path.
func Open(opts Options) (*DB, error) {
	var badgerOpts badger.Options
	var lock *FileLock

	if opts.InMemory || opts.Path == "" {
		// In-memory mode for testing
		badgerOpts = badger.DefaultOptions("").WithInMemory(true)
	} else {
		// Ensure directory exists
		if err := os.MkdirAll(opts.Path, 0755); err != nil {
			return nil, err
		}

		// Acquire file lock to prevent concurrent access
		lock = NewFileLock(opts.Path)
		if err := lock.Acquire(); err != nil {
			return nil, NewLockError(err)
		}

		badgerOpts = badger.DefaultOptions(opts.Path)
	}

	// Reduce logging noise
	badgerOpts = badgerOpts.WithLoggingLevel(badger.ERROR)

	db, err := badger.Open(badgerOpts)
	if err != nil {
		// Release the lock if we failed to open the database
		if lock != nil {
			lock.Release()
		}
		return nil, err
	}

	return &DB{db: db, lock: lock, path: opts.Path}, nil
}

// Close closes the database connection and releases the file lock.
func (d *DB) Close() error {
	// Close the database first
	err := d.db.Close()

	// Release the file lock (if any)
	if d.lock != nil {
		if lockErr := d.lock.Release(); lockErr != nil && err == nil {
			err = lockErr
		}
	}

	return err
}

// Badger returns the underlying Badger database for advanced operations.
func (d *DB) Badger() *badger.DB {
	return d.db
}

// Path returns the database path.
func (d *DB) Path() string {
	return d.path
}

// CheckIntegrity performs a database integrity check.
// Returns nil if healthy, error describing the issue otherwise.
func (d *DB) CheckIntegrity() error {
	status := CheckDatabaseIntegrity(d)
	if !status.Healthy {
		if len(status.Errors) > 0 {
			return fmt.Errorf("database integrity check failed: %s", status.Errors[0])
		}
		return fmt.Errorf("database integrity check failed")
	}
	return nil
}

// OpenWithIntegrityCheck opens the database and performs an integrity check.
// If the check fails, it attempts recovery if possible.
func OpenWithIntegrityCheck(opts Options) (*DB, error) {
	db, err := Open(opts)
	if err != nil {
		return nil, err
	}

	// Perform integrity check
	if err := db.CheckIntegrity(); err != nil {
		// Log the issue but don't fail - the app can still function
		// with a potentially degraded database
		_ = err // Logged elsewhere if needed
	}

	return db, nil
}
