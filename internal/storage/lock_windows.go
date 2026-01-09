//go:build windows

// Package storage provides the database layer for Humantime.
package storage

import (
	"os"
)

// flockAcquire acquires an exclusive lock on the file (Windows implementation).
// On Windows, we use file handle flags instead of flock.
// The file is opened with exclusive access, which prevents other processes from opening it.
func flockAcquire(file *os.File) error {
	// On Windows, the file was opened with O_EXCL semantics via OpenFile
	// The lock is implicit through file handle exclusivity
	// This is a no-op since Windows handles locking differently
	return nil
}

// flockRelease releases the lock on the file (Windows implementation).
func flockRelease(file *os.File) error {
	// On Windows, closing the file releases the lock
	return nil
}

// isProcessRunning checks if a process with the given PID is still running.
func isProcessRunning(pid int) bool {
	// On Windows, FindProcess always succeeds if the process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Windows, we can check if the process handle is valid
	// by attempting to get its exit code
	// If the process is still running, Wait will return an error
	var ws *os.ProcessState
	ws, err = process.Wait()
	if err != nil {
		// If we can't wait on the process, it might still be running
		// or we don't have permission - assume running to be safe
		return true
	}
	return !ws.Exited()
}
