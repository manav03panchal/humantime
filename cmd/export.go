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
	exportFlagFrom    string
	exportFlagUntil   string
	exportFlagFormat  string
	exportFlagBackup  bool
	exportFlagOutput  string
)

// exportCmd represents the export command.
var exportCmd = &cobra.Command{
	Use:     "export [PROJECT] [TIMEFRAME]",
	Aliases: []string{"ex", "x", "dump"},
	Short:   "Export time data",
	Long: `Export time blocks in various formats.

Examples:
  ht export
  ht export clientwork
  ht export --from "last month"
  ht export --format csv -o report.csv
  ht export --backup -o backup.json`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportFlagProject, "project", "p", "", "Filter by project SID")
	exportCmd.Flags().StringVar(&exportFlagFrom, "from", "", "Start of time range")
	exportCmd.Flags().StringVar(&exportFlagUntil, "until", "", "End of time range")
	exportCmd.Flags().StringVarP(&exportFlagFormat, "format", "F", "json", "Output format: json, csv")
	exportCmd.Flags().BoolVarP(&exportFlagBackup, "backup", "b", false, "Full database backup")
	exportCmd.Flags().StringVarP(&exportFlagOutput, "output", "o", "", "Output file (stdout if omitted)")

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
	parsed.Merge(exportFlagProject, "", "", exportFlagFrom, exportFlagUntil)
	if err := parsed.Process(); err != nil {
		return err
	}

	// Build filter
	filter := storage.BlockFilter{
		ProjectSID: parsed.ProjectSID,
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
	data := struct {
		Version    string                `json:"version"`
		ExportedAt string                `json:"exported_at"`
		Blocks     []*output.BlockOutput `json:"blocks"`
		Count      int                   `json:"count"`
	}{
		Version:    "2",
		ExportedAt: time.Now().Format(time.RFC3339),
		Blocks:     make([]*output.BlockOutput, len(blocks)),
		Count:      len(blocks),
	}

	for i, b := range blocks {
		data.Blocks[i] = convertBlockToOutput(b)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func exportCSV(w *os.File, blocks []*model.Block) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{
		"date", "project", "start", "end", "duration_hours", "note", "tags",
	}); err != nil {
		return err
	}

	// Write rows
	for _, b := range blocks {
		endStr := ""
		if !b.TimestampEnd.IsZero() {
			endStr = b.TimestampEnd.Format("15:04")
		}

		duration := b.Duration().Hours()
		tags := ""
		if len(b.Tags) > 0 {
			for i, t := range b.Tags {
				if i > 0 {
					tags += ","
				}
				tags += t
			}
		}

		if err := writer.Write([]string{
			b.TimestampStart.Format("2006-01-02"),
			b.ProjectSID,
			b.TimestampStart.Format("15:04"),
			endStr,
			strconv.FormatFloat(duration, 'f', 2, 64),
			b.Note,
			tags,
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

	blocks, err := ctx.BlockRepo.List()
	if err != nil {
		return err
	}

	activeBlock, err := ctx.ActiveBlockRepo.Get()
	if err != nil {
		return err
	}

	// Build backup
	backup := struct {
		Version     string             `json:"version"`
		ExportedAt  string             `json:"exported_at"`
		Projects    []*model.Project   `json:"projects"`
		Blocks      []*model.Block     `json:"blocks"`
		ActiveBlock *model.ActiveBlock `json:"active_block"`
	}{
		Version:     "2",
		ExportedAt:  time.Now().Format(time.RFC3339),
		Projects:    projects,
		Blocks:      blocks,
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
		cli.Printf("  Blocks: %d\n", len(blocks))
	}

	return nil
}

func convertBlockToOutput(b *model.Block) *output.BlockOutput {
	out := &output.BlockOutput{
		Key:             b.Key,
		ProjectSID:      b.ProjectSID,
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
