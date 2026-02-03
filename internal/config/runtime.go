// Package config provides centralized configuration for Humantime runtime values.
package config

import (
	"os"
	"strconv"
	"time"
)

// RuntimeConfig holds all runtime configuration values that were previously
// hardcoded as magic values throughout the codebase.
type RuntimeConfig struct {
	// Daemon configuration
	Daemon DaemonConfig

	// HTTP client configuration
	HTTP HTTPConfig

	// Retry queue configuration
	RetryQueue RetryQueueConfig

	// Storage configuration
	Storage StorageConfig

	// Scheduler configuration
	Scheduler SchedulerConfig
}

// DaemonConfig holds daemon-related configuration.
type DaemonConfig struct {
	// StartupWait is the time to wait for the daemon to start before checking status.
	// Default: 500ms
	StartupWait time.Duration

	// KillTimeout is the timeout for graceful shutdown before force kill.
	// Default: 5s
	KillTimeout time.Duration
}

// HTTPConfig holds HTTP client configuration.
type HTTPConfig struct {
	// Timeout is the default HTTP request timeout.
	// Default: 30s
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts.
	// Default: 3
	MaxRetries int

	// RetryDelays are the delays between retry attempts.
	// Default: [0s, 5s, 30s]
	RetryDelays []time.Duration
}

// RetryQueueConfig holds retry queue configuration.
type RetryQueueConfig struct {
	// CheckInterval is how often the queue checks for ready notifications.
	// Default: 30s
	CheckInterval time.Duration

	// BackoffSchedule is the exponential backoff schedule for failed notifications.
	// Default: [5s, 30s, 2m, 5m, 15m]
	BackoffSchedule []time.Duration
}

// StorageConfig holds storage-related configuration.
type StorageConfig struct {
	// MinFreeSpace is the minimum free space required for write operations.
	// Default: 10MB (10 * 1024 * 1024 bytes)
	MinFreeSpace uint64

	// MinFreeSpaceWarning is the threshold for warning about low disk space.
	// Default: 50MB (50 * 1024 * 1024 bytes)
	MinFreeSpaceWarning uint64
}

// SchedulerConfig holds scheduler-related configuration.
type SchedulerConfig struct {
	// IdleNotificationCooldown is the minimum time between idle notifications.
	// Default: 30m
	IdleNotificationCooldown time.Duration

	// SleepThreshold is the time gap that indicates the system was sleeping.
	// If elapsed time since last check exceeds this, stale checks are skipped.
	// Default: 1h
	SleepThreshold time.Duration
}

// DefaultRuntimeConfig returns the default runtime configuration.
func DefaultRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		Daemon: DaemonConfig{
			StartupWait: 500 * time.Millisecond,
			KillTimeout: 5 * time.Second,
		},
		HTTP: HTTPConfig{
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			RetryDelays: []time.Duration{
				0,                // Immediate first attempt
				5 * time.Second,  // Retry after 5s
				30 * time.Second, // Retry after 30s
			},
		},
		RetryQueue: RetryQueueConfig{
			CheckInterval: 30 * time.Second,
			BackoffSchedule: []time.Duration{
				5 * time.Second,
				30 * time.Second,
				2 * time.Minute,
				5 * time.Minute,
				15 * time.Minute,
			},
		},
		Storage: StorageConfig{
			MinFreeSpace:        10 * 1024 * 1024, // 10MB
			MinFreeSpaceWarning: 50 * 1024 * 1024, // 50MB
		},
		Scheduler: SchedulerConfig{
			IdleNotificationCooldown: 30 * time.Minute,
			SleepThreshold:           1 * time.Hour,
		},
	}
}

// Global holds the global runtime configuration instance.
// It is initialized with defaults and can be overridden via environment variables.
var Global = initGlobal()

// initGlobal initializes the global config with defaults and environment overrides.
func initGlobal() *RuntimeConfig {
	cfg := DefaultRuntimeConfig()
	cfg.loadFromEnv()
	return cfg
}

// loadFromEnv loads configuration overrides from environment variables.
func (c *RuntimeConfig) loadFromEnv() {
	// Daemon configuration
	if v := os.Getenv("HUMANTIME_DAEMON_STARTUP_WAIT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Daemon.StartupWait = d
		}
	}
	if v := os.Getenv("HUMANTIME_DAEMON_KILL_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Daemon.KillTimeout = d
		}
	}

	// HTTP configuration
	if v := os.Getenv("HUMANTIME_HTTP_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.HTTP.Timeout = d
		}
	}
	if v := os.Getenv("HUMANTIME_HTTP_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			c.HTTP.MaxRetries = n
		}
	}

	// Retry queue configuration
	if v := os.Getenv("HUMANTIME_RETRY_QUEUE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.RetryQueue.CheckInterval = d
		}
	}

	// Storage configuration
	if v := os.Getenv("HUMANTIME_MIN_FREE_SPACE"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			c.Storage.MinFreeSpace = n
		}
	}
	if v := os.Getenv("HUMANTIME_MIN_FREE_SPACE_WARNING"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			c.Storage.MinFreeSpaceWarning = n
		}
	}

	// Scheduler configuration
	if v := os.Getenv("HUMANTIME_IDLE_NOTIFICATION_COOLDOWN"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Scheduler.IdleNotificationCooldown = d
		}
	}
	if v := os.Getenv("HUMANTIME_SLEEP_THRESHOLD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Scheduler.SleepThreshold = d
		}
	}
}

// ReloadFromEnv reloads configuration from environment variables.
// This is useful for testing or when environment variables change.
func (c *RuntimeConfig) ReloadFromEnv() {
	c.loadFromEnv()
}

// Reset resets the configuration to defaults.
// This is primarily useful for testing.
func (c *RuntimeConfig) Reset() {
	defaults := DefaultRuntimeConfig()
	*c = *defaults
}
