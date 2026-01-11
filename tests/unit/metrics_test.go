package unit

import (
	"errors"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/daemon"
)

// TestMetricsRecordNotificationSent tests notification sent recording.
func TestMetricsRecordNotificationSent(t *testing.T) {
	m := daemon.NewMetrics()

	m.RecordNotificationSent(100)
	m.RecordNotificationSent(200)
	m.RecordNotificationSent(150)

	snapshot := m.Snapshot()

	if snapshot.NotificationsSentTotal != 3 {
		t.Errorf("NotificationsSentTotal = %d, want 3", snapshot.NotificationsSentTotal)
	}
}

// TestMetricsRecordNotificationFailed tests failed notification recording.
func TestMetricsRecordNotificationFailed(t *testing.T) {
	m := daemon.NewMetrics()

	m.RecordNotificationFailed(errors.New("error 1"))
	m.RecordNotificationFailed(errors.New("error 2"))

	snapshot := m.Snapshot()

	if snapshot.NotificationsFailedTotal != 2 {
		t.Errorf("NotificationsFailedTotal = %d, want 2", snapshot.NotificationsFailedTotal)
	}
}

// TestMetricsRecordReminderCheck tests reminder check recording.
func TestMetricsRecordReminderCheck(t *testing.T) {
	m := daemon.NewMetrics()

	m.RecordReminderCheck()
	m.RecordReminderCheck()

	snapshot := m.Snapshot()

	if snapshot.RemindersCheckedTotal != 2 {
		t.Errorf("RemindersCheckedTotal = %d, want 2", snapshot.RemindersCheckedTotal)
	}
}

// TestMetricsRecordError tests error recording.
func TestMetricsRecordError(t *testing.T) {
	m := daemon.NewMetrics()

	m.RecordError("network", errors.New("error 1"))
	m.RecordError("database", errors.New("error 2"))
	m.RecordError("network", errors.New("error 3"))

	snapshot := m.Snapshot()

	if snapshot.ErrorsTotal != 3 {
		t.Errorf("ErrorsTotal = %d, want 3", snapshot.ErrorsTotal)
	}

	// Check category breakdown
	if snapshot.ErrorsByCategory["network"] != 2 {
		t.Errorf("ErrorsByCategory[network] = %d, want 2", snapshot.ErrorsByCategory["network"])
	}
	if snapshot.ErrorsByCategory["database"] != 1 {
		t.Errorf("ErrorsByCategory[database] = %d, want 1", snapshot.ErrorsByCategory["database"])
	}
}

// TestMetricsInitialState tests initial metrics state.
func TestMetricsInitialState(t *testing.T) {
	m := daemon.NewMetrics()
	snapshot := m.Snapshot()

	if snapshot.NotificationsSentTotal != 0 {
		t.Error("Initial NotificationsSentTotal should be 0")
	}
	if snapshot.NotificationsFailedTotal != 0 {
		t.Error("Initial NotificationsFailedTotal should be 0")
	}
	if snapshot.RemindersCheckedTotal != 0 {
		t.Error("Initial RemindersCheckedTotal should be 0")
	}
	if snapshot.ErrorsTotal != 0 {
		t.Error("Initial ErrorsTotal should be 0")
	}
}

// TestMetricsWebhookLatency tests latency tracking.
func TestMetricsWebhookLatency(t *testing.T) {
	m := daemon.NewMetrics()

	m.RecordNotificationSent(100)
	m.RecordNotificationSent(200)
	m.RecordNotificationSent(300)

	snapshot := m.Snapshot()

	// Last latency should be 300
	if snapshot.WebhookLatencyMs != 300 {
		t.Errorf("WebhookLatencyMs = %d, want 300", snapshot.WebhookLatencyMs)
	}
}

// TestMetricsConcurrency tests concurrent metric updates.
func TestMetricsConcurrency(t *testing.T) {
	m := daemon.NewMetrics()
	done := make(chan bool)

	// Concurrent writers
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.RecordNotificationSent(10)
				m.RecordNotificationFailed(errors.New("test"))
				m.RecordError("test", errors.New("test"))
				m.RecordReminderCheck()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	snapshot := m.Snapshot()

	// Should have 10 * 100 = 1000 of each
	if snapshot.NotificationsSentTotal != 1000 {
		t.Errorf("Concurrent NotificationsSentTotal = %d, want 1000", snapshot.NotificationsSentTotal)
	}
	if snapshot.NotificationsFailedTotal != 1000 {
		t.Errorf("Concurrent NotificationsFailedTotal = %d, want 1000", snapshot.NotificationsFailedTotal)
	}
	// ErrorsTotal is notifications_failed + explicit errors = 1000 + 1000 = 2000
	if snapshot.ErrorsTotal != 2000 {
		t.Errorf("Concurrent ErrorsTotal = %d, want 2000", snapshot.ErrorsTotal)
	}
	if snapshot.RemindersCheckedTotal != 1000 {
		t.Errorf("Concurrent RemindersCheckedTotal = %d, want 1000", snapshot.RemindersCheckedTotal)
	}
}

// TestMetricsReset tests metrics reset functionality.
func TestMetricsReset(t *testing.T) {
	m := daemon.NewMetrics()

	m.RecordNotificationSent(100)
	m.RecordNotificationFailed(errors.New("test"))
	m.RecordError("test", errors.New("test"))

	m.Reset()

	snapshot := m.Snapshot()

	if snapshot.NotificationsSentTotal != 0 {
		t.Error("NotificationsSentTotal should be 0 after reset")
	}
	if snapshot.NotificationsFailedTotal != 0 {
		t.Error("NotificationsFailedTotal should be 0 after reset")
	}
	if snapshot.ErrorsTotal != 0 {
		t.Error("ErrorsTotal should be 0 after reset")
	}
}

// TestMetricsLastNotification tests last notification timestamp.
func TestMetricsLastNotification(t *testing.T) {
	m := daemon.NewMetrics()

	before := time.Now()
	m.RecordNotificationSent(100)

	snapshot := m.Snapshot()

	if snapshot.LastNotificationAt == nil {
		t.Error("LastNotificationAt should be set after sending")
	}
	if snapshot.LastNotificationAt.Before(before) {
		t.Error("LastNotificationAt should be after test start")
	}
}

// TestMetricsJSON tests JSON output.
func TestMetricsJSON(t *testing.T) {
	m := daemon.NewMetrics()
	m.RecordNotificationSent(100)

	jsonBytes, err := m.JSON()
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Error("JSON output should not be empty")
	}
}

// TestMetricsDirectAccessors tests direct accessor methods.
func TestMetricsDirectAccessors(t *testing.T) {
	m := daemon.NewMetrics()

	m.RecordNotificationSent(100)
	m.RecordNotificationSent(200)

	if m.NotificationsSent() != 2 {
		t.Errorf("NotificationsSent() = %d, want 2", m.NotificationsSent())
	}

	if m.WebhookLatency() != 200 {
		t.Errorf("WebhookLatency() = %d, want 200", m.WebhookLatency())
	}
}
