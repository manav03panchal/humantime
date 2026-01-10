package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/daemon"
)

// TestHealthCheckerBasic tests basic health check functionality.
func TestHealthCheckerBasic(t *testing.T) {
	checker := daemon.NewHealthChecker("test-version")

	status := checker.Check()

	if status == nil {
		t.Fatal("Check() should return non-nil status")
	}

	if status.Status != "healthy" {
		t.Errorf("Status should be 'healthy', got %q", status.Status)
	}
}

// TestHealthCheckerUptime tests uptime tracking.
func TestHealthCheckerUptime(t *testing.T) {
	checker := daemon.NewHealthChecker("test-version")

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	status := checker.Check()

	if status.UptimeSeconds < 0 {
		t.Error("UptimeSeconds should be non-negative")
	}
}

// TestHealthCheckerMemory tests memory reporting.
func TestHealthCheckerMemory(t *testing.T) {
	checker := daemon.NewHealthChecker("test-version")

	status := checker.Check()

	if status.MemoryMB < 0 {
		t.Error("MemoryMB should be non-negative")
	}
}

// TestHealthCheckerLastCheck tests last check timestamp.
func TestHealthCheckerLastCheck(t *testing.T) {
	checker := daemon.NewHealthChecker("test-version")

	before := time.Now()
	status := checker.Check()

	if status.LastCheck.Before(before) {
		t.Error("LastCheck should be after test start")
	}
}

// TestHealthStatusJSON tests JSON serialization of health status.
func TestHealthStatusJSON(t *testing.T) {
	status := &daemon.HealthStatus{
		Status:               "healthy",
		UptimeSeconds:        3600,
		MemoryMB:             25.5,
		PendingNotifications: 0,
		LastCheck:            time.Now(),
	}

	jsonBytes, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal health status: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Error("JSON output should not be empty")
	}

	// Unmarshal and verify
	var decoded daemon.HealthStatus
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal health status: %v", err)
	}

	if decoded.Status != "healthy" {
		t.Errorf("Decoded status should be 'healthy', got %q", decoded.Status)
	}
	if decoded.UptimeSeconds != 3600 {
		t.Errorf("Decoded uptime should be 3600, got %d", decoded.UptimeSeconds)
	}
}

// TestMetricsIntegration tests metrics integration with health check.
func TestMetricsIntegration(t *testing.T) {
	metrics := daemon.NewMetrics()

	// Record some activity
	metrics.RecordNotificationSent(100)
	metrics.RecordReminderCheck()

	snapshot := metrics.Snapshot()

	if snapshot.NotificationsSentTotal != 1 {
		t.Errorf("NotificationsSentTotal should be 1, got %d", snapshot.NotificationsSentTotal)
	}
	if snapshot.RemindersCheckedTotal != 1 {
		t.Errorf("RemindersCheckedTotal should be 1, got %d", snapshot.RemindersCheckedTotal)
	}
}

// TestHealthCheckerWithCustomCheck tests health checker with custom checks.
func TestHealthCheckerWithCustomCheck(t *testing.T) {
	checker := daemon.NewHealthChecker("test-version")

	// Add a passing check
	checker.AddCheck("test-check", func() error {
		return nil
	})

	status := checker.Check()

	if status.Status != "healthy" {
		t.Errorf("Status should be 'healthy', got %q", status.Status)
	}
}

// TestHealthCheckerPendingNotifications tests pending notification tracking.
func TestHealthCheckerPendingNotifications(t *testing.T) {
	checker := daemon.NewHealthChecker("test-version")

	checker.SetPendingNotifications(5)
	status := checker.Check()

	if status.PendingNotifications != 5 {
		t.Errorf("PendingNotifications should be 5, got %d", status.PendingNotifications)
	}
}

// TestHealthCheckerJSON tests JSON output.
func TestHealthCheckerJSON(t *testing.T) {
	checker := daemon.NewHealthChecker("1.0.0")

	jsonBytes, err := checker.JSON()
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Error("JSON output should not be empty")
	}

	// Verify it's valid JSON
	var status daemon.HealthStatus
	if err := json.Unmarshal(jsonBytes, &status); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if status.Version != "1.0.0" {
		t.Errorf("Version should be '1.0.0', got %q", status.Version)
	}
}

// TestHealthCheckerDetailedCheck tests detailed health check.
func TestHealthCheckerDetailedCheck(t *testing.T) {
	checker := daemon.NewHealthChecker("test-version")

	detailed := checker.DetailedCheck()

	if detailed.Status != "healthy" {
		t.Errorf("Status should be 'healthy', got %q", detailed.Status)
	}

	if detailed.MemoryDetails.AllocMB < 0 {
		t.Error("AllocMB should be non-negative")
	}
}

// TestHealthCheckerIsHealthy tests IsHealthy helper.
func TestHealthCheckerIsHealthy(t *testing.T) {
	checker := daemon.NewHealthChecker("test-version")

	if !checker.IsHealthy() {
		t.Error("Checker should be healthy by default")
	}
}

// TestHealthCheckerUptime tests Uptime method.
func TestHealthCheckerUptimeMethod(t *testing.T) {
	checker := daemon.NewHealthChecker("test-version")

	time.Sleep(50 * time.Millisecond)

	uptime := checker.Uptime()
	if uptime < 50*time.Millisecond {
		t.Errorf("Uptime should be at least 50ms, got %v", uptime)
	}
}
