package daemon

import (
	"encoding/json"
	"runtime"
	"sync"
	"time"
)

// HealthStatus represents the current health state of the daemon.
type HealthStatus struct {
	Status              string    `json:"status"`
	UptimeSeconds       int64     `json:"uptime_seconds"`
	MemoryMB            float64   `json:"memory_mb"`
	PendingNotifications int       `json:"pending_notifications"`
	LastCheck           time.Time `json:"last_check"`
	Version             string    `json:"version,omitempty"`
	Goroutines          int       `json:"goroutines"`
}

// HealthChecker provides health status for the daemon.
type HealthChecker struct {
	mu               sync.RWMutex
	startTime        time.Time
	lastCheck        time.Time
	pendingNotifs    int
	version          string
	customChecks     map[string]func() error
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(version string) *HealthChecker {
	return &HealthChecker{
		startTime:    time.Now(),
		version:      version,
		customChecks: make(map[string]func() error),
	}
}

// Check performs a health check and returns the status.
func (h *HealthChecker) Check() *HealthStatus {
	h.mu.Lock()
	h.lastCheck = time.Now()
	h.mu.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	h.mu.RLock()
	pending := h.pendingNotifs
	h.mu.RUnlock()

	return &HealthStatus{
		Status:              h.determineStatus(),
		UptimeSeconds:       int64(time.Since(h.startTime).Seconds()),
		MemoryMB:            float64(memStats.Alloc) / 1024 / 1024,
		PendingNotifications: pending,
		LastCheck:           h.lastCheck,
		Version:             h.version,
		Goroutines:          runtime.NumGoroutine(),
	}
}

// determineStatus checks all health indicators and returns the status.
func (h *HealthChecker) determineStatus() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Run custom checks
	for _, check := range h.customChecks {
		if err := check(); err != nil {
			return "unhealthy"
		}
	}

	return "healthy"
}

// SetPendingNotifications updates the pending notification count.
func (h *HealthChecker) SetPendingNotifications(count int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingNotifs = count
}

// AddCheck adds a custom health check function.
func (h *HealthChecker) AddCheck(name string, check func() error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.customChecks[name] = check
}

// RemoveCheck removes a custom health check.
func (h *HealthChecker) RemoveCheck(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.customChecks, name)
}

// JSON returns the health status as JSON.
func (h *HealthChecker) JSON() ([]byte, error) {
	status := h.Check()
	return json.MarshalIndent(status, "", "  ")
}

// Uptime returns how long the daemon has been running.
func (h *HealthChecker) Uptime() time.Duration {
	return time.Since(h.startTime)
}

// IsHealthy returns true if the daemon is healthy.
func (h *HealthChecker) IsHealthy() bool {
	return h.determineStatus() == "healthy"
}

// DetailedHealth provides more detailed health information.
type DetailedHealth struct {
	HealthStatus
	MemoryDetails MemoryDetails `json:"memory_details"`
	Checks        []CheckResult `json:"checks"`
}

// MemoryDetails provides detailed memory statistics.
type MemoryDetails struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	NumGC        uint32  `json:"num_gc"`
	LastGC       string  `json:"last_gc,omitempty"`
}

// CheckResult represents the result of a single health check.
type CheckResult struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

// DetailedCheck performs a detailed health check.
func (h *HealthChecker) DetailedCheck() *DetailedHealth {
	basic := h.Check()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	details := &DetailedHealth{
		HealthStatus: *basic,
		MemoryDetails: MemoryDetails{
			AllocMB:      float64(memStats.Alloc) / 1024 / 1024,
			TotalAllocMB: float64(memStats.TotalAlloc) / 1024 / 1024,
			SysMB:        float64(memStats.Sys) / 1024 / 1024,
			NumGC:        memStats.NumGC,
		},
	}

	if memStats.LastGC > 0 {
		details.MemoryDetails.LastGC = time.Unix(0, int64(memStats.LastGC)).Format(time.RFC3339)
	}

	// Run and report individual checks
	h.mu.RLock()
	for name, check := range h.customChecks {
		result := CheckResult{Name: name, Healthy: true}
		if err := check(); err != nil {
			result.Healthy = false
			result.Error = err.Error()
		}
		details.Checks = append(details.Checks, result)
	}
	h.mu.RUnlock()

	return details
}
