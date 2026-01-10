package daemon

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// HealthChecker Tests
// =============================================================================

func TestNewHealthChecker(t *testing.T) {
	checker := NewHealthChecker("1.0.0")
	assert.NotNil(t, checker)
	assert.Equal(t, "1.0.0", checker.version)
}

func TestHealthCheckerCheck(t *testing.T) {
	checker := NewHealthChecker("1.0.0")

	status := checker.Check()
	assert.NotNil(t, status)
	assert.Equal(t, "healthy", status.Status)
	assert.Equal(t, "1.0.0", status.Version)
	assert.GreaterOrEqual(t, status.Goroutines, 1)
	assert.GreaterOrEqual(t, status.MemoryMB, 0.0)
}

func TestHealthCheckerSetPendingNotifications(t *testing.T) {
	checker := NewHealthChecker("1.0.0")

	checker.SetPendingNotifications(5)
	status := checker.Check()
	assert.Equal(t, 5, status.PendingNotifications)
}

func TestHealthCheckerAddRemoveCheck(t *testing.T) {
	checker := NewHealthChecker("1.0.0")

	// Add a failing check
	checker.AddCheck("test", func() error {
		return errors.New("test error")
	})

	status := checker.Check()
	assert.Equal(t, "unhealthy", status.Status)

	// Remove the check
	checker.RemoveCheck("test")

	status = checker.Check()
	assert.Equal(t, "healthy", status.Status)
}

func TestHealthCheckerIsHealthy(t *testing.T) {
	checker := NewHealthChecker("1.0.0")

	assert.True(t, checker.IsHealthy())

	checker.AddCheck("fail", func() error {
		return errors.New("error")
	})

	assert.False(t, checker.IsHealthy())
}

func TestHealthCheckerUptime(t *testing.T) {
	checker := NewHealthChecker("1.0.0")

	time.Sleep(10 * time.Millisecond)

	uptime := checker.Uptime()
	assert.GreaterOrEqual(t, uptime, 10*time.Millisecond)
}

func TestHealthCheckerJSON(t *testing.T) {
	checker := NewHealthChecker("1.0.0")

	data, err := checker.JSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "healthy")
	assert.Contains(t, string(data), "1.0.0")
}

func TestHealthCheckerDetailedCheck(t *testing.T) {
	checker := NewHealthChecker("1.0.0")

	// Add a check
	checker.AddCheck("db", func() error { return nil })

	details := checker.DetailedCheck()
	assert.NotNil(t, details)
	assert.Equal(t, "healthy", details.Status)
	assert.GreaterOrEqual(t, details.MemoryDetails.AllocMB, 0.0)
	assert.GreaterOrEqual(t, details.MemoryDetails.SysMB, 0.0)
	assert.Len(t, details.Checks, 1)
	assert.Equal(t, "db", details.Checks[0].Name)
	assert.True(t, details.Checks[0].Healthy)
}

func TestHealthCheckerDetailedCheckWithFailure(t *testing.T) {
	checker := NewHealthChecker("1.0.0")

	checker.AddCheck("failing", func() error {
		return errors.New("check failed")
	})

	details := checker.DetailedCheck()
	assert.Len(t, details.Checks, 1)
	assert.False(t, details.Checks[0].Healthy)
	assert.Equal(t, "check failed", details.Checks[0].Error)
}

// =============================================================================
// Metrics Tests
// =============================================================================

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	assert.NotNil(t, m)
	assert.Equal(t, int64(0), m.NotificationsSent())
}

func TestMetricsRecordNotificationSent(t *testing.T) {
	m := NewMetrics()

	m.RecordNotificationSent(100)

	assert.Equal(t, int64(1), m.NotificationsSent())
	assert.Equal(t, int64(100), m.WebhookLatency())
}

