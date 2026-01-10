package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/manav03panchal/humantime/internal/scheduler"
	"github.com/manav03panchal/humantime/internal/storage"
)

// Daemon manages the background daemon process.
type Daemon struct {
	pidFile     *PIDFile
	scheduler   *scheduler.Scheduler
	db          *storage.DB
	reminderRepo *storage.ReminderRepo
	webhookRepo  *storage.WebhookRepo
	startedAt   time.Time
	debug       bool
}

// Status represents the daemon status.
type Status struct {
	Running   bool      `json:"running"`
	PID       int       `json:"pid,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
}

// NewDaemon creates a new daemon manager.
func NewDaemon(db *storage.DB) *Daemon {
	return &Daemon{
		pidFile:      NewPIDFile(),
		db:           db,
		reminderRepo: storage.NewReminderRepo(db),
		webhookRepo:  storage.NewWebhookRepo(db),
	}
}

// SetDebug enables debug mode.
func (d *Daemon) SetDebug(debug bool) {
	d.debug = debug
}

// GetStatus returns the current daemon status.
func (d *Daemon) GetStatus() *Status {
	status := &Status{}

	pid := d.pidFile.GetRunningPID()
	if pid > 0 {
		status.Running = true
		status.PID = pid

		// Try to read start time from state file
		if state, err := d.readState(); err == nil {
			status.StartedAt = state.StartedAt
			status.Uptime = formatUptime(time.Since(state.StartedAt))
		}
	}

	return status
}

// IsRunning returns true if the daemon is running.
func (d *Daemon) IsRunning() bool {
	return d.pidFile.IsRunning()
}

// Start starts the daemon in the foreground.
func (d *Daemon) Start(ctx context.Context) error {
	if d.IsRunning() {
		return ErrAlreadyRunning
	}

	// Write PID file
	if err := d.pidFile.Write(); err != nil {
		return err
	}

	// Record start time
	d.startedAt = time.Now()
	if err := d.writeState(&DaemonState{
		StartedAt: d.startedAt,
	}); err != nil {
		d.pidFile.Remove()
		return err
	}

	// Create scheduler
	d.scheduler = scheduler.NewScheduler(d.db)
	d.scheduler.SetDebug(d.debug)

	// Set up reminder checker
	reminderChecker := scheduler.NewReminderChecker(d.reminderRepo, d.webhookRepo)
	d.scheduler.SetReminderChecker(reminderChecker)

	// Start scheduler
	if err := d.scheduler.Start(); err != nil {
		d.pidFile.Remove()
		return err
	}

	// Setup signal handler
	sigHandler := NewSignalHandler()
	sigHandler.Setup()
	defer sigHandler.Cleanup()

	if d.debug {
		fmt.Printf("[DEBUG] Daemon started (PID: %d)\n", os.Getpid())
	}

	// Wait for shutdown signal
	sig := sigHandler.Wait(ctx)
	if d.debug && sig != nil {
		fmt.Printf("[DEBUG] Received signal: %v\n", sig)
	}

	// Cleanup
	d.scheduler.Stop()
	d.pidFile.Remove()
	d.removeState()

	return nil
}

// StartBackground starts the daemon in the background.
func (d *Daemon) StartBackground() (int, error) {
	if d.IsRunning() {
		return d.pidFile.GetRunningPID(), ErrAlreadyRunning
	}

	// Get the path to the current executable
	executable, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build command arguments
	args := []string{"daemon", "start", "--foreground"}
	if d.debug {
		args = append(args, "--debug")
	}

	// Create command
	cmd := exec.Command(executable, args...)

	// Detach from terminal
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Set up log file
	logPath := GetLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err == nil {
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			cmd.Stdout = logFile
			cmd.Stderr = logFile
		}
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait a moment for the process to start and write PID
	time.Sleep(500 * time.Millisecond)

	// Verify it's running
	if !d.pidFile.IsRunning() {
		return 0, fmt.Errorf("daemon failed to start")
	}

	return cmd.Process.Pid, nil
}

// Stop stops the running daemon.
func (d *Daemon) Stop() error {
	pid := d.pidFile.GetRunningPID()
	if pid == 0 {
		return ErrNotRunning
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM
	if err := process.Signal(os.Interrupt); err != nil {
		// Try SIGKILL as fallback
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-done:
		// Process exited
	case <-time.After(5 * time.Second):
		// Force kill
		process.Kill()
	}

	// Clean up PID file if still exists
	d.pidFile.Remove()
	d.removeState()

	return nil
}

// DaemonState holds persistent daemon state.
type DaemonState struct {
	StartedAt time.Time `json:"started_at"`
}

// getStatePath returns the path to the state file.
func getStatePath() string {
	return filepath.Join(xdg.StateHome, AppName, "daemon.json")
}

// writeState writes daemon state to file.
func (d *Daemon) writeState(state *DaemonState) error {
	path := getStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// readState reads daemon state from file.
func (d *Daemon) readState() (*DaemonState, error) {
	data, err := os.ReadFile(getStatePath())
	if err != nil {
		return nil, err
	}

	var state DaemonState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// removeState removes the state file.
func (d *Daemon) removeState() {
	os.Remove(getStatePath())
}

// GetLogPath returns the path to the daemon log file.
func GetLogPath() string {
	return filepath.Join(xdg.StateHome, AppName, "daemon.log")
}

// formatUptime formats a duration as uptime.
func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dd", days)
}
