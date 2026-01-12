//go:build windows

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
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

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("failed to convert path: %w", err)
	}

	ret, _, err := getDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("failed to get disk space: %w", err)
	}

	info := &DiskSpaceInfo{
		Path:       path,
		TotalBytes: totalBytes,
		FreeBytes:  freeBytesAvailable,
	}
	info.UsedBytes = info.TotalBytes - info.FreeBytes

	return info, nil
}

// isDiskFullError checks if an error indicates disk full condition.
func isDiskFullError(err error) bool {
	if err == nil {
		return false
	}

	// Windows error code for disk full: ERROR_DISK_FULL = 112
	const ERROR_DISK_FULL = syscall.Errno(112)

	// Check for PathError
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			return errno == ERROR_DISK_FULL
		}
	}

	// Check for syscall.Errno directly
	if errno, ok := err.(syscall.Errno); ok {
		return errno == ERROR_DISK_FULL
	}

	return false
}
