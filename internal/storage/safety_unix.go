//go:build !windows

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

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
