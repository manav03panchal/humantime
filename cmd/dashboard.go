package cmd

import (
	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/tui"
)

// dashboardCmd represents the dashboard command.
var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"dash", "d", "tui"},
	Short:   "Open the interactive TUI dashboard",
	Long: `Open an interactive terminal dashboard to view and manage time tracking.

The dashboard shows:
  - Current tracking status with live elapsed time
  - Recent time blocks
  - Goal progress (if goals are set)

Keyboard Controls:
  s - Start tracking (shows instructions)
  e - Stop current tracking
  r - Refresh data
  q - Quit dashboard

Examples:
  humantime dashboard
  humantime dash
  humantime tui`,
	RunE: runDashboard,
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}

func runDashboard(cmd *cobra.Command, args []string) error {
	// Configure the dashboard
	config := tui.DashboardConfig{
		BlockRepo:       ctx.BlockRepo,
		ActiveBlockRepo: ctx.ActiveBlockRepo,
		GoalRepo:        ctx.GoalRepo,
	}

	// Run the TUI dashboard
	return tui.Run(config)
}
