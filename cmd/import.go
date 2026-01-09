package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
)

// Import command flags.
var (
	importFlagDryRun bool
	importFlagForce  bool
)

// importCmd represents the import command.
var importCmd = &cobra.Command{
	Use:     "import FILE",
	Aliases: []string{"imp", "i", "restore"},
	Short:   "Import time data from a file",
	Long: `Import time data from JSON files. Supports Humantime backup format
and Zeit v1 export format.

Examples:
  humantime import backup.json
  humantime import zeit-export.json
  humantime import backup.json --dry-run
  humantime import backup.json --force`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	importCmd.Flags().BoolVar(&importFlagDryRun, "dry-run", false, "Preview import without making changes")
	importCmd.Flags().BoolVar(&importFlagForce, "force", false, "Overwrite existing data on conflicts")

	rootCmd.AddCommand(importCmd)
}

// HumantimeBackup represents a full Humantime backup.
type HumantimeBackup struct {
	Version     string              `json:"version"`
	ExportedAt  string              `json:"exported_at"`
	Config      *model.Config       `json:"config"`
	Projects    []*model.Project    `json:"projects"`
	Tasks       []*model.Task       `json:"tasks"`
	Blocks      []*model.Block      `json:"blocks"`
	Goals       []*model.Goal       `json:"goals"`
	ActiveBlock *model.ActiveBlock  `json:"active_block"`
}

// ZeitEntry represents a Zeit v1 time entry.
type ZeitEntry struct {
	ID      string `json:"id"`
	Begin   string `json:"begin"`
	End     string `json:"end"`
	Project string `json:"project"`
	Task    string `json:"task"`
	Notes   string `json:"notes"`
}

// ZeitExport represents Zeit v1 export format.
type ZeitExport struct {
	Entries []ZeitEntry `json:"entries"`
}

func runImport(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Detect format
	format := detectImportFormat(data)

	cli := ctx.CLIFormatter()

	switch format {
	case "humantime":
		return importHumantime(data, cli)
	case "zeit":
		return importZeit(data, cli)
	default:
		return fmt.Errorf("unrecognized file format")
	}
}

func detectImportFormat(data []byte) string {
	// Try to detect format by parsing
	var backup HumantimeBackup
	if err := json.Unmarshal(data, &backup); err == nil {
		if backup.Version != "" && (backup.Projects != nil || backup.Blocks != nil) {
			return "humantime"
		}
	}

	// Try Zeit format
	var zeit ZeitExport
	if err := json.Unmarshal(data, &zeit); err == nil {
		if len(zeit.Entries) > 0 {
			return "zeit"
		}
	}

	// Try Zeit as array
	var entries []ZeitEntry
	if err := json.Unmarshal(data, &entries); err == nil {
		if len(entries) > 0 && entries[0].Begin != "" {
			return "zeit"
		}
	}

	return "unknown"
}