func TestMetricsRecordNotificationFailed(t *testing.T) {
	m := NewMetrics()

	m.RecordNotificationFailed(errors.New("network error"))

	assert.Equal(t, int64(1), m.NotificationsFailed())
	assert.Equal(t, int64(1), m.ErrorsTotal())
}

func TestMetricsRecordReminderCheck(t *testing.T) {
	m := NewMetrics()

	m.RecordReminderCheck()
	m.RecordReminderCheck()

	assert.Equal(t, int64(2), m.RemindersChecked())
}

func TestMetricsRecordError(t *testing.T) {
	m := NewMetrics()

	m.RecordError("webhook", errors.New("timeout"))
	m.RecordError("webhook", errors.New("timeout"))
	m.RecordError("db", errors.New("connection failed"))

	assert.Equal(t, int64(3), m.ErrorsTotal())

	snap := m.Snapshot()
	assert.Equal(t, int64(2), snap.ErrorsByCategory["webhook"])
	assert.Equal(t, int64(1), snap.ErrorsByCategory["db"])
}

func TestMetricsSnapshot(t *testing.T) {
	m := NewMetrics()

	m.RecordNotificationSent(50)
	m.RecordNotificationFailed(errors.New("error"))
	m.RecordReminderCheck()

	snap := m.Snapshot()
	assert.Equal(t, int64(1), snap.NotificationsSentTotal)
	assert.Equal(t, int64(1), snap.NotificationsFailedTotal)
	assert.Equal(t, int64(1), snap.RemindersCheckedTotal)
	assert.Equal(t, int64(50), snap.WebhookLatencyMs)
	assert.NotNil(t, snap.LastNotificationAt)
	assert.NotNil(t, snap.LastReminderCheck)
}

func TestMetricsJSON(t *testing.T) {
	m := NewMetrics()

	m.RecordNotificationSent(100)

	data, err := m.JSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "notifications_sent_total")
}

func TestMetricsReset(t *testing.T) {
	m := NewMetrics()

	m.RecordNotificationSent(100)
	m.RecordNotificationFailed(errors.New("error"))
	m.RecordReminderCheck()
	m.RecordError("test", errors.New("test"))

	m.Reset()

	assert.Equal(t, int64(0), m.NotificationsSent())
	assert.Equal(t, int64(0), m.NotificationsFailed())
	assert.Equal(t, int64(0), m.RemindersChecked())
	assert.Equal(t, int64(0), m.ErrorsTotal())
	assert.Equal(t, int64(0), m.WebhookLatency())

	snap := m.Snapshot()
	assert.Nil(t, snap.LastNotificationAt)
	assert.Empty(t, snap.LastError)
}

func TestGlobalMetrics(t *testing.T) {
	assert.NotNil(t, GlobalMetrics)

	// Reset before and after to avoid affecting other tests
	GlobalMetrics.Reset()
	defer GlobalMetrics.Reset()

	GlobalMetrics.RecordNotificationSent(10)
	assert.Equal(t, int64(1), GlobalMetrics.NotificationsSent())
}

// =============================================================================
// PIDFile Tests
// =============================================================================

func TestNewPIDFile(t *testing.T) {
	pf := NewPIDFile()
	assert.NotNil(t, pf)
	assert.NotEmpty(t, pf.Path())
}

func TestGetPIDFilePath(t *testing.T) {
	path := GetPIDFilePath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, PIDFileName)
}

func TestPIDFileWriteAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	pf := &PIDFile{path: tmpDir + "/test.pid"}

	// Write a PID
	err := pf.WritePID(12345)
	assert.NoError(t, err)

	// Read it back
	pid, err := pf.Read()
	assert.NoError(t, err)
	assert.Equal(t, 12345, pid)
}

func TestPIDFileReadNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	pf := &PIDFile{path: tmpDir + "/nonexistent.pid"}

	_, err := pf.Read()
	assert.Error(t, err)
	assert.Equal(t, ErrNotRunning, err)
}

