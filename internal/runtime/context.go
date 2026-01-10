// Package runtime provides application runtime context for Humantime.
package runtime

import (
	"os"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/storage"
)

// Context holds the application runtime context.
type Context struct {
	DB        *storage.DB
	Config    *model.Config
	Formatter *output.Formatter

	// Repositories
	BlockRepo        *storage.BlockRepo
	ProjectRepo      *storage.ProjectRepo
	TaskRepo         *storage.TaskRepo
	ConfigRepo       *storage.ConfigRepo
	ActiveBlockRepo  *storage.ActiveBlockRepo
	GoalRepo         *storage.GoalRepo
	UndoRepo         *storage.UndoRepo
	ReminderRepo     *storage.ReminderRepo
	WebhookRepo      *storage.WebhookRepo
	NotifyConfigRepo *storage.NotifyConfigRepo

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
	taskRepo := storage.NewTaskRepo(db)
	configRepo := storage.NewConfigRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	undoRepo := storage.NewUndoRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	// Get or create config
	config, err := configRepo.Get()
	if err != nil {
		db.Close()
		return nil, err
	}

	// Create formatter
	formatter := output.NewFormatter()
	formatter.Format = opts.Format
	formatter.ColorMode = opts.ColorMode

	return &Context{
		DB:               db,
		Config:           config,
		Formatter:        formatter,
		BlockRepo:        blockRepo,
		ProjectRepo:      projectRepo,
		TaskRepo:         taskRepo,
		ConfigRepo:       configRepo,
		ActiveBlockRepo:  activeBlockRepo,
		GoalRepo:         goalRepo,
		UndoRepo:         undoRepo,
		ReminderRepo:     reminderRepo,
		WebhookRepo:      webhookRepo,
		NotifyConfigRepo: notifyConfigRepo,
		Debug:            opts.Debug,
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
