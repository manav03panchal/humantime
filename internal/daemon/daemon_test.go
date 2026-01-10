package daemon

import (
	"errors"
	"testing"
	"time"

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