func TestPIDFileReadInvalidPID(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := tmpDir + "/invalid.pid"

	// Write invalid content
	err := os.WriteFile(pidPath, []byte("not-a-number"), 0644)
	assert.NoError(t, err)

	pf := &PIDFile{path: pidPath}
	_, err = pf.Read()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PID")
}

func TestPIDFileRemove(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := tmpDir + "/remove.pid"

	pf := &PIDFile{path: pidPath}

	// Write a PID
	err := pf.WritePID(12345)
	assert.NoError(t, err)
	assert.True(t, pf.Exists())

	// Remove it
	err = pf.Remove()
	assert.NoError(t, err)
	assert.False(t, pf.Exists())
}

func TestPIDFileRemoveNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	pf := &PIDFile{path: tmpDir + "/nonexistent.pid"}

	// Should not error when removing non-existent file
	err := pf.Remove()
	assert.NoError(t, err)
}

func TestPIDFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := tmpDir + "/exists.pid"
	pf := &PIDFile{path: pidPath}

	assert.False(t, pf.Exists())

	err := pf.WritePID(12345)
	assert.NoError(t, err)

	assert.True(t, pf.Exists())
}

func TestPIDFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	expectedPath := tmpDir + "/test.pid"
	pf := &PIDFile{path: expectedPath}

	assert.Equal(t, expectedPath, pf.Path())
}

func TestPIDFileIsRunning(t *testing.T) {
	tmpDir := t.TempDir()
	pf := &PIDFile{path: tmpDir + "/running.pid"}

	// No PID file - not running
	assert.False(t, pf.IsRunning())

	// Write current process PID - should be running
	err := pf.WritePID(os.Getpid())
	assert.NoError(t, err)
	assert.True(t, pf.IsRunning())

	// Write non-existent PID - not running
	err = pf.WritePID(999999999)
	assert.NoError(t, err)
	assert.False(t, pf.IsRunning())
}

func TestPIDFileGetRunningPID(t *testing.T) {
	tmpDir := t.TempDir()
	pf := &PIDFile{path: tmpDir + "/getpid.pid"}

	// No PID file
	assert.Equal(t, 0, pf.GetRunningPID())

	// Write current PID
	currentPID := os.Getpid()
	err := pf.WritePID(currentPID)
	assert.NoError(t, err)

	assert.Equal(t, currentPID, pf.GetRunningPID())

	// Write non-running PID
	err = pf.WritePID(999999999)
	assert.NoError(t, err)
	assert.Equal(t, 0, pf.GetRunningPID())
}

func TestIsProcessRunning(t *testing.T) {
	// Current process should be running
	assert.True(t, IsProcessRunning(os.Getpid()))

	// Invalid PID
	assert.False(t, IsProcessRunning(0))
	assert.False(t, IsProcessRunning(-1))

	// Very high PID unlikely to exist
	assert.False(t, IsProcessRunning(999999999))
}

// =============================================================================
// Logger Tests
// =============================================================================

func TestLoggerWithWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer: &buf,
	}

	logger.Log("test message")
	output := buf.String()

	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "[")
	assert.Contains(t, output, "]")
}

func TestLoggerInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{writer: &buf}

	logger.Info("info message")
	assert.Contains(t, buf.String(), "INFO")
	assert.Contains(t, buf.String(), "info message")
}

func TestLoggerWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{writer: &buf}

	logger.Warn("warning message")
	assert.Contains(t, buf.String(), "WARN")
	assert.Contains(t, buf.String(), "warning message")
}

func TestLoggerError(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{writer: &buf}

	logger.Error("error message")
	assert.Contains(t, buf.String(), "ERROR")
	assert.Contains(t, buf.String(), "error message")
}

