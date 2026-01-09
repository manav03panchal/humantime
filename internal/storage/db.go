// Package storage provides the database layer for Humantime.
package storage

import (
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
	db *badger.DB
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

	if opts.InMemory || opts.Path == "" {
		// In-memory mode for testing
		badgerOpts = badger.DefaultOptions("").WithInMemory(true)
	} else {
		// Ensure directory exists
		if err := os.MkdirAll(opts.Path, 0755); err != nil {
			return nil, err
		}
		badgerOpts = badger.DefaultOptions(opts.Path)
	}

	// Reduce logging noise
	badgerOpts = badgerOpts.WithLoggingLevel(badger.ERROR)

	db, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// Badger returns the underlying Badger database for advanced operations.
func (d *DB) Badger() *badger.DB {
	return d.db
}
