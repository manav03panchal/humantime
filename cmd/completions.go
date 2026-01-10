package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// completeProjects returns a completion function for project SIDs.
func completeProjects(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if ctx == nil || ctx.ProjectRepo == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	projects, err := ctx.ProjectRepo.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, p := range projects {
		if strings.HasPrefix(p.SID, toComplete) {
			completions = append(completions, p.SID+"\t"+p.DisplayName)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeProjectsAndTasks returns a completion function for project/task notation.
func completeProjectsAndTasks(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if ctx == nil || ctx.ProjectRepo == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string

	// Check if we're completing a task (contains /)
	if strings.Contains(toComplete, "/") {
		parts := strings.SplitN(toComplete, "/", 2)
		projectSID := parts[0]
		taskPrefix := ""
		if len(parts) > 1 {
			taskPrefix = parts[1]
		}

		// Get tasks for this project
		if ctx.TaskRepo != nil {
			tasks, err := ctx.TaskRepo.ListByProject(projectSID)
			if err == nil {
				for _, t := range tasks {
					if strings.HasPrefix(t.SID, taskPrefix) {
						completions = append(completions, projectSID+"/"+t.SID+"\t"+t.DisplayName)
					}
				}
			}
		}
	} else {
		// Complete project names
		projects, err := ctx.ProjectRepo.List()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		for _, p := range projects {
			if strings.HasPrefix(p.SID, toComplete) {
				// Add both the project alone and with / suffix to suggest tasks
				completions = append(completions, p.SID+"\t"+p.DisplayName)
				completions = append(completions, p.SID+"/\ttasks...")
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

// completeStartArgs handles completion for the start command's natural language syntax.
func completeStartArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// If no args yet, suggest "on"
	if len(args) == 0 {
		if strings.HasPrefix("on", toComplete) {
			return []string{"on\tspecify project"}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// If first arg is "on", complete project/task
	if len(args) == 1 && args[0] == "on" {
		return completeProjectsAndTasks(cmd, args, toComplete)
	}

	// After project, suggest "with note" or time expressions
	if len(args) == 2 && args[0] == "on" {
		suggestions := []string{
			"with\tadd a note",
		}
		var filtered []string
		for _, s := range suggestions {
			if strings.HasPrefix(strings.Split(s, "\t")[0], toComplete) {
				filtered = append(filtered, s)
			}
		}
		return filtered, cobra.ShellCompDirectiveNoFileComp
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

// completeBlocksArgs handles completion for the blocks command.
func completeBlocksArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		if strings.HasPrefix("on", toComplete) {
			return []string{"on\tfilter by project"}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	if len(args) == 1 && args[0] == "on" {
		return completeProjectsAndTasks(cmd, args, toComplete)
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

// completeGoalTypes returns goal type completions.
func completeGoalTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{"daily\tdaily goal", "weekly\tweekly goal"}
	var filtered []string
	for _, t := range types {
		if strings.HasPrefix(strings.Split(t, "\t")[0], toComplete) {
			filtered = append(filtered, t)
		}
	}
	return filtered, cobra.ShellCompDirectiveNoFileComp
}

// completeProjectArgs handles completion for commands that take a project SID.
func completeProjectArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete first argument
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeProjects(cmd, args, toComplete)
}

// completeTaskArgs handles completion for commands that take a project/task notation.
func completeTaskArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete first argument
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeProjectsAndTasks(cmd, args, toComplete)
}

// completeReminders returns a completion function for reminder IDs.
func completeReminders(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if ctx == nil || ctx.ReminderRepo == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	reminders, err := ctx.ReminderRepo.ListPending()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, r := range reminders {
		shortID := r.ShortID()
		if strings.HasPrefix(shortID, toComplete) {
			completions = append(completions, shortID+"\t"+r.Title)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeWebhooks returns a completion function for webhook names.
func completeWebhooks(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if ctx == nil || ctx.WebhookRepo == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	webhooks, err := ctx.WebhookRepo.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, w := range webhooks {
		if strings.HasPrefix(w.Name, toComplete) {
			status := "enabled"
			if !w.Enabled {
				status = "disabled"
			}
			completions = append(completions, w.Name+"\t"+string(w.Type)+" ("+status+")")
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeWebhookTypes returns a completion function for webhook types.
func completeWebhookTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{
		"discord\tDiscord webhook",
		"slack\tSlack incoming webhook",
		"teams\tMicrosoft Teams webhook",
		"generic\tGeneric HTTP webhook",
	}

	var filtered []string
	for _, t := range types {
		if strings.HasPrefix(strings.Split(t, "\t")[0], toComplete) {
			filtered = append(filtered, t)
		}
	}
	return filtered, cobra.ShellCompDirectiveNoFileComp
}

// completeDaemonSubcommands returns completions for daemon subcommands.
func completeDaemonSubcommands(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	subcommands := []string{
		"start\tStart the daemon",
		"stop\tStop the daemon",
		"status\tShow daemon status",
		"logs\tView daemon logs",
		"install\tInstall as system service",
		"uninstall\tUninstall system service",
	}

	var filtered []string
	for _, s := range subcommands {
		if strings.HasPrefix(strings.Split(s, "\t")[0], toComplete) {
			filtered = append(filtered, s)
		}
	}
	return filtered, cobra.ShellCompDirectiveNoFileComp
}

// completeConfigKeys returns completion for config keys.
func completeConfigKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	keys := []string{
		"notify\tAll notification settings",
		"notify.idle-after\tIdle detection threshold",
		"notify.break-after\tBreak reminder threshold",
		"notify.break-reset\tBreak reset gap",
		"notify.goal-progress\tGoal progress milestones",
		"notify.daily-summary\tDaily summary time",
		"notify.end-of-day\tEnd of day recap time",
		"notify.idle\tEnable/disable idle notifications",
		"notify.break\tEnable/disable break notifications",
		"notify.goal\tEnable/disable goal notifications",
		"notify.daily_summary\tEnable/disable daily summary",
		"notify.end_of_day\tEnable/disable end of day recap",
		"notify.reminder\tEnable/disable reminders",
	}

	var filtered []string
	for _, k := range keys {
		if strings.HasPrefix(strings.Split(k, "\t")[0], toComplete) {
			filtered = append(filtered, k)
		}
	}
	return filtered, cobra.ShellCompDirectiveNoFileComp
}
