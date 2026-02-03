package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/runtime"
	"github.com/manav03panchal/humantime/internal/storage"
)

// projectCmd represents the project command.
var projectCmd = &cobra.Command{
	Use:     "project [PROJECT_SID]",
	Aliases: []string{"projects", "proj", "prj", "pj"},
	Short:   "Manage projects",
	Long: `List all projects, show details for a specific project, or manage projects.

Examples:
  ht projects
  ht project clientwork
  ht project new "Client Work"
  ht project archive clientwork`,
	RunE: runProjectList,
}

// Project subcommand flags.
var (
	projectCreateFlagSID   string
	projectCreateFlagColor string
	projectEditFlagName    string
	projectEditFlagColor   string
	projectDeleteFlagForce bool
)

// projectCreateCmd creates a new project.
var projectCreateCmd = &cobra.Command{
	Use:   "new NAME",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectCreate,
}

// projectEditCmd edits an existing project.
var projectEditCmd = &cobra.Command{
	Use:   "edit PROJECT_SID",
	Short: "Edit a project",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectEdit,
}

// projectDeleteCmd deletes a project.
var projectDeleteCmd = &cobra.Command{
	Use:   "archive PROJECT_SID",
	Short: "Archive a project (soft delete)",
	Long: `Archive a project. Archived projects are hidden from lists but data is preserved.

Examples:
  ht project archive myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectArchive,
}

func init() {
	// Create flags
	projectCreateCmd.Flags().StringVarP(&projectCreateFlagSID, "sid", "s", "", "Custom SID (auto-generated if omitted)")
	projectCreateCmd.Flags().StringVarP(&projectCreateFlagColor, "color", "c", "", "Hex color (#RRGGBB)")

	// Edit flags
	projectEditCmd.Flags().StringVarP(&projectEditFlagName, "name", "n", "", "Update display name")
	projectEditCmd.Flags().StringVarP(&projectEditFlagColor, "color", "c", "", "Update color")

	// Archive flags
	projectDeleteCmd.Flags().BoolVar(&projectDeleteFlagForce, "force", false, "Skip confirmation prompt")

	// Dynamic completion for projects
	projectCmd.ValidArgsFunction = completeProjectArgs
	projectEditCmd.ValidArgsFunction = completeProjectArgs
	projectDeleteCmd.ValidArgsFunction = completeProjectArgs

	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectEditCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	rootCmd.AddCommand(projectCmd)
}

func runProjectList(cmd *cobra.Command, args []string) error {
	// If a project SID is provided, show that project
	if len(args) > 0 {
		return showProject(args[0])
	}

	// List all projects
	projects, err := ctx.ProjectRepo.List()
	if err != nil {
		return err
	}

	// Filter out archived projects
	activeProjects := make([]*model.Project, 0)
	for _, p := range projects {
		if !p.Archived {
			activeProjects = append(activeProjects, p)
		}
	}

	// Calculate durations for each project
	blocks, err := ctx.BlockRepo.List()
	if err != nil {
		return err
	}
	projectDurations := make(map[string]int64)
	for _, b := range blocks {
		projectDurations[b.ProjectSID] += b.DurationSeconds()
	}

	if ctx.IsJSON() {
		outputs := make([]*output.ProjectOutput, len(activeProjects))
		for i, p := range activeProjects {
			outputs[i] = output.NewProjectOutput(p, secondsToDuration(projectDurations[p.SID]))
		}
		return ctx.Formatter.JSON(output.ProjectsResponse{Projects: outputs})
	}

	return printProjectsCLI(activeProjects, projectDurations)
}

func showProject(sid string) error {
	project, err := ctx.ProjectRepo.Get(sid)
	if err != nil {
		if storage.IsErrKeyNotFound(err) {
			return runtime.ErrProjectNotFound
		}
		return err
	}

	// Get blocks for this project
	blocks, err := ctx.BlockRepo.ListByProject(sid)
	if err != nil {
		return err
	}

	// Calculate total duration
	var totalDuration int64
	for _, b := range blocks {
		totalDuration += b.DurationSeconds()
	}

	if ctx.IsJSON() {
		out := output.NewProjectOutput(project, secondsToDuration(totalDuration))
		return ctx.Formatter.JSON(out)
	}

	cli := ctx.CLIFormatter()
	cli.Title("Project: " + project.SID)
	cli.Printf("  Display Name: %s\n", project.DisplayName)
	if project.Color != "" {
		cli.Printf("  Color: %s\n", project.Color)
	}
	cli.Printf("  Total Time: %s\n", cli.Duration(output.FormatDuration(secondsToDuration(totalDuration))))
	cli.Printf("  Blocks: %d\n", len(blocks))
	cli.Println("")

	if len(blocks) > 0 {
		cli.Println("Recent Blocks:")
		limit := 5
		if len(blocks) < limit {
			limit = len(blocks)
		}
		for i := 0; i < limit; i++ {
			b := blocks[i]
			cli.Printf("  %s  %s",
				output.FormatDate(b.TimestampStart),
				cli.Duration(output.FormatDuration(b.Duration())))
			if b.Note != "" {
				cli.Printf("  %s", cli.Note(b.Note))
			}
			cli.Println("")
		}
	}

	return nil
}

func runProjectCreate(cmd *cobra.Command, args []string) error {
	displayName := args[0]

	// Generate or use provided SID
	sid := projectCreateFlagSID
	if sid == "" {
		sid = parser.ConvertToSID(displayName)
	}

	// Validate SID
	if !parser.ValidateSID(sid) {
		return runtime.ErrInvalidSID
	}

	// Validate color
	if projectCreateFlagColor != "" && !model.ValidateColor(projectCreateFlagColor) {
		return runtime.ErrInvalidColor
	}

	// Check if project exists
	exists, err := ctx.ProjectRepo.Exists(sid)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("project '%s' already exists", sid)
	}

	// Create project
	project := model.NewProject(sid, displayName, projectCreateFlagColor)
	if err := ctx.ProjectRepo.Create(project); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(output.NewProjectOutput(project, 0))
	}

	cli := ctx.CLIFormatter()
	cli.Success("Created project: " + sid)
	cli.Printf("  Display Name: %s\n", displayName)
	if project.Color != "" {
		cli.Printf("  Color: %s\n", project.Color)
	}

	return nil
}

func runProjectEdit(cmd *cobra.Command, args []string) error {
	sid := args[0]

	// Get existing project
	project, err := ctx.ProjectRepo.Get(sid)
	if err != nil {
		if storage.IsErrKeyNotFound(err) {
			return runtime.ErrProjectNotFound
		}
		return err
	}

	// Apply updates
	updated := false

	if projectEditFlagName != "" {
		project.DisplayName = projectEditFlagName
		updated = true
	}

	if projectEditFlagColor != "" {
		if !model.ValidateColor(projectEditFlagColor) {
			return runtime.ErrInvalidColor
		}
		project.Color = projectEditFlagColor
		updated = true
	}

	if !updated {
		return fmt.Errorf("no updates specified (use --name or --color)")
	}

	// Save
	if err := ctx.ProjectRepo.Update(project); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(output.NewProjectOutput(project, 0))
	}

	cli := ctx.CLIFormatter()
	cli.Success("Updated project: " + sid)
	cli.Printf("  Display Name: %s\n", project.DisplayName)
	if project.Color != "" {
		cli.Printf("  Color: %s\n", project.Color)
	}

	return nil
}

func runProjectArchive(cmd *cobra.Command, args []string) error {
	sid := args[0]

	// Verify project exists
	project, err := ctx.ProjectRepo.Get(sid)
	if err != nil {
		if storage.IsErrKeyNotFound(err) {
			return runtime.ErrProjectNotFound
		}
		return err
	}

	// Count associated data
	blocks, _ := ctx.BlockRepo.ListByProject(sid)

	// Calculate total tracked time
	var totalSeconds int64
	for _, b := range blocks {
		totalSeconds += b.DurationSeconds()
	}

	cli := ctx.CLIFormatter()

	// Show what will be archived
	if !projectDeleteFlagForce {
		cli.Warning(fmt.Sprintf("About to archive project '%s'", sid))
		cli.Println("")
		cli.Printf("  Display Name: %s\n", project.DisplayName)
		cli.Printf("  Total Time:   %s\n", output.FormatDuration(secondsToDuration(totalSeconds)))
		cli.Printf("  Blocks:       %d\n", len(blocks))
		cli.Println("")
		cli.Muted("Data will be preserved but project hidden from lists.")
		cli.Println("")

		confirmed, err := promptConfirmation("Are you sure? (y/N): ")
		if err != nil || !confirmed {
			cli.Muted("Aborted.")
			return nil
		}
	}

	// Archive the project
	project.Archived = true
	if err := ctx.ProjectRepo.Update(project); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(map[string]any{
			"status":      "archived",
			"project_sid": sid,
			"blocks":      len(blocks),
		})
	}

	cli.Success(fmt.Sprintf("Archived project '%s'", sid))
	return nil
}

func printProjectsCLI(projects []*model.Project, durations map[string]int64) error {
	cli := ctx.CLIFormatter()

	if len(projects) == 0 {
		cli.Muted("No projects found.")
		cli.Muted("Use 'ht start <project>' to create one.")
		return nil
	}

	cli.Title(fmt.Sprintf("Projects (%d)", len(projects)))
	cli.Println("")

	// Calculate max widths
	maxSIDLen := 0
	maxDurationLen := 0
	for _, p := range projects {
		if len(p.SID) > maxSIDLen {
			maxSIDLen = len(p.SID)
		}
		dur := output.FormatDuration(secondsToDuration(durations[p.SID]))
		if len(dur) > maxDurationLen {
			maxDurationLen = len(dur)
		}
	}
	if maxSIDLen < 12 {
		maxSIDLen = 12
	}
	if maxDurationLen < 8 {
		maxDurationLen = 8
	}

	var totalDuration int64
	for _, p := range projects {
		dur := durations[p.SID]
		totalDuration += dur
		padding := maxSIDLen - len(p.SID)
		cli.Printf("  %s%s  %*s", cli.ProjectName(p.SID), strings.Repeat(" ", padding), maxDurationLen, cli.Duration(output.FormatDuration(secondsToDuration(dur))))
		if p.DisplayName != p.SID {
			cli.Printf("  %s", p.DisplayName)
		}
		cli.Println("")
	}

	cli.Println("")
	cli.Println(strings.Repeat("─", maxSIDLen+maxDurationLen+6))
	cli.Printf("  %-*s  %*s\n", maxSIDLen, "Total:", maxDurationLen, cli.Duration(output.FormatDuration(secondsToDuration(totalDuration))))

	return nil
}

func secondsToDuration(seconds int64) time.Duration {
	return time.Duration(seconds) * time.Second
}

// promptConfirmation is defined in blocks.go
