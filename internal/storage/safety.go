package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/manav03panchal/humantime/internal/errors"
)

const (
	// MinFreeSpace is the minimum free space required for write operations (10MB).
	MinFreeSpace = 10 * 1024 * 1024
	// MinFreeSpaceWarning is the threshold for warning about low disk space (50MB).
	MinFreeSpaceWarning = 50 * 1024 * 1024
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
// Returns an error if free space is below MinFreeSpace.
func CheckDiskSpace(path string) error {
	info, err := GetDiskSpace(path)
	if err != nil {
		// If we can't check disk space, proceed but log warning
		return nil
	}

	if info.FreeBytes < MinFreeSpace {
		return errors.NewSystemError(
			fmt.Sprintf("insufficient disk space: %d MB free, need at least %d MB",
				info.FreeBytes/(1024*1024),
				MinFreeSpace/(1024*1024)),
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

	if info.FreeBytes < MinFreeSpaceWarning {
		return fmt.Sprintf("Warning: Low disk space (%d MB free)", info.FreeBytes/(1024*1024))
	}

	return ""
}

// GetDiskSpace returns disk space information for the given path.
func GetDiskSpace(path string) (*DiskSpaceInfo, error) {
	// Ensure path exists or use parent directory
	for {
		if _, err := os.Stat(path); err == nil {
			break
		}
		parent := filepath.Dir(path)
		if parent == path {
			// Reached root, use as-is
			break
		}
		path = parent
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("failed to get disk space: %w", err)
	}

	info := &DiskSpaceInfo{
		Path:       path,
		TotalBytes: stat.Blocks * uint64(stat.Bsize),
		FreeBytes:  stat.Bavail * uint64(stat.Bsize),
	}
	info.UsedBytes = info.TotalBytes - info.FreeBytes

	return info, nil
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

// isDiskFullError checks if an error indicates disk full condition.
func isDiskFullError(err error) bool {
	if err == nil {
		return false
	}

	// Check for ENOSPC
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			return errno == syscall.ENOSPC
		}
	}

	// Check for syscall.Errno directly
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.ENOSPC
	}

	return false
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
