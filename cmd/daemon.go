package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/daemon"
	"github.com/manav03panchal/humantime/internal/notify"
)

// Daemon command flags.
var (
	daemonStartFlagForeground bool
	daemonLogsFlagTail        int
	daemonLogsFlagFollow      bool
	daemonInstallFlagForce    bool
)

// daemonCmd represents the daemon command.
var daemonCmd = &cobra.Command{
	Use:     "daemon [command]",
	Aliases: []string{"d", "bg", "service"},
	Short:   "Manage the background daemon",
	Long: `Manage the Humantime background daemon that monitors activity
and sends notifications for reminders, idle detection, break reminders,
goal progress, and daily summaries.

Examples:
  humantime daemon start
  humantime daemon status
  humantime daemon stop
  humantime daemon logs --tail 20`,
	RunE: runDaemonStatus,
}

// daemonStartCmd starts the daemon.
var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the background daemon",
	Long: `Start the Humantime background daemon.

The daemon monitors your time tracking and sends notifications via configured webhooks.

Examples:
  humantime daemon start           # Start in background
  humantime daemon start -f        # Start in foreground (for debugging)`,
	RunE: runDaemonStart,
}

// daemonStopCmd stops the daemon.
var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background daemon",
	RunE:  runDaemonStop,
}

// daemonStatusCmd shows daemon status.
var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	RunE:  runDaemonStatus,
}

// daemonLogsCmd shows daemon logs.
var daemonLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View daemon logs",
	Long: `View the daemon log file.

Examples:
  humantime daemon logs
  humantime daemon logs --tail 50
  humantime daemon logs -f`,
	RunE: runDaemonLogs,
}

// daemonInstallCmd installs the daemon as a system service.
var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install daemon as a system service",
	Long: `Install the Humantime daemon as a system service that starts automatically on login.

On macOS, this creates a launchd agent in ~/Library/LaunchAgents.
On Linux, this creates a systemd user service in ~/.config/systemd/user.

Examples:
  humantime daemon install
  humantime daemon install --force   # Reinstall if already installed`,
	RunE: runDaemonInstall,
}

// daemonUninstallCmd uninstalls the daemon system service.
var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall daemon system service",
	Long: `Remove the Humantime daemon from system services.

This stops the service and removes the service configuration.`,
	RunE: runDaemonUninstall,
}

func init() {
	// Add flags
	daemonStartCmd.Flags().BoolVar(&daemonStartFlagForeground, "foreground", false,
		"Run in foreground (don't daemonize)")

	daemonLogsCmd.Flags().IntVarP(&daemonLogsFlagTail, "tail", "n", 20,
		"Number of lines to show")
	daemonLogsCmd.Flags().BoolVar(&daemonLogsFlagFollow, "follow", false,
		"Follow log output (like tail -f)")

	daemonInstallCmd.Flags().BoolVar(&daemonInstallFlagForce, "force", false,
		"Force reinstall if already installed")

	// Add subcommands
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonLogsCmd)
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)

	rootCmd.AddCommand(daemonCmd)
}

// runDaemonStart handles the daemon start command.
func runDaemonStart(cmd *cobra.Command, args []string) error {
	// For background start, ctx may be nil (we skip DB init to avoid lock conflicts)
	if !daemonStartFlagForeground {
		// Background mode - spawn child process without holding database lock
		d := daemon.NewDaemon(nil)
		d.SetDebug(flagDebug)

		// Check if already running
		if d.IsRunning() {
			status := d.GetStatus()
			return fmt.Errorf("daemon is already running (PID: %d)", status.PID)
		}

		// Start in background
		pid, err := d.StartBackground()
		if err != nil {
			return err
		}

		fmt.Println("Starting humantime daemon...")
		fmt.Printf("Daemon started (PID: %d)\n", pid)
		return nil
	}

	// Foreground mode - ctx is initialized
	d := daemon.NewDaemon(ctx.DB)
	d.SetDebug(ctx.Debug)

	// Check if already running
	if d.IsRunning() {
		status := d.GetStatus()
		if ctx.IsJSON() {
			return ctx.Formatter.PrintJSON(map[string]interface{}{
				"status": "already_running",
				"pid":    status.PID,
			})
		}
		return fmt.Errorf("daemon is already running (PID: %d)", status.PID)
	}

	// Check for webhooks
	dispatcher := notify.NewDispatcher(ctx.WebhookRepo)
	webhookCount := dispatcher.CountEnabledWebhooks()
	if webhookCount == 0 && !ctx.IsJSON() {
		ctx.Formatter.Println("Warning: no webhooks configured. Add with: humantime webhook add")
		ctx.Formatter.Println("")
	}

	// Run in foreground
	if !ctx.IsJSON() {
		ctx.Formatter.Printf("Starting humantime daemon (foreground mode)...\n")
	}
	return d.Start(context.Background())
}

// runDaemonStop handles the daemon stop command.
func runDaemonStop(cmd *cobra.Command, args []string) error {
	d := daemon.NewDaemon(nil)

	if !d.IsRunning() {
		fmt.Println("Daemon is not running")
		return nil
	}

	status := d.GetStatus()
	pid := status.PID

	fmt.Println("Stopping humantime daemon...")

	if err := d.Stop(); err != nil {
		return err
	}

	fmt.Printf("Daemon stopped (was PID: %d)\n", pid)
	return nil
}

