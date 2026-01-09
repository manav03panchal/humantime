package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/runtime"
	"github.com/manav03panchal/humantime/internal/storage"
)

// taskCmd represents the task command.
var taskCmd = &cobra.Command{
	Use:     "task [PROJECT/TASK]",
	Aliases: []string{"tasks", "tsk"},
	Short:   "Manage tasks",
	Long: `List all tasks, show details for a specific task, or manage tasks.

Tasks are typically auto-created when using project/task notation with the start command.
This command provides explicit task management for advanced use cases.

Examples:
  humantime task
  humantime task list clientwork
  humantime task create clientwork "Bug Fix"
  humantime task edit clientwork/bugfix --name "Critical Bug Fix"
  humantime task delete clientwork/bugfix`,
	RunE: runTaskList,
}

// Task subcommand flags.
var (
	taskCreateFlagSID   string
	taskCreateFlagColor string
	taskEditFlagName    string
	taskEditFlagColor   string
)

// taskListCmd lists tasks for a project.
var taskListCmd = &cobra.Command{
	Use:   "list [PROJECT_SID]",
	Short: "List tasks for a project",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTaskListByProject,
}

// taskCreateCmd creates a new task.
var taskCreateCmd = &cobra.Command{
	Use:   "create PROJECT_SID NAME",
	Short: "Create a new task",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskCreate,
}

// taskEditCmd edits an existing task.
var taskEditCmd = &cobra.Command{
	Use:   "edit PROJECT/TASK",
	Short: "Edit a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskEdit,
}

// taskDeleteCmd deletes a task.
var taskDeleteCmd = &cobra.Command{
	Use:   "delete PROJECT/TASK",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskDelete,
}

func init() {
	// Create flags
	taskCreateCmd.Flags().StringVarP(&taskCreateFlagSID, "sid", "s", "", "Custom SID (auto-generated if omitted)")
	taskCreateCmd.Flags().StringVarP(&taskCreateFlagColor, "color", "c", "", "Hex color (#RRGGBB)")

	// Edit flags
	taskEditCmd.Flags().StringVarP(&taskEditFlagName, "name", "n", "", "Update display name")
	taskEditCmd.Flags().StringVarP(&taskEditFlagColor, "color", "c", "", "Update color")

	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskEditCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	rootCmd.AddCommand(taskCmd)
}

func runTaskList(cmd *cobra.Command, args []string) error {
	// If a project/task notation is provided, show that task
	if len(args) > 0 {
		projectSID, taskSID := parser.ParseProjectTask(args[0])
		if taskSID != "" {
			return showTask(projectSID, taskSID)
		}
		// If only project provided, list tasks for that project
		return listTasksByProject(projectSID)
	}

	// List all tasks across all projects
	return listAllTasks()
}

func runTaskListByProject(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return listAllTasks()
	}
	return listTasksByProject(args[0])
}

func listAllTasks() error {
	tasks, err := ctx.TaskRepo.List()
	if err != nil {
		return err
	}

	// Calculate durations for each task
	blocks, err := ctx.BlockRepo.List()
	if err != nil {
		return err
	}
	taskDurations := make(map[string]int64)
	for _, b := range blocks {
		key := b.ProjectSID + "/" + b.TaskSID
		taskDurations[key] += b.DurationSeconds()
	}

	if ctx.IsJSON() {
		return printTasksJSON(tasks, taskDurations)
	}

	return printTasksCLI(tasks, taskDurations, "")
}

func listTasksByProject(projectSID string) error {
	// Verify project exists
	exists, err := ctx.ProjectRepo.Exists(projectSID)
	if err != nil {
		return err
	}
	if !exists {
		return runtime.ErrProjectNotFound
	}

	tasks, err := ctx.TaskRepo.ListByProject(projectSID)
	if err != nil {
		return err
	}

	// Calculate durations for each task
	blocks, err := ctx.BlockRepo.ListByProject(projectSID)
	if err != nil {
		return err
	}
	taskDurations := make(map[string]int64)
	for _, b := range blocks {
		key := b.ProjectSID + "/" + b.TaskSID
		taskDurations[key] += b.DurationSeconds()
	}

	if ctx.IsJSON() {
		return printTasksJSON(tasks, taskDurations)
	}

	return printTasksCLI(tasks, taskDurations, projectSID)
}

