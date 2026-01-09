//go:build !windows

// Package storage provides the database layer for Humantime.
package storage

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// flockAcquire acquires an exclusive lock on the file (Unix implementation).
func flockAcquire(file *os.File) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return ErrLockAlreadyHeld
		}
		return fmt.Errorf("%w: %v", ErrLockAcquireFailed, err)
	}
	return nil
}

// flockRelease releases the lock on the file (Unix implementation).
func flockRelease(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

// isProcessRunning checks if a process with the given PID is still running.
func isProcessRunning(pid int) bool {
	// On Unix systems, sending signal 0 to a process checks if it exists
	// without actually sending a signal
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to actually send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
