package validate

import (
	"path/filepath"
	"strings"
	"unicode"
)

// SanitizeString removes or replaces potentially dangerous characters.
// It keeps alphanumeric characters, spaces, and common punctuation.
func SanitizeString(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) ||
			r == ' ' || r == '-' || r == '_' || r == '.' ||
			r == ',' || r == ':' || r == '/' || r == '@' {
			sb.WriteRune(r)
		}
	}

	return sb.String()
}

// SanitizeProjectName cleans a project name for safe use.
func SanitizeProjectName(name string) string {
	// Trim whitespace
	name = strings.TrimSpace(name)

	// Remove control characters
	var sb strings.Builder
	for _, r := range name {
		if !unicode.IsControl(r) {
			sb.WriteRune(r)
		}
	}

	return sb.String()
}

// SanitizeNote cleans a note/description for safe storage.
func SanitizeNote(note string) string {
	// Trim whitespace
	note = strings.TrimSpace(note)

	// Remove null bytes (common injection attempt)
	note = strings.ReplaceAll(note, "\x00", "")

	// Normalize line endings
	note = strings.ReplaceAll(note, "\r\n", "\n")
	note = strings.ReplaceAll(note, "\r", "\n")

	return note
}

// SanitizePath cleans a file path and prevents traversal attacks.
func SanitizePath(path string) string {
	// Clean the path
	path = filepath.Clean(path)

	// Remove any .. components
	parts := strings.Split(path, string(filepath.Separator))
	var cleaned []string
	for _, part := range parts {
		if part != ".." && part != "" {
			cleaned = append(cleaned, part)
		}
	}

	return filepath.Join(cleaned...)
}

// IsPathTraversal checks if a path contains traversal patterns.
func IsPathTraversal(path string) bool {
	// Check for .. in the path
	if strings.Contains(path, "..") {
		return true
	}

	// Check if cleaned path differs significantly
	cleaned := filepath.Clean(path)
	if cleaned != path && strings.Contains(path, "..") {
		return true
	}

	return false
}

// IsWithinDirectory checks if a path is within the given base directory.
func IsWithinDirectory(path, baseDir string) bool {
	// Clean both paths
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}

	// Ensure base ends with separator for proper prefix check
	if !strings.HasSuffix(absBase, string(filepath.Separator)) {
		absBase += string(filepath.Separator)
	}

	// Check if path is within base
	return strings.HasPrefix(absPath, absBase) || absPath == strings.TrimSuffix(absBase, string(filepath.Separator))
}

// SanitizeTag cleans a tag for safe use.
func SanitizeTag(tag string) string {
	// Trim whitespace
	tag = strings.TrimSpace(tag)

	// Convert to lowercase
	tag = strings.ToLower(tag)

	// Keep only alphanumeric and dashes
	var sb strings.Builder
	for _, r := range tag {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' {
			sb.WriteRune(r)
		}
	}

	return sb.String()
}

// StripControlChars removes all control characters from a string.
func StripControlChars(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if !unicode.IsControl(r) || r == '\n' || r == '\t' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// TruncateString truncates a string to the given length, adding "..." if truncated.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// SafeFilename converts a string to a safe filename.
func SafeFilename(s string) string {
	// Replace unsafe characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\x00", "",
	)
	s = replacer.Replace(s)

	// Trim whitespace and dots from ends
	s = strings.Trim(s, " .")

	// Limit length
	if len(s) > 200 {
		s = s[:200]
	}

	return s
}
