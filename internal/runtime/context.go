// Package runtime provides application runtime context for Humantime.
package runtime

import (
	"os"

	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/storage"
)

// Context holds the application runtime context.
type Context struct {
	DB        *storage.DB
	Formatter *output.Formatter

	// Repositories
	BlockRepo       *storage.BlockRepo
	ProjectRepo     *storage.ProjectRepo
	ActiveBlockRepo *storage.ActiveBlockRepo
	UndoRepo        *storage.UndoRepo

	// Debug mode
	Debug bool
}

// Options configures the runtime context.
type Options struct {
	DBPath    string
	InMemory  bool
	Format    output.Format
	ColorMode output.ColorMode
	Debug     bool
}

// DefaultOptions returns default runtime options.
func DefaultOptions() Options {
	return Options{
		DBPath:    storage.DefaultPath(),
		InMemory:  false,
		Format:    output.FormatCLI,
		ColorMode: output.ColorAuto,
		Debug:     false,
	}
}

// New creates a new runtime context.
func New(opts Options) (*Context, error) {
	// Check for environment variable override
	if envPath := os.Getenv("HUMANTIME_DATABASE"); envPath != "" {
		if envPath == ":memory:" {
			opts.InMemory = true
		} else {
			opts.DBPath = envPath
		}
	}

	// Open database
	db, err := storage.Open(storage.Options{
		Path:     opts.DBPath,
		InMemory: opts.InMemory,
	})
	if err != nil {
		return nil, err
	}

	// Create repositories
	blockRepo := storage.NewBlockRepo(db)
	projectRepo := storage.NewProjectRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	undoRepo := storage.NewUndoRepo(db)

	// Create formatter
	formatter := output.NewFormatter()
	formatter.Format = opts.Format
	formatter.ColorMode = opts.ColorMode

	return &Context{
		DB:              db,
		Formatter:       formatter,
		BlockRepo:       blockRepo,
		ProjectRepo:     projectRepo,
		ActiveBlockRepo: activeBlockRepo,
		UndoRepo:        undoRepo,
		Debug:           opts.Debug,
	}, nil
}

// Close closes the runtime context.
func (c *Context) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

// CLIFormatter returns a CLI formatter.
func (c *Context) CLIFormatter() *output.CLIFormatter {
	return output.NewCLIFormatter(c.Formatter)
}

// JSONFormatter returns a JSON formatter.
func (c *Context) JSONFormatter() *output.JSONFormatter {
	return output.NewJSONFormatter(c.Formatter)
}

// IsJSON returns true if output format is JSON.
func (c *Context) IsJSON() bool {
	return c.Formatter.Format == output.FormatJSON
}

// IsCLI returns true if output format is CLI.
func (c *Context) IsCLI() bool {
	return c.Formatter.Format == output.FormatCLI
}

// Debugf prints debug output if debug mode is enabled.
func (c *Context) Debugf(format string, args ...interface{}) {
	if c.Debug {
		c.Formatter.Printf("[DEBUG] "+format+"\n", args...)
	}
}
