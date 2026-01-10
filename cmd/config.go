package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
)

// configCmd represents the config command.
var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"cfg", "settings"},
	Short:   "Manage application configuration",
	Long: `View and modify application configuration settings.

Examples:
  humantime config get notify
  humantime config set notify.idle-after 45m
  humantime config set notify.break-after 90m
  humantime config set notify.daily-summary 08:30
  humantime config set notify.idle enabled
  humantime config set notify.break disabled`,
}

// configGetCmd gets configuration values.
var configGetCmd = &cobra.Command{
	Use:   "get [KEY]",
	Short: "Get configuration value",
	Long: `Get a configuration value or show all values in a section.

Keys:
  notify              Show all notification settings
  notify.idle-after   Idle detection threshold
  notify.break-after  Break reminder threshold
  notify.break-reset  Break reset gap duration
  notify.goal-progress  Goal progress milestones
  notify.daily-summary  Daily summary time (HH:MM)
  notify.end-of-day   End of day recap time (HH:MM)
  notify.<type>       Enable status for notification type

Examples:
  humantime config get notify
  humantime config get notify.idle-after`,
	RunE: runConfigGet,
}

// configSetCmd sets configuration values.
var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set configuration value",
	Long: `Set a configuration value.

Keys and values:
  notify.idle-after DURATION    Idle threshold (e.g., 30m, 1h)
  notify.break-after DURATION   Break threshold (e.g., 2h, 90m) or 0 to disable
  notify.break-reset DURATION   Break reset gap (e.g., 15m)
  notify.goal-progress LIST     Milestones (e.g., 50,75,100)
  notify.daily-summary TIME     Time for morning summary (HH:MM) or "" to disable
  notify.end-of-day TIME        Time for EOD recap (HH:MM) or "" to disable
  notify.<type> enabled|disabled Enable/disable notification type

Notification types: idle, break, goal, daily_summary, end_of_day, reminder

Examples:
  humantime config set notify.idle-after 45m
  humantime config set notify.break-after 0
  humantime config set notify.idle disabled
  humantime config set notify.goal-progress 25,50,75,100`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

// runConfigGet handles the config get command.
func runConfigGet(cmd *cobra.Command, args []string) error {
	key := ""
	if len(args) > 0 {
		key = args[0]
	}

	// Handle notification config
	if key == "" || strings.HasPrefix(key, "notify") {
		return getNotifyConfig(key)
	}

	return fmt.Errorf("unknown config key: %s", key)
}

// getNotifyConfig displays notification configuration.
func getNotifyConfig(key string) error {
	config, err := ctx.NotifyConfigRepo.Get()
	if err != nil {
		return err
	}

	// If specific key requested
	if key != "" && key != "notify" {
		field := strings.TrimPrefix(key, "notify.")
		return printNotifyField(config, field)
	}

	// Show all notification settings
	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"idle_after":      config.IdleAfter.String(),
			"break_after":     config.BreakAfter.String(),
			"break_reset":     config.BreakReset.String(),
			"goal_milestones": config.GoalMilestones,
			"daily_summary":   config.DailySummaryAt,
			"end_of_day":      config.EndOfDayAt,
			"enabled":         config.Enabled,
		})
	}

	ctx.Formatter.Println("Notification Settings:")
	ctx.Formatter.Println("")
	ctx.Formatter.Printf("  idle-after:     %s\n", config.IdleAfter)
	ctx.Formatter.Printf("  break-after:    %s\n", config.BreakAfter)
	ctx.Formatter.Printf("  break-reset:    %s\n", config.BreakReset)
	ctx.Formatter.Printf("  goal-progress:  %v\n", config.GoalMilestones)
	ctx.Formatter.Printf("  daily-summary:  %s\n", config.DailySummaryAt)
	ctx.Formatter.Printf("  end-of-day:     %s\n", config.EndOfDayAt)
	ctx.Formatter.Println("")
	ctx.Formatter.Println("Notification Types:")
	for typeName, enabled := range config.Enabled {
		status := "enabled"
		if !enabled {
			status = "disabled"
		}
		ctx.Formatter.Printf("  %-14s  %s\n", typeName+":", status)
	}

	return nil
}

