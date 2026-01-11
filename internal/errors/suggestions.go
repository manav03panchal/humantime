package errors

import "errors"

// Suggestions maps common errors to helpful suggestions.
var Suggestions = map[error]string{
	// User input errors
	ErrNoActiveTracking:   "Use 'humantime start on <project>' to begin tracking.",
	ErrProjectRequired:    "Specify a project with 'on <project>' or use the -p flag.",
	ErrInvalidSID:         "SIDs must be alphanumeric with dashes, underscores, or periods (max 32 chars).",
	ErrInvalidTimestamp:   "Try formats like '2 hours ago', 'yesterday at 3pm', '9am', or '1 minute'.",
	ErrEndBeforeStart:     "Check your timestamps - end time must be after start time.",
	ErrBlockNotFound:      "Use 'humantime blocks' to see available blocks.",
	ErrProjectNotFound:    "Use 'humantime project' to see available projects.",
	ErrTaskNotFound:       "Use 'humantime task' to see tasks for a project.",
	ErrGoalNotFound:       "Use 'humantime goal' to see or create goals.",
	ErrReminderNotFound:   "Use 'humantime remind list' to see active reminders.",
	ErrWebhookNotFound:    "Use 'humantime webhook list' to see configured webhooks.",
	ErrInvalidColor:       "Use hex color format like '#FF5733' or '#00FF00'.",
	ErrInvalidGoalType:    "Use --daily or --weekly to set goal type.",
	ErrInvalidDuration:    "Try formats like '1h30m', '90m', '2h', or '45 minutes'.",
	ErrInvalidURL:         "Provide a valid URL starting with https:// (or http:// for localhost).",

	// System errors
	ErrDiskFull:           "Free up disk space and try again. Your active tracking is preserved in memory.",
	ErrDatabaseCorrupted:  "Run 'humantime doctor' to diagnose and repair database issues.",
	ErrNetworkUnavailable: "Check your internet connection. Notifications will retry automatically.",
	ErrLockHeld:           "Another humantime instance is running. Use 'humantime daemon stop' or check for stale processes.",
	ErrTimeout:            "The operation took too long. Try again or check your network connection.",
	ErrPermissionDenied:   "Check file permissions in your data directory (~/.local/share/humantime/).",
}

// GetSuggestion returns a suggestion for an error, if available.
// It walks the error chain to find matching suggestions.
func GetSuggestion(err error) string {
	if err == nil {
		return ""
	}

	// Check exact match first
	for knownErr, suggestion := range Suggestions {
		if errors.Is(err, knownErr) {
			return suggestion
		}
	}

	// Check if it's a UserError with a suggestion
	if ue, ok := AsUserError(err); ok && ue.Suggestion != "" {
		return ue.Suggestion
	}

	return ""
}

// GetCategorySuggestion returns a generic suggestion based on error category.
func GetCategorySuggestion(err error) string {
	if IsUserError(err) {
		return "Check your input and try again. Use --help for usage information."
	}
	if IsSystemError(err) {
		return "This is a system error. Check system resources and try again."
	}
	if IsRecoverableError(err) {
		return "This error may resolve itself. The operation will be retried automatically."
	}
	return ""
}

// CommandExamples provides example commands for common errors.
var CommandExamples = map[error][]string{
	ErrNoActiveTracking: {
		"humantime start on myproject",
		"humantime start on myproject with \"working on feature\"",
		"humantime start on myproject/task-1",
	},
	ErrProjectRequired: {
		"humantime start on myproject",
		"humantime start -p myproject",
	},
	ErrInvalidTimestamp: {
		"humantime start at 9am on myproject",
		"humantime stop at 5pm",
		"humantime edit --start '2 hours ago'",
		"humantime remind \"meeting\" in 1 minute",
	},
	ErrInvalidDuration: {
		"humantime goal set myproject --daily 8h",
		"humantime pomodoro 25m",
		"humantime remind \"break\" 30m",
	},
}

// GetExamples returns example commands for an error.
func GetExamples(err error) []string {
	for knownErr, examples := range CommandExamples {
		if errors.Is(err, knownErr) {
			return examples
		}
	}
	return nil
}