func TestLoggerDebug(t *testing.T) {
	t.Run("debug_disabled", func(t *testing.T) {
		var buf bytes.Buffer
		logger := &Logger{writer: &buf, debug: false}

		logger.Debug("debug message")
		assert.Empty(t, buf.String())
	})

	t.Run("debug_enabled", func(t *testing.T) {
		var buf bytes.Buffer
		logger := &Logger{writer: &buf, debug: true}

		logger.Debug("debug message")
		assert.Contains(t, buf.String(), "DEBUG")
		assert.Contains(t, buf.String(), "debug message")
	})
}

func TestLoggerSetDebug(t *testing.T) {
	logger := &Logger{}
	assert.False(t, logger.debug)

	logger.SetDebug(true)
	assert.True(t, logger.debug)

	logger.SetDebug(false)
	assert.False(t, logger.debug)
}

func TestLoggerLogFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{writer: &buf}

	logger.Log("value: %d, name: %s", 42, "test")
	assert.Contains(t, buf.String(), "value: 42, name: test")
}

func TestLoggerClose(t *testing.T) {
	// Test closing with nil file
	logger := &Logger{}
	err := logger.Close()
	assert.NoError(t, err)
}

func TestGetLogDir(t *testing.T) {
	dir := GetLogDir()
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, AppName)
}

func TestGetLogPath(t *testing.T) {
	path := GetLogPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, AppName)
	assert.Contains(t, path, ".log")
}

// =============================================================================
// formatUptime Tests
// =============================================================================

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"one_minute", 60 * time.Second, "1m"},
		{"minutes", 5 * time.Minute, "5m"},
		{"one_hour", 60 * time.Minute, "1h"},
		{"hours_with_minutes", 1*time.Hour + 30*time.Minute, "1h 30m"},
		{"hours_only", 2 * time.Hour, "2h"},
		{"one_day", 24 * time.Hour, "1d"},
		{"days_with_hours", 24*time.Hour + 12*time.Hour, "1d 12h"},
		{"days_only", 48 * time.Hour, "2d"},
		{"complex", 3*24*time.Hour + 5*time.Hour, "3d 5h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUptime(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Status Tests
// =============================================================================

func TestStatusStruct(t *testing.T) {
	now := time.Now()
	status := Status{
		Running:   true,
		PID:       12345,
		StartedAt: now,
		Uptime:    "5m",
	}

	assert.True(t, status.Running)
	assert.Equal(t, 12345, status.PID)
	assert.Equal(t, now, status.StartedAt)
	assert.Equal(t, "5m", status.Uptime)
}

// =============================================================================
// DaemonState Tests
// =============================================================================

func TestDaemonStateStruct(t *testing.T) {
	now := time.Now()
	state := DaemonState{
		StartedAt: now,
	}

	assert.Equal(t, now, state.StartedAt)
}

// =============================================================================
// Error Constants Tests
// =============================================================================

func TestErrNotRunning(t *testing.T) {
	assert.NotNil(t, ErrNotRunning)
	assert.Contains(t, ErrNotRunning.Error(), "not running")
}

func TestErrAlreadyRunning(t *testing.T) {
	assert.NotNil(t, ErrAlreadyRunning)
	assert.Contains(t, ErrAlreadyRunning.Error(), "already running")
}

// =============================================================================
// ServiceManager Tests
// =============================================================================

func TestNewServiceManager(t *testing.T) {
	sm, err := NewServiceManager()
	assert.NoError(t, err)
	assert.NotNil(t, sm)
	assert.NotEmpty(t, sm.executablePath)
}

func TestServiceManagerSetDebug(t *testing.T) {
	sm := &ServiceManager{}
	assert.False(t, sm.debug)

	sm.SetDebug(true)
	assert.True(t, sm.debug)
}

func TestServiceManagerGetLaunchdPath(t *testing.T) {
	sm := &ServiceManager{}
	path := sm.getLaunchdPath()
	assert.Contains(t, path, "LaunchAgents")
	assert.Contains(t, path, "com.humantime.daemon.plist")
}

func TestServiceManagerGetSystemdPath(t *testing.T) {
	sm := &ServiceManager{}
	path := sm.getSystemdPath()
	assert.Contains(t, path, "systemd")
	assert.Contains(t, path, "humantime.service")
}

// =============================================================================
// Concurrent Logger Access Test
// =============================================================================

func TestLoggerConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{writer: &buf}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				logger.Log("goroutine %d iteration %d", n, j)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have logged 1000 messages without deadlock
	output := buf.String()
	assert.NotEmpty(t, output)
}