// printNotifyField prints a single notification config field.
func printNotifyField(config *model.NotifyConfig, field string) error {
	var value interface{}

	switch field {
	case "idle-after":
		value = config.IdleAfter.String()
	case "break-after":
		value = config.BreakAfter.String()
	case "break-reset":
		value = config.BreakReset.String()
	case "goal-progress":
		value = config.GoalMilestones
	case "daily-summary":
		value = config.DailySummaryAt
	case "end-of-day":
		value = config.EndOfDayAt
	case "idle", "break", "goal", "daily_summary", "end_of_day", "reminder":
		if config.IsTypeEnabled(field) {
			value = "enabled"
		} else {
			value = "disabled"
		}
	default:
		return fmt.Errorf("unknown notify field: %s", field)
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"key":   "notify." + field,
			"value": value,
		})
	}

	ctx.Formatter.Printf("%v\n", value)
	return nil
}

// runConfigSet handles the config set command.
func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	if !strings.HasPrefix(key, "notify.") {
		return fmt.Errorf("unknown config key: %s", key)
	}

	field := strings.TrimPrefix(key, "notify.")
	return setNotifyField(field, value)
}

// setNotifyField sets a notification config field.
func setNotifyField(field, value string) error {
	config, err := ctx.NotifyConfigRepo.Get()
	if err != nil {
		return err
	}

	switch field {
	case "idle-after":
		duration, err := parseNotifyDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		config.IdleAfter = duration

	case "break-after":
		duration, err := parseNotifyDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		config.BreakAfter = duration

	case "break-reset":
		duration, err := parseNotifyDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		config.BreakReset = duration

	case "goal-progress":
		milestones, err := parseMilestones(value)
		if err != nil {
			return fmt.Errorf("invalid milestones: %w", err)
		}
		config.GoalMilestones = milestones

	case "daily-summary":
		if value != "" && value != "disabled" && value != "off" {
			if _, err := parseTimeOfDay(value); err != nil {
				return fmt.Errorf("invalid time format: %w", err)
			}
			config.DailySummaryAt = value
		} else {
			config.DailySummaryAt = ""
		}

	case "end-of-day":
		if value != "" && value != "disabled" && value != "off" {
			if _, err := parseTimeOfDay(value); err != nil {
				return fmt.Errorf("invalid time format: %w", err)
			}
			config.EndOfDayAt = value
		} else {
			config.EndOfDayAt = ""
		}

	case "idle", "break", "goal", "daily_summary", "end_of_day", "reminder":
		enabled, err := parseEnabled(value)
		if err != nil {
			return err
		}
		config.SetTypeEnabled(field, enabled)

	default:
		return fmt.Errorf("unknown notify field: %s", field)
	}

	// Validate
	if err := config.Validate(); err != nil {
		return err
	}

	// Save
	if err := ctx.NotifyConfigRepo.Set(config); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"status": "updated",
			"key":    "notify." + field,
			"value":  value,
		})
	}

	ctx.Formatter.Printf("Updated notify.%s = %s\n", field, value)
	return nil
}

// parseNotifyDuration parses a duration string like "30m", "2h", "1h30m".
func parseNotifyDuration(s string) (time.Duration, error) {
	if s == "0" || s == "off" || s == "disabled" {
		return 0, nil
	}

	// Try standard Go duration format
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s (try 30m, 2h, 1h30m)", s)
}

// parseMilestones parses a comma-separated list of milestone percentages.
func parseMilestones(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	var milestones []int

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid milestone: %s", p)
		}
		if n < 1 || n > 100 {
			return nil, fmt.Errorf("milestone must be 1-100: %d", n)
		}
		milestones = append(milestones, n)
	}

	if len(milestones) == 0 {
		return nil, fmt.Errorf("at least one milestone required")
	}

	return milestones, nil
}

// parseTimeOfDay parses a time string in HH:MM format.
func parseTimeOfDay(s string) (time.Time, error) {
	t, err := time.Parse("15:04", s)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse("3:04", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s (use HH:MM)", s)
}

// parseEnabled parses an enabled/disabled value.
func parseEnabled(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "enabled", "on", "true", "yes", "1":
		return true, nil
	case "disabled", "off", "false", "no", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value: %s (use enabled/disabled)", s)
	}
}
