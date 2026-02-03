package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultRuntimeConfig(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	// Test daemon defaults
	if cfg.Daemon.StartupWait != 500*time.Millisecond {
		t.Errorf("expected Daemon.StartupWait = 500ms, got %v", cfg.Daemon.StartupWait)
	}
	if cfg.Daemon.KillTimeout != 5*time.Second {
		t.Errorf("expected Daemon.KillTimeout = 5s, got %v", cfg.Daemon.KillTimeout)
	}

	// Test HTTP defaults
	if cfg.HTTP.Timeout != 30*time.Second {
		t.Errorf("expected HTTP.Timeout = 30s, got %v", cfg.HTTP.Timeout)
	}
	if cfg.HTTP.MaxRetries != 3 {
		t.Errorf("expected HTTP.MaxRetries = 3, got %d", cfg.HTTP.MaxRetries)
	}
	if len(cfg.HTTP.RetryDelays) != 3 {
		t.Errorf("expected HTTP.RetryDelays length = 3, got %d", len(cfg.HTTP.RetryDelays))
	}

	// Test retry queue defaults
	if cfg.RetryQueue.CheckInterval != 30*time.Second {
		t.Errorf("expected RetryQueue.CheckInterval = 30s, got %v", cfg.RetryQueue.CheckInterval)
	}
	if len(cfg.RetryQueue.BackoffSchedule) != 5 {
		t.Errorf("expected RetryQueue.BackoffSchedule length = 5, got %d", len(cfg.RetryQueue.BackoffSchedule))
	}

	// Test storage defaults
	if cfg.Storage.MinFreeSpace != 10*1024*1024 {
		t.Errorf("expected Storage.MinFreeSpace = 10MB, got %d", cfg.Storage.MinFreeSpace)
	}
	if cfg.Storage.MinFreeSpaceWarning != 50*1024*1024 {
		t.Errorf("expected Storage.MinFreeSpaceWarning = 50MB, got %d", cfg.Storage.MinFreeSpaceWarning)
	}

	// Test scheduler defaults
	if cfg.Scheduler.IdleNotificationCooldown != 30*time.Minute {
		t.Errorf("expected Scheduler.IdleNotificationCooldown = 30m, got %v", cfg.Scheduler.IdleNotificationCooldown)
	}
	if cfg.Scheduler.SleepThreshold != 1*time.Hour {
		t.Errorf("expected Scheduler.SleepThreshold = 1h, got %v", cfg.Scheduler.SleepThreshold)
	}
}

func TestGlobalConfigExists(t *testing.T) {
	if Global == nil {
		t.Fatal("Global config should not be nil")
	}
}

func TestConfigReset(t *testing.T) {
	// Modify global config
	Global.HTTP.Timeout = 1 * time.Second

	// Reset
	Global.Reset()

	// Verify it's back to defaults
	if Global.HTTP.Timeout != 30*time.Second {
		t.Errorf("expected HTTP.Timeout = 30s after reset, got %v", Global.HTTP.Timeout)
	}
}

func TestConfigLoadFromEnv(t *testing.T) {
	// Save and restore global state
	originalCfg := *Global
	defer func() {
		*Global = originalCfg
	}()

	// Set environment variables
	os.Setenv("HUMANTIME_HTTP_TIMEOUT", "60s")
	os.Setenv("HUMANTIME_HTTP_MAX_RETRIES", "5")
	os.Setenv("HUMANTIME_DAEMON_KILL_TIMEOUT", "10s")
	defer func() {
		os.Unsetenv("HUMANTIME_HTTP_TIMEOUT")
		os.Unsetenv("HUMANTIME_HTTP_MAX_RETRIES")
		os.Unsetenv("HUMANTIME_DAEMON_KILL_TIMEOUT")
	}()

	// Create new config with env overrides
	cfg := DefaultRuntimeConfig()
	cfg.loadFromEnv()

	// Verify env overrides
	if cfg.HTTP.Timeout != 60*time.Second {
		t.Errorf("expected HTTP.Timeout = 60s from env, got %v", cfg.HTTP.Timeout)
	}
	if cfg.HTTP.MaxRetries != 5 {
		t.Errorf("expected HTTP.MaxRetries = 5 from env, got %d", cfg.HTTP.MaxRetries)
	}
	if cfg.Daemon.KillTimeout != 10*time.Second {
		t.Errorf("expected Daemon.KillTimeout = 10s from env, got %v", cfg.Daemon.KillTimeout)
	}
}

func TestConfigLoadFromEnvInvalidValues(t *testing.T) {
	// Save and restore global state
	originalCfg := *Global
	defer func() {
		*Global = originalCfg
	}()

	// Set invalid environment variables
	os.Setenv("HUMANTIME_HTTP_TIMEOUT", "invalid")
	os.Setenv("HUMANTIME_HTTP_MAX_RETRIES", "not-a-number")
	defer func() {
		os.Unsetenv("HUMANTIME_HTTP_TIMEOUT")
		os.Unsetenv("HUMANTIME_HTTP_MAX_RETRIES")
	}()

	// Create new config with invalid env - should keep defaults
	cfg := DefaultRuntimeConfig()
	cfg.loadFromEnv()

	// Verify defaults are kept when env values are invalid
	if cfg.HTTP.Timeout != 30*time.Second {
		t.Errorf("expected HTTP.Timeout = 30s (default), got %v", cfg.HTTP.Timeout)
	}
	if cfg.HTTP.MaxRetries != 3 {
		t.Errorf("expected HTTP.MaxRetries = 3 (default), got %d", cfg.HTTP.MaxRetries)
	}
}

func TestBackoffScheduleValues(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	expected := []time.Duration{
		5 * time.Second,
		30 * time.Second,
		2 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
	}

	for i, expectedDuration := range expected {
		if cfg.RetryQueue.BackoffSchedule[i] != expectedDuration {
			t.Errorf("BackoffSchedule[%d]: expected %v, got %v",
				i, expectedDuration, cfg.RetryQueue.BackoffSchedule[i])
		}
	}
}

func TestHTTPRetryDelayValues(t *testing.T) {
	cfg := DefaultRuntimeConfig()

	expected := []time.Duration{
		0,
		5 * time.Second,
		30 * time.Second,
	}

	for i, expectedDuration := range expected {
		if cfg.HTTP.RetryDelays[i] != expectedDuration {
			t.Errorf("RetryDelays[%d]: expected %v, got %v",
				i, expectedDuration, cfg.HTTP.RetryDelays[i])
		}
	}
}
