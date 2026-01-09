package cmd

import (
	"fmt"
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
  humantime project
  humantime project clientwork
  humantime project create "Client Work" --color "#FF5733"
  humantime project edit clientwork --name "New Name"`,
	RunE: runProjectList,
}

// Project subcommand flags.
var (
	projectCreateFlagSID   string
	projectCreateFlagColor string
	projectEditFlagName    string
	projectEditFlagColor   string
)

// projectCreateCmd creates a new project.
var projectCreateCmd = &cobra.Command{
	Use:   "create NAME",
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

func init() {
	// Create flags
	projectCreateCmd.Flags().StringVarP(&projectCreateFlagSID, "sid", "s", "", "Custom SID (auto-generated if omitted)")
	projectCreateCmd.Flags().StringVarP(&projectCreateFlagColor, "color", "c", "", "Hex color (#RRGGBB)")

	// Edit flags
	projectEditCmd.Flags().StringVarP(&projectEditFlagName, "name", "n", "", "Update display name")
	projectEditCmd.Flags().StringVarP(&projectEditFlagColor, "color", "c", "", "Update color")

	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectEditCmd)
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
		outputs := make([]*output.ProjectOutput, len(projects))
		for i, p := range projects {
			outputs[i] = output.NewProjectOutput(p, secondsToDuration(projectDurations[p.SID]))
		}
		return ctx.Formatter.JSON(output.ProjectsResponse{Projects: outputs})
	}

	return printProjectsCLI(projects, projectDurations)
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

	// Get tasks for this project
	tasks, err := ctx.TaskRepo.ListByProject(sid)
	if err != nil {
		return err
	}

	// Calculate task durations
	taskDurations := make(map[string]int64)
	var totalDuration int64
	for _, b := range blocks {
		taskDurations[b.TaskSID] += b.DurationSeconds()
		totalDuration += b.DurationSeconds()
	}

	if ctx.IsJSON() {
		out := output.NewProjectOutput(project, secondsToDuration(totalDuration))
		out.Tasks = make([]*output.TaskOutput, len(tasks))
		for i, t := range tasks {
			out.Tasks[i] = output.NewTaskOutput(t, secondsToDuration(taskDurations[t.SID]))
		}
		return ctx.Formatter.JSON(out)
	}

	cli := ctx.CLIFormatter()
	cli.Title("Project: " + project.SID)
	cli.Printf("  Display Name: %s\n", project.DisplayName)
	if project.Color != "" {
		cli.Printf("  Color: %s\n", project.Color)
	}
	cli.Printf("  Total Time: %s\n", cli.Duration(output.FormatDuration(secondsToDuration(totalDuration))))
	cli.Println("")

	if len(tasks) > 0 {
		cli.Println("Tasks:")
		for _, t := range tasks {
			dur := taskDurations[t.SID]
			cli.Printf("  • %s  %s\n", cli.TaskName(t.SID), cli.Duration(output.FormatDuration(secondsToDuration(dur))))
		}
		cli.Println("")
	}

	if len(blocks) > 0 {
		cli.Println("Recent Blocks:")
		limit := 5
		if len(blocks) < limit {
			limit = len(blocks)
		}
		for i := 0; i < limit; i++ {
			b := blocks[i]
			cli.Printf("  %s  %s  %s",
				output.FormatDate(b.TimestampStart),
				cli.TaskName(b.TaskSID),
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

func printProjectsCLI(projects []*model.Project, durations map[string]int64) error {
	cli := ctx.CLIFormatter()

	if len(projects) == 0 {
		cli.Muted("No projects found.")
		cli.Muted("Use 'humantime start on <project>' to create one.")
		return nil
	}

	cli.Title(fmt.Sprintf("Projects (%d)", len(projects)))
	cli.Println("")

	var totalDuration int64
	for _, p := range projects {
		dur := durations[p.SID]
		totalDuration += dur
		cli.Printf("• %s  %s\n", cli.ProjectName(p.SID), cli.Duration(output.FormatDuration(secondsToDuration(dur))))
		if p.DisplayName != p.SID {
			cli.Printf("  %s\n", p.DisplayName)
		}
	}

	cli.Println("")
	cli.Printf("Total tracked: %s\n", cli.Duration(output.FormatDuration(secondsToDuration(totalDuration))))

	return nil
}

func secondsToDuration(seconds int64) time.Duration {
	return time.Duration(seconds) * time.Second
}