// runDaemonStatus handles the daemon status command.
func runDaemonStatus(cmd *cobra.Command, args []string) error {
	d := daemon.NewDaemon(nil)
	status := d.GetStatus()

	fmt.Println("Humantime Daemon Status")
	fmt.Println("")

	if status.Running {
		fmt.Printf("  Status:    running\n")
		fmt.Printf("  PID:       %d\n", status.PID)
		fmt.Printf("  Uptime:    %s\n", status.Uptime)
	} else {
		fmt.Printf("  Status:    stopped\n")
		fmt.Println("")
		fmt.Println("Start with: humantime daemon start")
	}

	return nil
}

// runDaemonLogs handles the daemon logs command.
func runDaemonLogs(cmd *cobra.Command, args []string) error {
	logPath := daemon.GetLogPath()

	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		fmt.Println("No log file found.")
		fmt.Printf("Log path: %s\n", logPath)
		return nil
	}

	if daemonLogsFlagFollow {
		// Follow mode - tail -f style
		return followLogs(logPath)
	}

	// Read last N lines
	lines, err := tailFile(logPath, daemonLogsFlagTail)
	if err != nil {
		return err
	}

	for _, line := range lines {
		fmt.Println(line)
	}

	return nil
}

// tailFile reads the last n lines from a file.
func tailFile(path string, n int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// followLogs follows the log file in real-time.
func followLogs(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Seek to end
	file.Seek(0, 2)

	scanner := bufio.NewScanner(file)
	for {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return err
		}

		// Reset scanner and wait for more data
		// This is a simple implementation - production would use fsnotify
		select {
		case <-context.Background().Done():
			return nil
		default:
			// Small delay before checking for new lines
			scanner = bufio.NewScanner(file)
		}
	}
}

// formatWebhookNames formats a list of webhook names.
func formatWebhookNames(webhooks []string, count int) string {
	if len(webhooks) == 0 {
		return "none"
	}
	if len(webhooks) > 3 {
		return strings.Join(webhooks[:3], ", ") + " (+" + strconv.Itoa(len(webhooks)-3) + " more)"
	}
	return strings.Join(webhooks, ", ") + " (" + strconv.Itoa(count) + " enabled)"
}

// runDaemonInstall handles the daemon install command.
func runDaemonInstall(cmd *cobra.Command, args []string) error {
	mgr, err := daemon.NewServiceManager()
	if err != nil {
		return err
	}
	mgr.SetDebug(ctx.Debug)

	// Check if already installed
	if mgr.IsInstalled() && !daemonInstallFlagForce {
		if ctx.IsJSON() {
			return ctx.Formatter.PrintJSON(map[string]interface{}{
				"status": "already_installed",
			})
		}
		ctx.Formatter.Println("Service is already installed.")
		ctx.Formatter.Println("Use --force to reinstall.")
		return nil
	}

	// Uninstall first if reinstalling
	if mgr.IsInstalled() && daemonInstallFlagForce {
		if !ctx.IsJSON() {
			ctx.Formatter.Println("Removing existing service...")
		}
		if err := mgr.Uninstall(); err != nil {
			return fmt.Errorf("failed to remove existing service: %w", err)
		}
	}

	// Install the service
	if !ctx.IsJSON() {
		ctx.Formatter.Println("Installing Humantime daemon as system service...")
	}

	if err := mgr.Install(); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"status":  "installed",
			"message": "Service will start automatically on login",
		})
	}

	ctx.Formatter.Println("")
	ctx.Formatter.Println("✓ Service installed successfully")
	ctx.Formatter.Println("")
	ctx.Formatter.Println("The daemon will now start automatically when you log in.")
	ctx.Formatter.Println("To start it now: humantime daemon start")
	ctx.Formatter.Println("To remove: humantime daemon uninstall")

	return nil
}

// runDaemonUninstall handles the daemon uninstall command.
func runDaemonUninstall(cmd *cobra.Command, args []string) error {
	mgr, err := daemon.NewServiceManager()
	if err != nil {
		return err
	}
	mgr.SetDebug(ctx.Debug)

	// Check if installed
	if !mgr.IsInstalled() {
		if ctx.IsJSON() {
			return ctx.Formatter.PrintJSON(map[string]interface{}{
				"status": "not_installed",
			})
		}
		ctx.Formatter.Println("Service is not installed.")
		return nil
	}

	// Stop the daemon if running
	d := daemon.NewDaemon(ctx.DB)
	if d.IsRunning() {
		if !ctx.IsJSON() {
			ctx.Formatter.Println("Stopping running daemon...")
		}
		if err := d.Stop(); err != nil {
			// Continue anyway - we want to uninstall
			if ctx.Debug {
				ctx.Formatter.Printf("[DEBUG] Warning: failed to stop daemon: %v\n", err)
			}
		}
	}

	// Uninstall the service
	if !ctx.IsJSON() {
		ctx.Formatter.Println("Uninstalling Humantime daemon service...")
	}

	if err := mgr.Uninstall(); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"status": "uninstalled",
		})
	}

	ctx.Formatter.Println("")
	ctx.Formatter.Println("✓ Service uninstalled successfully")
	ctx.Formatter.Println("")
	ctx.Formatter.Println("The daemon will no longer start automatically.")
	ctx.Formatter.Println("To reinstall: humantime daemon install")

	return nil
}
