package daemon

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks daemon operational metrics.
type Metrics struct {
	// Counters
	notificationsSent   atomic.Int64
	notificationsFailed atomic.Int64
	remindersChecked    atomic.Int64
	errorsTotal         atomic.Int64

	// Gauges with mutex for complex types
	mu                 sync.RWMutex
	webhookLatencyMs   int64
	lastNotificationAt time.Time
	lastReminderCheck  time.Time
	lastError          string
	lastErrorAt        time.Time

	// Error breakdown
	errorsByCategory map[string]int64
}

// NewMetrics creates a new metrics tracker.
func NewMetrics() *Metrics {
	return &Metrics{
		errorsByCategory: make(map[string]int64),
	}
}

// MetricsSnapshot represents a point-in-time view of metrics.
type MetricsSnapshot struct {
	NotificationsSentTotal   int64             `json:"notifications_sent_total"`
	NotificationsFailedTotal int64             `json:"notifications_failed_total"`
	RemindersCheckedTotal    int64             `json:"reminders_checked_total"`
	ErrorsTotal              int64             `json:"errors_total"`
	WebhookLatencyMs         int64             `json:"webhook_latency_ms"`
	LastNotificationAt       *time.Time        `json:"last_notification_at,omitempty"`
	LastReminderCheck        *time.Time        `json:"last_reminder_check,omitempty"`
	LastError                string            `json:"last_error,omitempty"`
	LastErrorAt              *time.Time        `json:"last_error_at,omitempty"`
	ErrorsByCategory         map[string]int64  `json:"errors_by_category,omitempty"`
}

// Snapshot returns a copy of current metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap := MetricsSnapshot{
		NotificationsSentTotal:   m.notificationsSent.Load(),
		NotificationsFailedTotal: m.notificationsFailed.Load(),
		RemindersCheckedTotal:    m.remindersChecked.Load(),
		ErrorsTotal:              m.errorsTotal.Load(),
		WebhookLatencyMs:         m.webhookLatencyMs,
		LastError:                m.lastError,
		ErrorsByCategory:         make(map[string]int64, len(m.errorsByCategory)),
	}

	if !m.lastNotificationAt.IsZero() {
		snap.LastNotificationAt = &m.lastNotificationAt
	}
	if !m.lastReminderCheck.IsZero() {
		snap.LastReminderCheck = &m.lastReminderCheck
	}
	if !m.lastErrorAt.IsZero() {
		snap.LastErrorAt = &m.lastErrorAt
	}

	for k, v := range m.errorsByCategory {
		snap.ErrorsByCategory[k] = v
	}

	return snap
}

// JSON returns metrics as JSON.
func (m *Metrics) JSON() ([]byte, error) {
	return json.MarshalIndent(m.Snapshot(), "", "  ")
}

// RecordNotificationSent records a successful notification.
func (m *Metrics) RecordNotificationSent(latencyMs int64) {
	m.notificationsSent.Add(1)

	m.mu.Lock()
	m.webhookLatencyMs = latencyMs
	m.lastNotificationAt = time.Now()
	m.mu.Unlock()
}

// RecordNotificationFailed records a failed notification.
func (m *Metrics) RecordNotificationFailed(err error) {
	m.notificationsFailed.Add(1)
	m.RecordError("notification", err)
}

// RecordReminderCheck records a reminder check cycle.
func (m *Metrics) RecordReminderCheck() {
	m.remindersChecked.Add(1)

	m.mu.Lock()
	m.lastReminderCheck = time.Now()
	m.mu.Unlock()
}

// RecordError records an error with category.
func (m *Metrics) RecordError(category string, err error) {
	m.errorsTotal.Add(1)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastError = err.Error()
	m.lastErrorAt = time.Now()

	if category != "" {
		m.errorsByCategory[category]++
	}
}

// NotificationsSent returns the total notifications sent.
func (m *Metrics) NotificationsSent() int64 {
	return m.notificationsSent.Load()
}

// NotificationsFailed returns the total failed notifications.
func (m *Metrics) NotificationsFailed() int64 {
	return m.notificationsFailed.Load()
}

// RemindersChecked returns the total reminder checks.
func (m *Metrics) RemindersChecked() int64 {
	return m.remindersChecked.Load()
}

// ErrorsTotal returns the total errors.
func (m *Metrics) ErrorsTotal() int64 {
	return m.errorsTotal.Load()
}

// WebhookLatency returns the last webhook latency in ms.
func (m *Metrics) WebhookLatency() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.webhookLatencyMs
}

// Reset resets all metrics to zero.
func (m *Metrics) Reset() {
	m.notificationsSent.Store(0)
	m.notificationsFailed.Store(0)
	m.remindersChecked.Store(0)
	m.errorsTotal.Store(0)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.webhookLatencyMs = 0
	m.lastNotificationAt = time.Time{}
	m.lastReminderCheck = time.Time{}
	m.lastError = ""
	m.lastErrorAt = time.Time{}
	m.errorsByCategory = make(map[string]int64)
}

// GlobalMetrics is the default metrics instance.
var GlobalMetrics = NewMetrics()
