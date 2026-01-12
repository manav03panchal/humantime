// Package daemon provides background process management for Humantime.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/adrg/xdg"
)

const (
	// AppName is the application name used for runtime directories.
	AppName = "humantime"
	// PIDFileName is the PID file name.
	PIDFileName = "humantime.pid"
)

// PIDFile manages the daemon PID file.
type PIDFile struct {
	path string
}

// NewPIDFile creates a new PID file manager.
func NewPIDFile() *PIDFile {
	return &PIDFile{
		path: GetPIDFilePath(),
	}
}

// GetPIDFilePath returns the path to the PID file.
func GetPIDFilePath() string {
	// Use XDG state directory for PID file (persists across reboots but is user-specific)
	// This is more reliable than runtime dir which may not exist on macOS
	return filepath.Join(xdg.StateHome, AppName, PIDFileName)
}

// Write writes the current process PID to the file.
func (p *PIDFile) Write() error {
	return p.WritePID(os.Getpid())
}

// WritePID writes a specific PID to the file.
func (p *PIDFile) WritePID(pid int) error {
	// Ensure directory exists
	dir := filepath.Dir(p.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create PID directory: %w", err)
	}

	// Write PID
	data := []byte(strconv.Itoa(pid))
	if err := os.WriteFile(p.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// Read reads the PID from the file.
func (p *PIDFile) Read() (int, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrNotRunning
		}
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// Remove removes the PID file.
func (p *PIDFile) Remove() error {
	if err := os.Remove(p.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

// Exists checks if the PID file exists.
func (p *PIDFile) Exists() bool {
	_, err := os.Stat(p.path)
	return err == nil
}

// IsRunning checks if the daemon is currently running.
func (p *PIDFile) IsRunning() bool {
	pid, err := p.Read()
	if err != nil {
		return false
	}
	return IsProcessRunning(pid)
}

// GetRunningPID returns the PID if the daemon is running, or 0 if not.
func (p *PIDFile) GetRunningPID() int {
	pid, err := p.Read()
	if err != nil {
		return 0
	}
	if !IsProcessRunning(pid) {
		return 0
	}
	return pid
}

// Path returns the PID file path.
func (p *PIDFile) Path() string {
	return p.path
}

// IsProcessRunning checks if a process with the given PID is running.
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0 to check
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Errors
var (
	ErrNotRunning     = fmt.Errorf("daemon is not running")
	ErrAlreadyRunning = fmt.Errorf("daemon is already running")
)
