package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/notify"
	"github.com/manav03panchal/humantime/internal/runtime"
)

// Webhook command flags.
var (
	webhookAddFlagType     string
	webhookAddFlagTemplate string
	webhookRemoveFlagForce bool
	webhookTestFlagAll     bool
)

// webhookCmd represents the webhook command.
var webhookCmd = &cobra.Command{
	Use:     "webhook [command]",
	Aliases: []string{"w", "wh", "hook"},
	Short:   "Configure notification webhooks",
	Long: `Configure webhooks for Discord, Slack, Teams, or custom endpoints.

Webhooks receive notifications for reminders, idle detection, break reminders,
goal progress, and daily summaries.

Examples:
  humantime webhook add discord https://discord.com/api/webhooks/...
  humantime webhook add slack https://hooks.slack.com/services/...
  humantime webhook list
  humantime webhook test discord
  humantime webhook disable slack
  humantime webhook remove discord`,
	RunE: runWebhookList,
}

// webhookAddCmd adds a new webhook.
var webhookAddCmd = &cobra.Command{
	Use:   "add NAME URL",
	Short: "Add a new webhook",
	Long: `Add a webhook for receiving notifications.

The webhook type is auto-detected from the URL:
  - Discord: discord.com/api/webhooks/...
  - Slack:   hooks.slack.com/services/...
  - Teams:   outlook.office.com/webhook/...
  - Generic: Any other URL

Examples:
  humantime webhook add discord https://discord.com/api/webhooks/123/abc
  humantime webhook add my-webhook https://example.com/hook --type generic`,
	Args: cobra.ExactArgs(2),
	RunE: runWebhookAdd,
}

// webhookListCmd lists all webhooks.
var webhookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all webhooks",
	RunE:  runWebhookList,
}

// webhookTestCmd tests a webhook.
var webhookTestCmd = &cobra.Command{
	Use:   "test [NAME]",
	Short: "Test a webhook by sending a test notification",
	Long: `Send a test notification to verify webhook configuration.

Examples:
  humantime webhook test discord
  humantime webhook test --all`,
	RunE: runWebhookTest,
}

// webhookRemoveCmd removes a webhook.
var webhookRemoveCmd = &cobra.Command{
	Use:     "remove NAME",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a webhook",
	Args:    cobra.ExactArgs(1),
	RunE:    runWebhookRemove,
}

// webhookEnableCmd enables a webhook.
var webhookEnableCmd = &cobra.Command{
	Use:   "enable NAME",
	Short: "Enable a webhook",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhookEnable,
}

// webhookDisableCmd disables a webhook.
var webhookDisableCmd = &cobra.Command{
	Use:   "disable NAME",
	Short: "Disable a webhook",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhookDisable,
}

func init() {
	// Add flags
	webhookAddCmd.Flags().StringVarP(&webhookAddFlagType, "type", "t", "",
		"Webhook type: discord, slack, teams, generic (auto-detected from URL if not specified)")
	webhookAddCmd.Flags().StringVar(&webhookAddFlagTemplate, "template", "",
		"Custom payload template (required for generic type with custom format)")

	webhookRemoveCmd.Flags().BoolVar(&webhookRemoveFlagForce, "force", false,
		"Skip confirmation")

	webhookTestCmd.Flags().BoolVarP(&webhookTestFlagAll, "all", "a", false,
		"Test all enabled webhooks")

	// Dynamic completion for webhook names
	webhookTestCmd.ValidArgsFunction = completeWebhookArgs
	webhookRemoveCmd.ValidArgsFunction = completeWebhookArgs
	webhookEnableCmd.ValidArgsFunction = completeWebhookArgs
	webhookDisableCmd.ValidArgsFunction = completeWebhookArgs

	// Add subcommands
	webhookCmd.AddCommand(webhookAddCmd)
	webhookCmd.AddCommand(webhookListCmd)
	webhookCmd.AddCommand(webhookTestCmd)
	webhookCmd.AddCommand(webhookRemoveCmd)
	webhookCmd.AddCommand(webhookEnableCmd)
	webhookCmd.AddCommand(webhookDisableCmd)

	rootCmd.AddCommand(webhookCmd)
}

// completeWebhookArgs provides completion for webhook names.
func completeWebhookArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Initialize context for completion
	if ctx == nil {
		opts := runtime.DefaultOptions()
		var err error
		ctx, err = runtime.New(opts)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		defer ctx.Close()
	}

	webhooks, err := ctx.WebhookRepo.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var names []string
	for _, wh := range webhooks {
		if strings.HasPrefix(wh.Name, toComplete) {
			names = append(names, wh.Name)
		}
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// runWebhookAdd handles the webhook add command.
func runWebhookAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	webhookURL := args[1]

	// Validate name
	if !model.IsValidWebhookName(name) {
		return fmt.Errorf("invalid webhook name: must be alphanumeric with dash/underscore, max 50 chars")
	}

	// Validate URL
	if _, err := url.Parse(webhookURL); err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}

	// Check if webhook already exists
	exists, err := ctx.WebhookRepo.Exists(name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("webhook %q already exists", name)
	}

	// Determine type
	webhookType := webhookAddFlagType
	if webhookType == "" {
		webhookType = model.DetectWebhookType(webhookURL)
	}

	if !model.IsValidWebhookType(webhookType) {
		return fmt.Errorf("invalid webhook type: must be discord, slack, teams, or generic")
	}

	// Create webhook
	webhook := model.NewWebhook(name, webhookType, webhookURL)
	if webhookAddFlagTemplate != "" {
		webhook.Template = webhookAddFlagTemplate
	}

	if err := ctx.WebhookRepo.Create(webhook); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"name":       webhook.Name,
			"type":       webhook.Type,
			"url":        webhook.MaskedURL(),
			"enabled":    webhook.Enabled,
			"created_at": webhook.CreatedAt,
		})
	}

	ctx.Formatter.Println("Added webhook:", name)
	ctx.Formatter.Printf("  Type: %s\n", webhook.Type)
	ctx.Formatter.Printf("  URL: %s\n", webhook.MaskedURL())
	ctx.Formatter.Printf("  Status: enabled\n")
	ctx.Formatter.Println("")
	ctx.Formatter.Printf("Test with: humantime webhook test %s\n", name)

	return nil
}