func importHumantime(data []byte, cli *output.CLIFormatter) error {
	var backup HumantimeBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("failed to parse backup: %w", err)
	}

	// Statistics
	stats := struct {
		Projects   int
		Tasks      int
		Blocks     int
		Goals      int
		Skipped    int
		Duplicates int
	}{}

	if importFlagDryRun {
		cli.Title("Dry Run - Import Preview")
	} else {
		cli.Title("Importing Humantime Backup")
	}

	// Import projects
	for _, p := range backup.Projects {
		if importFlagDryRun {
			stats.Projects++
			continue
		}

		exists, _ := ctx.ProjectRepo.Exists(p.SID)
		if exists && !importFlagForce {
			stats.Duplicates++
			continue
		}

		if exists {
			if err := ctx.ProjectRepo.Update(p); err != nil {
				return fmt.Errorf("failed to update project %s: %w", p.SID, err)
			}
		} else {
			if err := ctx.ProjectRepo.Create(p); err != nil {
				return fmt.Errorf("failed to create project %s: %w", p.SID, err)
			}
		}
		stats.Projects++
	}

	// Import tasks
	for _, t := range backup.Tasks {
		if importFlagDryRun {
			stats.Tasks++
			continue
		}

		exists, _ := ctx.TaskRepo.Exists(t.ProjectSID, t.SID)
		if exists && !importFlagForce {
			stats.Duplicates++
			continue
		}

		if exists {
			if err := ctx.TaskRepo.Update(t); err != nil {
				return fmt.Errorf("failed to update task %s/%s: %w", t.ProjectSID, t.SID, err)
			}
		} else {
			if err := ctx.TaskRepo.Create(t); err != nil {
				return fmt.Errorf("failed to create task %s/%s: %w", t.ProjectSID, t.SID, err)
			}
		}
		stats.Tasks++
	}

	// Import blocks
	for _, b := range backup.Blocks {
		if importFlagDryRun {
			stats.Blocks++
			continue
		}

		// Check for duplicate by key
		_, err := ctx.BlockRepo.Get(b.Key)
		if err == nil && !importFlagForce {
			stats.Duplicates++
			continue
		}

		if err == nil {
			// Update existing
			if err := ctx.BlockRepo.Update(b); err != nil {
				return fmt.Errorf("failed to update block %s: %w", b.Key, err)
			}
		} else {
			// Create new
			if err := ctx.BlockRepo.Create(b); err != nil {
				return fmt.Errorf("failed to create block %s: %w", b.Key, err)
			}
		}
		stats.Blocks++
	}

	// Import goals
	for _, g := range backup.Goals {
		if importFlagDryRun {
			stats.Goals++
			continue
		}

		exists, _ := ctx.GoalRepo.Exists(g.ProjectSID)
		if exists && !importFlagForce {
			stats.Duplicates++
			continue
		}

		if err := ctx.GoalRepo.Upsert(g); err != nil {
			return fmt.Errorf("failed to import goal for %s: %w", g.ProjectSID, err)
		}
		stats.Goals++
	}

	// Print summary
	cli.Println("")
	if importFlagDryRun {
		cli.Printf("Would import:\n")
	} else {
		cli.Success("Import complete")
	}
	cli.Printf("  Projects: %d\n", stats.Projects)
	cli.Printf("  Tasks: %d\n", stats.Tasks)
	cli.Printf("  Blocks: %d\n", stats.Blocks)
	cli.Printf("  Goals: %d\n", stats.Goals)
	if stats.Duplicates > 0 {
		cli.Printf("  Skipped (duplicates): %d\n", stats.Duplicates)
	}

	return nil
}

func importZeit(data []byte, cli *output.CLIFormatter) error {
	// Try parsing as object with entries array
	var zeit ZeitExport
	if err := json.Unmarshal(data, &zeit); err != nil {
		// Try parsing as direct array
		var entries []ZeitEntry
		if err := json.Unmarshal(data, &entries); err != nil {
			return fmt.Errorf("failed to parse Zeit export: %w", err)
		}
		zeit.Entries = entries
	}

	stats := struct {
		Projects int
		Blocks   int
		Skipped  int
		Errors   int
	}{}

	if importFlagDryRun {
		cli.Title("Dry Run - Zeit Import Preview")
	} else {
		cli.Title("Importing Zeit Data")
	}

	// Track created projects
	createdProjects := make(map[string]bool)

	for _, entry := range zeit.Entries {
		// Parse timestamps
		begin, err := time.Parse(time.RFC3339, entry.Begin)
		if err != nil {
			stats.Errors++
			continue
		}

		var end time.Time
		if entry.End != "" {
			end, err = time.Parse(time.RFC3339, entry.End)
			if err != nil {
				stats.Errors++
				continue
			}
		}

		// Create project if needed
		projectSID := entry.Project
		if projectSID == "" {
			projectSID = "imported"
		}

		if !createdProjects[projectSID] && !importFlagDryRun {
			_, created, err := ctx.ProjectRepo.GetOrCreate(projectSID, projectSID)
			if err != nil {
				return fmt.Errorf("failed to create project %s: %w", projectSID, err)
			}
			if created {
				stats.Projects++
			}
			createdProjects[projectSID] = true
		}

		// Create block
		if importFlagDryRun {
			stats.Blocks++
			continue
		}

		block := model.NewBlock(ctx.Config.UserKey, projectSID, entry.Task, entry.Notes, begin)
		if !end.IsZero() {
			block.TimestampEnd = end
		}

		if err := ctx.BlockRepo.Create(block); err != nil {
			stats.Errors++
			continue
		}
		stats.Blocks++
	}

	// Print summary
	cli.Println("")
	if importFlagDryRun {
		cli.Printf("Would import:\n")
	} else {
		cli.Success("Zeit import complete")
	}
	cli.Printf("  Projects: %d\n", stats.Projects)
	cli.Printf("  Blocks: %d\n", stats.Blocks)
	if stats.Errors > 0 {
		cli.Printf("  Errors: %d\n", stats.Errors)
	}

	return nil
}
