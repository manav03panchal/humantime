package daemon

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adrg/xdg"
)

// Logger provides file-based logging for the daemon.
type Logger struct {
	file   *os.File
	writer io.Writer
	mu     sync.Mutex
	debug  bool
}

// NewLogger creates a new daemon logger.
func NewLogger() (*Logger, error) {
	logPath := GetLogPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file for appending
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{
		file:   file,
		writer: file,
	}, nil
}

// SetDebug enables debug logging.
func (l *Logger) SetDebug(debug bool) {
	l.debug = debug
}

// Close closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Log writes a log message with timestamp.
func (l *Logger) Log(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "[%s] %s\n", timestamp, msg)
}

// Info logs an info message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.Log("INFO  "+format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.Log("WARN  "+format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.Log("ERROR "+format, args...)
}

// Debug logs a debug message (only if debug mode is enabled).
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debug {
		l.Log("DEBUG "+format, args...)
	}
}

// GetLogDir returns the directory containing log files.
func GetLogDir() string {
	return filepath.Join(xdg.StateHome, AppName)
}

// RotateLog rotates the log file if it exceeds maxSize bytes.
func (l *Logger) RotateLog(maxSize int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return nil
	}

	// Check current file size
	info, err := l.file.Stat()
	if err != nil {
		return err
	}

	if info.Size() < maxSize {
		return nil // No rotation needed
	}

	// Close current file
	l.file.Close()

	logPath := GetLogPath()
	backupPath := logPath + ".old"

	// Remove old backup if exists
	os.Remove(backupPath)

	// Rename current to backup
	if err := os.Rename(logPath, backupPath); err != nil {
		return err
	}

	// Create new log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.writer = file

	return nil
}
