package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"

	"github.com/adrg/xdg"
)

// ServiceManager handles system service installation.
type ServiceManager struct {
	executablePath string
	debug          bool
}

// NewServiceManager creates a new service manager.
func NewServiceManager() (*ServiceManager, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	return &ServiceManager{
		executablePath: execPath,
	}, nil
}

// SetDebug enables debug output.
func (m *ServiceManager) SetDebug(debug bool) {
	m.debug = debug
}

// Install installs the daemon as a system service.
func (m *ServiceManager) Install() error {
	switch runtime.GOOS {
	case "darwin":
		return m.installLaunchd()
	case "linux":
		return m.installSystemd()
	case "windows":
		return fmt.Errorf("Windows service installation not yet supported")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Uninstall removes the daemon from system services.
func (m *ServiceManager) Uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return m.uninstallLaunchd()
	case "linux":
		return m.uninstallSystemd()
	case "windows":
		return fmt.Errorf("Windows service uninstallation not yet supported")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// IsInstalled checks if the service is installed.
func (m *ServiceManager) IsInstalled() bool {
	switch runtime.GOOS {
	case "darwin":
		return m.isLaunchdInstalled()
	case "linux":
		return m.isSystemdInstalled()
	default:
		return false
	}
}

// macOS launchd support

const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.humantime.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ExecutablePath}}</string>
        <string>daemon</string>
        <string>start</string>
        <string>--foreground</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>
    <key>WorkingDirectory</key>
    <string>{{.WorkingDirectory}}</string>
</dict>
</plist>
`

func (m *ServiceManager) getLaunchdPath() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.humantime.daemon.plist")
}

func (m *ServiceManager) installLaunchd() error {
	plistPath := m.getLaunchdPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Generate plist content
	tmpl, err := template.New("plist").Parse(launchdPlist)
	if err != nil {
		return fmt.Errorf("failed to parse plist template: %w", err)
	}

	data := struct {
		ExecutablePath   string
		LogPath          string
		WorkingDirectory string
	}{
		ExecutablePath:   m.executablePath,
		LogPath:          GetLogPath(),
		WorkingDirectory: filepath.Dir(m.executablePath),
	}

	file, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("failed to create plist file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	// Load the service
	cmd := exec.Command("launchctl", "load", plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to load service: %w: %s", err, string(output))
	}

	if m.debug {
		fmt.Printf("[DEBUG] Installed launchd service at %s\n", plistPath)
	}

	return nil
}

func (m *ServiceManager) uninstallLaunchd() error {
	plistPath := m.getLaunchdPath()

	// Unload the service first
	cmd := exec.Command("launchctl", "unload", plistPath)
	cmd.Run() // Ignore error if not loaded

	// Remove the plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	if m.debug {
		fmt.Printf("[DEBUG] Uninstalled launchd service from %s\n", plistPath)
	}

	return nil
}

func (m *ServiceManager) isLaunchdInstalled() bool {
	_, err := os.Stat(m.getLaunchdPath())
	return err == nil
}

// Linux systemd support

const systemdUnit = `[Unit]
Description=Humantime Background Daemon
After=network.target

[Service]
Type=simple
ExecStart={{.ExecutablePath}} daemon start --foreground
Restart=on-failure
RestartSec=5
StandardOutput=append:{{.LogPath}}
StandardError=append:{{.LogPath}}
Environment="HOME={{.HomeDirectory}}"
Environment="XDG_DATA_HOME={{.DataHome}}"
Environment="XDG_STATE_HOME={{.StateHome}}"

[Install]
WantedBy=default.target
`

func (m *ServiceManager) getSystemdPath() string {
	return filepath.Join(xdg.ConfigHome, "systemd", "user", "humantime.service")
}

func (m *ServiceManager) installSystemd() error {
	unitPath := m.getSystemdPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(unitPath), 0755); err != nil {
		return fmt.Errorf("failed to create systemd user directory: %w", err)
	}

	// Generate unit content
	tmpl, err := template.New("unit").Parse(systemdUnit)
	if err != nil {
		return fmt.Errorf("failed to parse unit template: %w", err)
	}

	data := struct {
		ExecutablePath string
		LogPath        string
		HomeDirectory  string
		DataHome       string
		StateHome      string
	}{
		ExecutablePath: m.executablePath,
		LogPath:        GetLogPath(),
		HomeDirectory:  os.Getenv("HOME"),
		DataHome:       xdg.DataHome,
		StateHome:      xdg.StateHome,
	}

	file, err := os.Create(unitPath)
	if err != nil {
		return fmt.Errorf("failed to create unit file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to write unit: %w", err)
	}

	// Reload systemd
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w: %s", err, string(output))
	}

	// Enable the service
	cmd = exec.Command("systemctl", "--user", "enable", "humantime.service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable service: %w: %s", err, string(output))
	}

	// Start the service
	cmd = exec.Command("systemctl", "--user", "start", "humantime.service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start service: %w: %s", err, string(output))
	}

	if m.debug {
		fmt.Printf("[DEBUG] Installed systemd user service at %s\n", unitPath)
	}

	return nil
}

func (m *ServiceManager) uninstallSystemd() error {
	unitPath := m.getSystemdPath()

	// Stop the service
	cmd := exec.Command("systemctl", "--user", "stop", "humantime.service")
	cmd.Run() // Ignore error if not running

	// Disable the service
	cmd = exec.Command("systemctl", "--user", "disable", "humantime.service")
	cmd.Run() // Ignore error if not enabled

	// Remove the unit file
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}

	// Reload systemd
	cmd = exec.Command("systemctl", "--user", "daemon-reload")
	cmd.Run() // Ignore reload errors

	if m.debug {
		fmt.Printf("[DEBUG] Uninstalled systemd user service from %s\n", unitPath)
	}

	return nil
}

func (m *ServiceManager) isSystemdInstalled() bool {
	_, err := os.Stat(m.getSystemdPath())
	return err == nil
}
