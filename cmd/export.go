package cmd

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/storage"
)

// Export command flags.
var (
	exportFlagProject string
	exportFlagTask    string
	exportFlagFrom    string
	exportFlagUntil   string
	exportFlagFormat  string
	exportFlagBackup  bool
	exportFlagOutput  string
)

// exportCmd represents the export command.
var exportCmd = &cobra.Command{
	Use:     "export [on PROJECT[/TASK]] [TIMEFRAME]",
	Aliases: []string{"ex", "x", "dump"},
	Short:   "Export time data",
	Long: `Export time blocks in various formats. Can export filtered blocks or
create a full database backup.

Examples:
  humantime export
  humantime export on clientwork
  humantime export on clientwork from last month
  humantime export --format csv -o report.csv
  humantime export --backup -o backup.json`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportFlagProject, "project", "p", "", "Filter by project SID")
	exportCmd.Flags().StringVarP(&exportFlagTask, "task", "t", "", "Filter by task SID")
	exportCmd.Flags().StringVar(&exportFlagFrom, "from", "", "Start of time range")
	exportCmd.Flags().StringVar(&exportFlagUntil, "until", "", "End of time range")
	exportCmd.Flags().StringVarP(&exportFlagFormat, "format", "F", "json", "Output format: json, csv")
	exportCmd.Flags().BoolVarP(&exportFlagBackup, "backup", "b", false, "Full database backup")
	exportCmd.Flags().StringVarP(&exportFlagOutput, "output", "o", "", "Output file (stdout if omitted)")

	// Dynamic completion for projects/tasks
	exportCmd.ValidArgsFunction = completeBlocksArgs
	exportCmd.RegisterFlagCompletionFunc("project", completeProjects)

	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	// Handle backup mode
	if exportFlagBackup {
		return runBackup()
	}

	// Parse arguments
	parsed := parser.Parse(args)
	parsed.Merge(exportFlagProject, exportFlagTask, "", exportFlagFrom, exportFlagUntil)
	if err := parsed.Process(); err != nil {
		return err
	}

	// Build filter
	filter := storage.BlockFilter{
		ProjectSID: parsed.ProjectSID,
		TaskSID:    parsed.TaskSID,
	}

	if parsed.HasStart {
		filter.StartAfter = parsed.TimestampStart
	}
	if parsed.HasEnd {
		filter.EndBefore = parsed.TimestampEnd
	}

	// Get blocks
	blocks, err := ctx.BlockRepo.ListFiltered(filter)
	if err != nil {
		return err
	}

	// Determine output destination
	var writer *os.File
	if exportFlagOutput != "" {
		f, err := os.Create(exportFlagOutput)
		if err != nil {
			return err
		}
		defer f.Close()
		writer = f
	} else {
		writer = os.Stdout
	}

	// Export based on format
	switch exportFlagFormat {
	case "csv":
		return exportCSV(writer, blocks)
	default:
		return exportJSON(writer, blocks)
	}
}

func exportJSON(w *os.File, blocks []*model.Block) error {
	output := struct {
		Version    string              `json:"version"`
		ExportedAt string              `json:"exported_at"`
		Blocks     []*output.BlockOutput `json:"blocks"`
		Count      int                 `json:"count"`
	}{
		Version:    "1",
		ExportedAt: time.Now().Format(time.RFC3339),
		Blocks:     make([]*output.BlockOutput, len(blocks)),
		Count:      len(blocks),
	}

	for i, b := range blocks {
		output.Blocks[i] = convertBlockToOutput(b)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func exportCSV(w *os.File, blocks []*model.Block) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{
		"key", "project", "task", "note", "start", "end", "duration_seconds",
	}); err != nil {
		return err
	}

	// Write rows
	for _, b := range blocks {
		endStr := ""
		if !b.TimestampEnd.IsZero() {
			endStr = b.TimestampEnd.Format(time.RFC3339)
		}

		if err := writer.Write([]string{
			b.Key,
			b.ProjectSID,
			b.TaskSID,
			b.Note,
			b.TimestampStart.Format(time.RFC3339),
			endStr,
			formatInt(b.DurationSeconds()),
		}); err != nil {
			return err
		}
	}

	return nil
}

func runBackup() error {
	// Get all data
	projects, err := ctx.ProjectRepo.List()
	if err != nil {
		return err
	}

	tasks, err := ctx.TaskRepo.List()
	if err != nil {
		return err
	}

	blocks, err := ctx.BlockRepo.List()
	if err != nil {
		return err
	}

	goals, err := ctx.GoalRepo.List()
	if err != nil {
		return err
	}

	activeBlock, err := ctx.ActiveBlockRepo.Get()
	if err != nil {
		return err
	}

	// Build backup
	backup := struct {
		Version     string              `json:"version"`
		ExportedAt  string              `json:"exported_at"`
		Config      *model.Config       `json:"config"`
		Projects    []*model.Project    `json:"projects"`
		Tasks       []*model.Task       `json:"tasks"`
		Blocks      []*model.Block      `json:"blocks"`
		Goals       []*model.Goal       `json:"goals"`
		ActiveBlock *model.ActiveBlock  `json:"active_block"`
	}{
		Version:     "1",
		ExportedAt:  time.Now().Format(time.RFC3339),
		Config:      ctx.Config,
		Projects:    projects,
		Tasks:       tasks,
		Blocks:      blocks,
		Goals:       goals,
		ActiveBlock: activeBlock,
	}

	// Determine output destination
	var writer *os.File
	if exportFlagOutput != "" {
		f, err := os.Create(exportFlagOutput)
		if err != nil {
			return err
		}
		defer f.Close()
		writer = f
	} else {
		writer = os.Stdout
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(backup); err != nil {
		return err
	}

	// Print summary if writing to file
	if exportFlagOutput != "" && !ctx.IsJSON() {
		cli := ctx.CLIFormatter()
		cli.Success("Backup created: " + exportFlagOutput)
		cli.Printf("  Projects: %d\n", len(projects))
		cli.Printf("  Tasks: %d\n", len(tasks))
		cli.Printf("  Blocks: %d\n", len(blocks))
		cli.Printf("  Goals: %d\n", len(goals))
	}

	return nil
}

func convertBlockToOutput(b *model.Block) *output.BlockOutput {
	out := &output.BlockOutput{
		Key:             b.Key,
		ProjectSID:      b.ProjectSID,
		TaskSID:         b.TaskSID,
		Note:            b.Note,
		TimestampStart:  b.TimestampStart.Format(time.RFC3339),
		DurationSeconds: b.DurationSeconds(),
		IsActive:        b.IsActive(),
	}
	if !b.TimestampEnd.IsZero() {
		out.TimestampEnd = b.TimestampEnd.Format(time.RFC3339)
	}
	return out
}

func formatInt(n int64) string {
	return strconv.FormatInt(n, 10)
}