func showTask(projectSID, taskSID string) error {
	task, err := ctx.TaskRepo.Get(projectSID, taskSID)
	if err != nil {
		if storage.IsErrKeyNotFound(err) {
			return runtime.ErrTaskNotFound
		}
		return err
	}

	// Get blocks for this task
	blocks, err := ctx.BlockRepo.ListByProject(projectSID)
	if err != nil {
		return err
	}

	// Filter blocks for this task and calculate duration
	var taskBlocks []*model.Block
	var totalDuration int64
	for _, b := range blocks {
		if b.TaskSID == taskSID {
			taskBlocks = append(taskBlocks, b)
			totalDuration += b.DurationSeconds()
		}
	}

	if ctx.IsJSON() {
		out := output.NewTaskOutput(task, secondsToDuration(totalDuration))
		return ctx.Formatter.JSON(out)
	}

	cli := ctx.CLIFormatter()
	cli.Title(fmt.Sprintf("Task: %s/%s", projectSID, taskSID))
	cli.Printf("  Display Name: %s\n", task.DisplayName)
	if task.Color != "" {
		cli.Printf("  Color: %s\n", task.Color)
	}
	cli.Printf("  Total Time: %s\n", cli.Duration(output.FormatDuration(secondsToDuration(totalDuration))))
	cli.Println("")

	if len(taskBlocks) > 0 {
		cli.Println("Recent Blocks:")
		limit := 5
		if len(taskBlocks) < limit {
			limit = len(taskBlocks)
		}
		for i := 0; i < limit; i++ {
			b := taskBlocks[i]
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

func runTaskCreate(cmd *cobra.Command, args []string) error {
	projectSID := args[0]
	displayName := args[1]

	// Verify project exists
	exists, err := ctx.ProjectRepo.Exists(projectSID)
	if err != nil {
		return err
	}
	if !exists {
		return runtime.ErrProjectNotFound
	}

	// Generate or use provided SID
	taskSID := taskCreateFlagSID
	if taskSID == "" {
		taskSID = parser.ConvertToSID(displayName)
	}

	// Validate SID
	if !parser.ValidateSID(taskSID) {
		return runtime.ErrInvalidSID
	}

	// Validate color
	if taskCreateFlagColor != "" && !model.ValidateColor(taskCreateFlagColor) {
		return runtime.ErrInvalidColor
	}

	// Check if task exists
	exists, err = ctx.TaskRepo.Exists(projectSID, taskSID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("task '%s/%s' already exists", projectSID, taskSID)
	}

	// Create task
	task := model.NewTask(projectSID, taskSID, displayName, taskCreateFlagColor)
	if err := ctx.TaskRepo.Create(task); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(output.NewTaskOutput(task, 0))
	}

	cli := ctx.CLIFormatter()
	cli.Success(fmt.Sprintf("Created task: %s/%s", projectSID, taskSID))
	cli.Printf("  Display Name: %s\n", displayName)
	if task.Color != "" {
		cli.Printf("  Color: %s\n", task.Color)
	}

	return nil
}

func runTaskEdit(cmd *cobra.Command, args []string) error {
	projectSID, taskSID := parser.ParseProjectTask(args[0])

	if taskSID == "" {
		return fmt.Errorf("task must be specified as PROJECT/TASK")
	}

	// Get existing task
	task, err := ctx.TaskRepo.Get(projectSID, taskSID)
	if err != nil {
		if storage.IsErrKeyNotFound(err) {
			return runtime.ErrTaskNotFound
		}
		return err
	}

	// Apply updates
	updated := false

	if taskEditFlagName != "" {
		task.DisplayName = taskEditFlagName
		updated = true
	}

	if taskEditFlagColor != "" {
		if !model.ValidateColor(taskEditFlagColor) {
			return runtime.ErrInvalidColor
		}
		task.Color = taskEditFlagColor
		updated = true
	}

	if !updated {
		return fmt.Errorf("no updates specified (use --name or --color)")
	}

	// Save
	if err := ctx.TaskRepo.Update(task); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(output.NewTaskOutput(task, 0))
	}

	cli := ctx.CLIFormatter()
	cli.Success(fmt.Sprintf("Updated task: %s/%s", projectSID, taskSID))
	cli.Printf("  Display Name: %s\n", task.DisplayName)
	if task.Color != "" {
		cli.Printf("  Color: %s\n", task.Color)
	}

	return nil
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	projectSID, taskSID := parser.ParseProjectTask(args[0])

	if taskSID == "" {
		return fmt.Errorf("task must be specified as PROJECT/TASK")
	}

	// Verify task exists
	exists, err := ctx.TaskRepo.Exists(projectSID, taskSID)
	if err != nil {
		return err
	}
	if !exists {
		return runtime.ErrTaskNotFound
	}

	// Delete task
	if err := ctx.TaskRepo.Delete(projectSID, taskSID); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(map[string]string{
			"status":      "deleted",
			"project_sid": projectSID,
			"task_sid":    taskSID,
		})
	}

	cli := ctx.CLIFormatter()
	cli.Success(fmt.Sprintf("Deleted task: %s/%s", projectSID, taskSID))

	return nil
}

