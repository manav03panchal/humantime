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

// completeStartArgs handles completion for the start command's natural language syntax.
func completeStartArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// If no args yet, suggest project names
	if len(args) == 0 {
		return completeProjects(cmd, args, toComplete)
	}

	// After project, suggest note
	if len(args) == 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

// completeBlocksArgs handles completion for the blocks/list command.
func completeBlocksArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Suggest time ranges
	timeRanges := []string{
		"today\ttoday's blocks",
		"yesterday\tyesterday's blocks",
		"week\tthis week",
		"month\tthis month",
	}

	var filtered []string
	for _, r := range timeRanges {
		if strings.HasPrefix(strings.Split(r, "\t")[0], toComplete) {
			filtered = append(filtered, r)
		}
	}

	// Also suggest projects
	projects := completeProjectsForArg(toComplete)
	filtered = append(filtered, projects...)

	return filtered, cobra.ShellCompDirectiveNoFileComp
}

// completeProjectsForArg returns project completions for a given prefix.
func completeProjectsForArg(toComplete string) []string {
	if ctx == nil || ctx.ProjectRepo == nil {
		return nil
	}

	projects, err := ctx.ProjectRepo.List()
	if err != nil {
		return nil
	}

	var completions []string
	for _, p := range projects {
		if strings.HasPrefix(p.SID, toComplete) {
			completions = append(completions, p.SID+"\t"+p.DisplayName)
		}
	}
	return completions
}

// completeProjectArgs handles completion for commands that take a project SID.
func completeProjectArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete first argument
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeProjects(cmd, args, toComplete)
}
