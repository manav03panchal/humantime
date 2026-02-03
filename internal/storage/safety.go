package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/manav03panchal/humantime/internal/config"
	"github.com/manav03panchal/humantime/internal/errors"
)

// DiskSpaceInfo contains information about available disk space.
type DiskSpaceInfo struct {
	Path       string
	TotalBytes uint64
	FreeBytes  uint64
	UsedBytes  uint64
}

// FreePercent returns the percentage of free space.
func (d *DiskSpaceInfo) FreePercent() float64 {
	if d.TotalBytes == 0 {
		return 0
	}
	return float64(d.FreeBytes) / float64(d.TotalBytes) * 100
}

// CheckDiskSpace checks if there's enough disk space at the given path.
// Returns an error if free space is below MinFreeSpace from config.
func CheckDiskSpace(path string) error {
	info, err := GetDiskSpace(path)
	if err != nil {
		// If we can't check disk space, proceed but log warning
		return nil
	}

	minFreeSpace := config.Global.Storage.MinFreeSpace
	if info.FreeBytes < minFreeSpace {
		return errors.NewSystemError(
			fmt.Sprintf("insufficient disk space: %d MB free, need at least %d MB",
				info.FreeBytes/(1024*1024),
				minFreeSpace/(1024*1024)),
			errors.ErrDiskFull,
		)
	}

	return nil
}

// CheckDiskSpaceWarning checks disk space and returns a warning message if low.
// Returns empty string if disk space is adequate.
func CheckDiskSpaceWarning(path string) string {
	info, err := GetDiskSpace(path)
	if err != nil {
		return ""
	}

	minFreeSpaceWarning := config.Global.Storage.MinFreeSpaceWarning
	if info.FreeBytes < minFreeSpaceWarning {
		return fmt.Sprintf("Warning: Low disk space (%d MB free)", info.FreeBytes/(1024*1024))
	}

	return ""
}

// SafeWrite performs a write operation with disk space check.
// It checks disk space before the write and wraps disk-full errors appropriately.
func SafeWrite(path string, data []byte, perm os.FileMode) error {
	// Check disk space before write
	if err := CheckDiskSpace(filepath.Dir(path)); err != nil {
		return err
	}

	// Use atomic write pattern: write to temp file, then rename
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".humantime-*.tmp")
	if err != nil {
		if isDiskFullError(err) {
			return errors.NewSystemErrorWithOp("create temp file", "disk full", errors.ErrDiskFull)
		}
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on failure
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Write data
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		if isDiskFullError(err) {
			return errors.NewSystemErrorWithOp("write", "disk full", errors.ErrDiskFull)
		}
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Sync to ensure data is on disk
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		if isDiskFullError(err) {
			return errors.NewSystemErrorWithOp("sync", "disk full", errors.ErrDiskFull)
		}
		return fmt.Errorf("failed to sync data: %w", err)
	}

	// Close before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}

// EnsureDirectory creates a directory with safe permissions if it doesn't exist.
func EnsureDirectory(path string) error {
	if err := CheckDiskSpace(filepath.Dir(path)); err != nil {
		return err
	}

	if err := os.MkdirAll(path, 0700); err != nil {
		if isDiskFullError(err) {
			return errors.NewSystemErrorWithOp("mkdir", "disk full", errors.ErrDiskFull)
		}
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}
