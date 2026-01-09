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