func printTasksJSON(tasks []*model.Task, durations map[string]int64) error {
	outputs := make([]*output.TaskOutput, len(tasks))
	for i, t := range tasks {
		key := t.ProjectSID + "/" + t.SID
		outputs[i] = output.NewTaskOutput(t, secondsToDuration(durations[key]))
	}
	return ctx.Formatter.JSON(TasksResponse{Tasks: outputs})
}

// TasksResponse represents the tasks list output in JSON.
type TasksResponse struct {
	Tasks []*output.TaskOutput `json:"tasks"`
}

func printTasksCLI(tasks []*model.Task, durations map[string]int64, projectFilter string) error {
	cli := ctx.CLIFormatter()

	if len(tasks) == 0 {
		if projectFilter != "" {
			cli.Muted(fmt.Sprintf("No tasks found for project '%s'.", projectFilter))
		} else {
			cli.Muted("No tasks found.")
		}
		cli.Muted("Tasks are auto-created when using 'humantime start on project/task'.")
		return nil
	}

	title := "Tasks"
	if projectFilter != "" {
		title = fmt.Sprintf("Tasks for %s", cli.ProjectName(projectFilter))
	}
	cli.Title(fmt.Sprintf("%s (%d)", title, len(tasks)))
	cli.Println("")

	// Group tasks by project if no filter
	if projectFilter == "" {
		tasksByProject := make(map[string][]*model.Task)
		for _, t := range tasks {
			tasksByProject[t.ProjectSID] = append(tasksByProject[t.ProjectSID], t)
		}

		var totalDuration int64
		for projectSID, projectTasks := range tasksByProject {
			cli.Printf("%s\n", cli.ProjectName(projectSID))
			for _, t := range projectTasks {
				key := t.ProjectSID + "/" + t.SID
				dur := durations[key]
				totalDuration += dur
				cli.Printf("  %s %s  %s\n", taskBullet(), cli.TaskName(t.SID), cli.Duration(output.FormatDuration(secondsToDuration(dur))))
				if t.DisplayName != t.SID {
					cli.Printf("      %s\n", t.DisplayName)
				}
			}
			cli.Println("")
		}

		cli.Printf("Total tracked: %s\n", cli.Duration(output.FormatDuration(secondsToDuration(totalDuration))))
	} else {
		var totalDuration int64
		for _, t := range tasks {
			key := t.ProjectSID + "/" + t.SID
			dur := durations[key]
			totalDuration += dur
			cli.Printf("%s %s  %s\n", taskBullet(), cli.TaskName(t.SID), cli.Duration(output.FormatDuration(secondsToDuration(dur))))
			if t.DisplayName != t.SID {
				cli.Printf("    %s\n", t.DisplayName)
			}
		}

		cli.Println("")
		cli.Printf("Total tracked: %s\n", cli.Duration(output.FormatDuration(secondsToDuration(totalDuration))))
	}

	return nil
}

func taskBullet() string {
	return "-"
}