// =============================================================================
// Daemon Tests
// =============================================================================

func TestNewDaemon(t *testing.T) {
	// Create an in-memory database for testing
	db, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		t.Skip("skipping test - couldn't create in-memory database")
	}
	defer db.Close()

	daemon := NewDaemon(db)
	assert.NotNil(t, daemon)
	assert.NotNil(t, daemon.pidFile)
	assert.NotNil(t, daemon.db)
	assert.NotNil(t, daemon.reminderRepo)
	assert.NotNil(t, daemon.webhookRepo)
	assert.NotNil(t, daemon.blockRepo)
	assert.NotNil(t, daemon.activeBlockRepo)
	assert.NotNil(t, daemon.goalRepo)
	assert.NotNil(t, daemon.notifyConfigRepo)
}

func TestDaemonSetDebug(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		t.Skip("skipping test - couldn't create in-memory database")
	}
	defer db.Close()

	daemon := NewDaemon(db)
	assert.False(t, daemon.debug)

	daemon.SetDebug(true)
	assert.True(t, daemon.debug)

	daemon.SetDebug(false)
	assert.False(t, daemon.debug)
}

func TestDaemonGetStatus(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		t.Skip("skipping test - couldn't create in-memory database")
	}
	defer db.Close()

	daemon := NewDaemon(db)

	status := daemon.GetStatus()
	assert.NotNil(t, status)
	// By default, daemon should not be running
	assert.False(t, status.Running)
}

func TestDaemonIsRunning(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		t.Skip("skipping test - couldn't create in-memory database")
	}
	defer db.Close()

	daemon := NewDaemon(db)

	// By default, daemon should not be running
	assert.False(t, daemon.IsRunning())
}

func TestGetStatePath(t *testing.T) {
	path := getStatePath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "daemon")
}

func TestDaemonWriteReadState(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		t.Skip("skipping test - couldn't create in-memory database")
	}
	defer db.Close()

	daemon := NewDaemon(db)

	// Write state
	now := time.Now()
	err = daemon.writeState(&DaemonState{StartedAt: now})
	assert.NoError(t, err)

	// Read state back
	state, err := daemon.readState()
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.WithinDuration(t, now, state.StartedAt, time.Second)

	// Clean up
	daemon.removeState()
}

func TestDaemonReadStateNotExist(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		t.Skip("skipping test - couldn't create in-memory database")
	}
	defer db.Close()

	daemon := NewDaemon(db)

	// Ensure state file doesn't exist
	daemon.removeState()

	state, err := daemon.readState()
	assert.Error(t, err)
	assert.Nil(t, state)
}

func TestDaemonRemoveState(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	if err != nil {
		t.Skip("skipping test - couldn't create in-memory database")
	}
	defer db.Close()

	daemon := NewDaemon(db)

	// Write state first
	daemon.writeState(&DaemonState{StartedAt: time.Now()})

	// Remove state (doesn't return error)
	daemon.removeState()

	// Verify it's gone
	_, err = daemon.readState()
	assert.Error(t, err)
}

// =============================================================================
// SignalHandler Tests
// =============================================================================

func TestNewSignalHandler(t *testing.T) {
	handler := NewSignalHandler()
	assert.NotNil(t, handler)
}

func TestSignalHandlerSetup(t *testing.T) {
	handler := NewSignalHandler()
	handler.Setup()
	defer handler.Cleanup()

	// Should not panic and channel should be created
	assert.NotNil(t, handler)
}