// runWebhookList handles the webhook list command.
func runWebhookList(cmd *cobra.Command, args []string) error {
	webhooks, err := ctx.WebhookRepo.List()
	if err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"webhooks": webhooks,
			"count":    len(webhooks),
		})
	}

	if len(webhooks) == 0 {
		ctx.Formatter.Println("No webhooks configured.")
		ctx.Formatter.Println("")
		ctx.Formatter.Println("Add one with: humantime webhook add discord <url>")
		return nil
	}

	ctx.Formatter.Println("Configured Webhooks:")
	ctx.Formatter.Println("")

	// Header
	ctx.Formatter.Printf("  %-12s %-10s %-10s %s\n", "Name", "Type", "Status", "Last Used")
	ctx.Formatter.Println("  " + strings.Repeat("-", 50))

	for _, wh := range webhooks {
		status := "enabled"
		if !wh.Enabled {
			status = "disabled"
		}

		lastUsed := "never"
		if !wh.LastUsed.IsZero() {
			lastUsed = formatTimeAgo(wh.LastUsed)
		}

		ctx.Formatter.Printf("  %-12s %-10s %-10s %s\n", wh.Name, wh.Type, status, lastUsed)
	}

	ctx.Formatter.Println("")
	ctx.Formatter.Printf("%d webhooks\n", len(webhooks))

	return nil
}

// runWebhookTest handles the webhook test command.
func runWebhookTest(cmd *cobra.Command, args []string) error {
	dispatcher := notify.NewDispatcher(ctx.WebhookRepo)
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if webhookTestFlagAll {
		// Test all enabled webhooks
		webhooks, err := ctx.WebhookRepo.ListEnabled()
		if err != nil {
			return err
		}

		if len(webhooks) == 0 {
			return fmt.Errorf("no enabled webhooks to test")
		}

		var results []notify.DispatchResult
		for _, wh := range webhooks {
			result := dispatcher.TestWebhook(c, wh.Name)
			results = append(results, result)
		}

		if ctx.IsJSON() {
			return ctx.Formatter.PrintJSON(map[string]interface{}{
				"results": results,
			})
		}

		for _, result := range results {
			if result.Success {
				ctx.Formatter.Printf("✓ %s: Success (%dms)\n", result.WebhookName, result.Duration.Milliseconds())
			} else {
				ctx.Formatter.Printf("✗ %s: Failed - %s\n", result.WebhookName, result.Error)
			}
		}

		return nil
	}

	// Test single webhook
	if len(args) == 0 {
		return fmt.Errorf("webhook name required (or use --all)")
	}

	name := args[0]

	ctx.Formatter.Printf("Testing webhook: %s\n", name)
	ctx.Formatter.Println("Sending test notification...")

	result := dispatcher.TestWebhook(c, name)

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"webhook":     name,
			"success":     result.Success,
			"status_code": result.StatusCode,
			"duration_ms": result.Duration.Milliseconds(),
			"error":       errorString(result.Error),
		})
	}

	if result.Success {
		ctx.Formatter.Printf("✓ Success! Message delivered in %dms\n", result.Duration.Milliseconds())
		ctx.Formatter.Println("")
		ctx.Formatter.Println("Check your notification channel for the test message.")
	} else {
		ctx.Formatter.Printf("✗ Failed: %s\n", result.Error)
		ctx.Formatter.Println("")
		ctx.Formatter.Println("The webhook URL may be invalid or the service may be unavailable.")
	}

	return nil
}

// runWebhookRemove handles the webhook remove command.
func runWebhookRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if exists
	exists, err := ctx.WebhookRepo.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("webhook %q not found", name)
	}

	// Confirmation (skip if --force)
	if !webhookRemoveFlagForce && !ctx.IsJSON() {
		ctx.Formatter.Printf("Remove webhook %q? [y/N] ", name)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			ctx.Formatter.Println("Cancelled.")
			return nil
		}
	}

	if err := ctx.WebhookRepo.Delete(name); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"status":  "removed",
			"webhook": name,
		})
	}

	ctx.Formatter.Printf("Removed webhook: %s\n", name)
	return nil
}

// runWebhookEnable handles the webhook enable command.
func runWebhookEnable(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := ctx.WebhookRepo.Enable(name); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"status":  "enabled",
			"webhook": name,
		})
	}

	ctx.Formatter.Printf("Enabled webhook: %s\n", name)
	return nil
}

// runWebhookDisable handles the webhook disable command.
func runWebhookDisable(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := ctx.WebhookRepo.Disable(name); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"status":   "disabled",
			"webhook":  name,
		})
	}

	ctx.Formatter.Printf("Disabled webhook: %s\n", name)
	return nil
}

// formatTimeAgo formats a time as a human-readable relative time.
func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 48*time.Hour:
		return "yesterday"
	default:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	}
}

// errorString returns the error message or empty string if nil.
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
