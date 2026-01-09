package parser

import (
	"regexp"
	"strings"
	"time"
)

// ParsedArgs holds the parsed command arguments.
type ParsedArgs struct {
	ProjectSID     string
	TaskSID        string
	Note           string
	TimestampStart time.Time
	TimestampEnd   time.Time

	// Raw strings before processing
	RawProject        string
	RawTimestampStart string
	RawTimestampEnd   string

	// Flags for what was found
	HasProject   bool
	HasTask      bool
	HasNote      bool
	HasStart     bool
	HasEnd       bool
}

// Keywords for natural language parsing.
var (
	projectKeywords = []string{"on", "to", "of"}
	noteKeyword     = "with"
	endKeywords     = []string{"end", "ended", "until", "to"}
	skipWords       = map[string]bool{
		"block": true, "working": true, "work": true, "all": true,
		"at": true, "from": true, "note": true,
	}
)

// noteRegex matches 'with note "..."' or "with note '...'" patterns.
var noteRegex = regexp.MustCompile(`(?i)with\s+note\s+['"]([^'"]+)['"]`)

// Parse parses command-line arguments into structured data.
func Parse(args []string) *ParsedArgs {
	result := &ParsedArgs{}
	if len(args) == 0 {
		return result
	}

	// Join args for regex matching
	fullInput := strings.Join(args, " ")

	// Extract note first (it can contain spaces)
	if match := noteRegex.FindStringSubmatch(fullInput); match != nil {
		result.Note = match[1]
		result.HasNote = true
		// Remove note from input for further processing
		fullInput = noteRegex.ReplaceAllString(fullInput, "")
	}

	// Re-split after note extraction
	tokens := tokenize(fullInput)

	// State machine for parsing
	var (
		expectProject   bool
		expectTimestamp bool
		isEndTimestamp  bool
		timestampTokens []string
	)

	for i, token := range tokens {
		tokenLower := strings.ToLower(token)

		// Skip reserved words
		if skipWords[tokenLower] {
			continue
		}

		// Check for project keywords
		if containsString(projectKeywords, tokenLower) {
			expectProject = true
			continue
		}

		// Check for end keywords
		if containsString(endKeywords, tokenLower) {
			// Save any accumulated timestamp tokens as start
			if len(timestampTokens) > 0 {
				result.RawTimestampStart = strings.Join(timestampTokens, " ")
				result.HasStart = true
				timestampTokens = nil
			}
			isEndTimestamp = true
			expectTimestamp = true
			continue
		}

		// If expecting project, the next token is the project/task
		// But first check if it looks like a time expression (e.g., "11am", "9pm")
		if expectProject {
			if isTimeLike(token) {
				// It's a time expression, not a project name
				timestampTokens = append(timestampTokens, token)
				expectTimestamp = true
				expectProject = false
				continue
			}
			result.RawProject = token
			result.ProjectSID, result.TaskSID = ParseProjectTask(token)
			result.HasProject = result.ProjectSID != ""
			result.HasTask = result.TaskSID != ""
			expectProject = false
			continue
		}

		// If it's the first token and looks like a project (contains no spaces, not a number)
		if i == 0 && !isTimeLike(token) && !expectTimestamp {
			// Check if next token is a keyword
			if len(tokens) > 1 && containsString(projectKeywords, strings.ToLower(tokens[1])) {
				// This is a command like "blocks on project"
				continue
			}
		}

		// Otherwise, it's part of a timestamp
		timestampTokens = append(timestampTokens, token)
		expectTimestamp = true
	}

	// Process remaining timestamp tokens
	if len(timestampTokens) > 0 {
		tsString := strings.Join(timestampTokens, " ")
		if isEndTimestamp {
			result.RawTimestampEnd = tsString
			result.HasEnd = true
		} else {
			result.RawTimestampStart = tsString
			result.HasStart = true
		}
	}

	return result
}

// Process converts raw strings to typed values.
func (p *ParsedArgs) Process() error {
	// Process project/task SIDs
	if p.RawProject != "" && p.ProjectSID == "" {
		p.ProjectSID, p.TaskSID = ParseProjectTask(p.RawProject)
		p.HasProject = p.ProjectSID != ""
		p.HasTask = p.TaskSID != ""
	}

	// Normalize SIDs
	if p.ProjectSID != "" {
		p.ProjectSID = NormalizeSID(p.ProjectSID)
	}
	if p.TaskSID != "" {
		p.TaskSID = NormalizeSID(p.TaskSID)
	}

	// Process timestamps
	if p.RawTimestampStart != "" {
		result := ParseTimestamp(p.RawTimestampStart)
		if result.Error != nil {
			return result.Error
		}
		p.TimestampStart = result.Time
	} else {
		p.TimestampStart = time.Now()
	}

	if p.RawTimestampEnd != "" {
		result := ParseTimestamp(p.RawTimestampEnd)
		if result.Error != nil {
			return result.Error
		}
		p.TimestampEnd = result.Time
	}

	return nil
}

// Merge merges flag values into parsed args (flags override).
func (p *ParsedArgs) Merge(projectFlag, taskFlag, noteFlag, startFlag, endFlag string) {
	if projectFlag != "" {
		p.ProjectSID = projectFlag
		p.HasProject = true
	}
	if taskFlag != "" {
		p.TaskSID = taskFlag
		p.HasTask = true
	}
	if noteFlag != "" {
		p.Note = noteFlag
		p.HasNote = true
	}
	if startFlag != "" {
		p.RawTimestampStart = startFlag
		p.HasStart = true
	}
	if endFlag != "" {
		p.RawTimestampEnd = endFlag
		p.HasEnd = true
	}
}

// tokenize splits input into tokens, preserving quoted strings.
func tokenize(input string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range input {
		if (r == '"' || r == '\'') && !inQuote {
			inQuote = true
			quoteChar = r
			continue
		}
		if r == quoteChar && inQuote {
			inQuote = false
			quoteChar = 0
			continue
		}
		if r == ' ' && !inQuote {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// containsString checks if a slice contains a string.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// isTimeLike checks if a token looks like a time expression.
func isTimeLike(token string) bool {
	timeLikeWords := []string{
		"now", "today", "yesterday", "tomorrow",
		"hour", "hours", "minute", "minutes", "second", "seconds",
		"day", "days", "week", "weeks", "month", "months", "year", "years",
		"ago", "last", "this", "next", "previous", "current",
		"am", "pm", "morning", "afternoon", "evening", "night",
		"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
		"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec",
	}

	tokenLower := strings.ToLower(token)
	for _, word := range timeLikeWords {
		if tokenLower == word || strings.HasPrefix(tokenLower, word) {
			return true
		}
	}

	// Check if it's a number (could be time)
	if len(token) > 0 && token[0] >= '0' && token[0] <= '9' {
		return true
	}

	return false
}
