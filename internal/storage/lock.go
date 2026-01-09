// Package storage provides the database layer for Humantime.
package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// LockFileName is the name of the lock file in the data directory.
	LockFileName = "humantime.lock"
)

var (
	// ErrLockAcquireFailed is returned when the lock cannot be acquired.
	ErrLockAcquireFailed = errors.New("failed to acquire database lock")
	// ErrLockAlreadyHeld is returned when another process holds the lock.
	ErrLockAlreadyHeld = errors.New("database is locked by another process")
)

// FileLock represents a file-based lock for preventing concurrent access.
type FileLock struct {
	path string
	file *os.File
}

// NewFileLock creates a new file lock at the specified directory.
func NewFileLock(dir string) *FileLock {
	return &FileLock{
		path: filepath.Join(dir, LockFileName),
	}
}

// Acquire attempts to acquire the lock.
// It returns an error if the lock is already held by another process.
func (l *FileLock) Acquire() error {
	// Check for stale lock first
	if err := l.cleanStaleLock(); err != nil {
		return err
	}

	// Create the lock file
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLockAcquireFailed, err)
	}

	// Try to acquire an exclusive lock (non-blocking)
	if err := flockAcquire(file); err != nil {
		file.Close()
		if errors.Is(err, ErrLockAlreadyHeld) {
			// Read the PID from the lock file to provide a better error message
			pid := l.readPID()
			if pid > 0 {
				return fmt.Errorf("%w: PID %d", ErrLockAlreadyHeld, pid)
			}
			return ErrLockAlreadyHeld
		}
		return err
	}

	// Write our PID to the lock file
	if err := file.Truncate(0); err != nil {
		flockRelease(file)
		file.Close()
		return fmt.Errorf("%w: %v", ErrLockAcquireFailed, err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		flockRelease(file)
		file.Close()
		return fmt.Errorf("%w: %v", ErrLockAcquireFailed, err)
	}
	if _, err := fmt.Fprintf(file, "%d", os.Getpid()); err != nil {
		flockRelease(file)
		file.Close()
		return fmt.Errorf("%w: %v", ErrLockAcquireFailed, err)
	}
	if err := file.Sync(); err != nil {
		flockRelease(file)
		file.Close()
		return fmt.Errorf("%w: %v", ErrLockAcquireFailed, err)
	}

	l.file = file
	return nil
}

// Release releases the lock.
func (l *FileLock) Release() error {
	if l.file == nil {
		return nil
	}

	// Unlock and close the file
	if err := flockRelease(l.file); err != nil {
		l.file.Close()
		return err
	}

	if err := l.file.Close(); err != nil {
		return err
	}

	// Remove the lock file
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return err
	}

	l.file = nil
	return nil
}

// cleanStaleLock checks if a lock file exists and removes it if the process is no longer running.
func (l *FileLock) cleanStaleLock() error {
	pid := l.readPID()
	if pid <= 0 {
		// No valid PID found, no stale lock to clean
		return nil
	}

	// Check if the process is still running
	if isProcessRunning(pid) {
		// Process is still running, lock is valid
		return nil
	}

	// Process is not running, remove the stale lock
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean stale lock: %v", err)
	}

	return nil
}

// readPID reads the PID from the lock file.
// Returns 0 if the file doesn't exist or doesn't contain a valid PID.
func (l *FileLock) readPID() int {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return 0
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0
	}

	return pid
}

// isProcessRunning is implemented in platform-specific files (lock_unix.go, lock_windows.go)

// LockError provides a user-friendly error message for lock failures.
type LockError struct {
	Err error
	PID int
}

func (e *LockError) Error() string {
	if e.PID > 0 {
		return fmt.Sprintf("cannot access database: another humantime instance (PID %d) is running", e.PID)
	}
	return fmt.Sprintf("cannot access database: %v", e.Err)
}

func (e *LockError) Unwrap() error {
	return e.Err
}

// NewLockError creates a new LockError with a helpful message.
func NewLockError(err error) *LockError {
	lockErr := &LockError{Err: err}

	// Try to extract PID from the error message
	if errors.Is(err, ErrLockAlreadyHeld) {
		// Try to parse PID from error message
		errStr := err.Error()
		if strings.Contains(errStr, "PID") {
			parts := strings.Split(errStr, "PID ")
			if len(parts) > 1 {
				if pid, parseErr := strconv.Atoi(strings.TrimSpace(parts[1])); parseErr == nil {
					lockErr.PID = pid
				}
			}
		}
	}

	return lockErr
}
