// Package storage provides the database layer for Humantime.
package storage

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/manav03panchal/humantime/internal/runtime"
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

// WrapError checks if an error is a disk full error and wraps it appropriately.
// This should be called on errors returned from Badger write operations.
func (d *DB) WrapError(err error, op string) error {
	if err == nil {
		return nil
	}
	return runtime.WrapDiskFullError(err, op, d.path)
}
