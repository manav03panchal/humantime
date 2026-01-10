package security

import (
	"strings"
	"testing"

	"github.com/manav03panchal/humantime/internal/validate"
)

// TestSanitizeStringBasic tests basic string sanitization.
func TestSanitizeStringBasic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"hello-world", "hello-world"},
		{"test_123", "test_123"},
		{"", ""},
	}

	for _, tc := range tests {
		result := validate.SanitizeString(tc.input)
		if result != tc.expected {
			t.Errorf("SanitizeString(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

// TestSanitizeStringRemovesControlChars tests control character removal.
func TestSanitizeStringRemovesControlChars(t *testing.T) {
	// Control characters that should be removed
	inputs := []string{
		"hello\x00world",      // null byte
		"hello\x1bworld",      // escape
		"hello\x7fworld",      // delete
	}

	for _, input := range inputs {
		result := validate.SanitizeString(input)

		// Should not contain null bytes or escape chars
		if strings.Contains(result, "\x00") {
			t.Errorf("SanitizeString should remove null bytes from %q", input)
		}
		if strings.Contains(result, "\x1b") {
			t.Errorf("SanitizeString should remove escape chars from %q", input)
		}
	}
}

// TestSanitizePathBasic tests basic path sanitization.
func TestSanitizePathBasic(t *testing.T) {
	// Test that clean paths remain usable
	tests := []struct {
		input string
	}{
		{"/home/user/file.txt"},
		{"relative/path"},
		{"file.txt"},
	}

	for _, tc := range tests {
		result := validate.SanitizePath(tc.input)
		// Path should not be empty if input was not empty
		if tc.input != "" && result == "" {
			t.Errorf("SanitizePath(%q) should not be empty", tc.input)
		}
	}
}

// TestIsPathTraversal tests path traversal detection.
func TestIsPathTraversal(t *testing.T) {
	traversalPaths := []string{
		"../etc/passwd",
		"..\\windows\\system32",
		"/path/../../../etc/passwd",
		"foo/../../bar",
	}

	for _, path := range traversalPaths {
		if !validate.IsPathTraversal(path) {
			t.Errorf("IsPathTraversal(%q) should return true", path)
		}
	}
}

// TestIsPathTraversalSafe tests safe paths.
func TestIsPathTraversalSafe(t *testing.T) {
	safePaths := []string{
		"/home/user/file.txt",
		"relative/path",
		"file.txt",
		"/absolute/path/to/file",
	}

	for _, path := range safePaths {
		if validate.IsPathTraversal(path) {
			t.Errorf("IsPathTraversal(%q) should return false", path)
		}
	}
}

// TestIsWithinDirectory tests directory containment check.
func TestIsWithinDirectory(t *testing.T) {
	tests := []struct {
		path    string
		baseDir string
		within  bool
	}{
		{"/home/user/data/file.txt", "/home/user/data", true},
		{"/home/user/data/subdir/file", "/home/user/data", true},
	}

	for _, tc := range tests {
		result := validate.IsWithinDirectory(tc.path, tc.baseDir)
		if result != tc.within {
			t.Errorf("IsWithinDirectory(%q, %q) = %v, want %v",
				tc.path, tc.baseDir, result, tc.within)
		}
	}
}

// TestNullByteInjection tests null byte injection prevention.
func TestNullByteInjection(t *testing.T) {
	input := "valid.txt\x00.exe"
	result := validate.SanitizeString(input)

	if strings.Contains(result, "\x00") {
		t.Error("Sanitized string should not contain null bytes")
	}
}

// TestSanitizeProjectName tests project name sanitization.
func TestSanitizeProjectName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  my project  ", "my project"},
		{"project\x00name", "projectname"},
		{"normal", "normal"},
	}

	for _, tc := range tests {
		result := validate.SanitizeProjectName(tc.input)
		if result != tc.expected {
			t.Errorf("SanitizeProjectName(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

// TestSanitizeNote tests note sanitization.
func TestSanitizeNote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  note  ", "note"},
		{"line1\r\nline2", "line1\nline2"},
		{"null\x00byte", "nullbyte"},
	}

	for _, tc := range tests {
		result := validate.SanitizeNote(tc.input)
		if result != tc.expected {
			t.Errorf("SanitizeNote(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

// TestSanitizeTag tests tag sanitization.
func TestSanitizeTag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  MyTag  ", "mytag"},
		{"Work-Project", "work-project"},
		{"tag@123!", "tag123"},
	}

	for _, tc := range tests {
		result := validate.SanitizeTag(tc.input)
		if result != tc.expected {
			t.Errorf("SanitizeTag(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

// TestTruncateString tests string truncation.
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"ab", 2, "ab"},
		{"abc", 2, "ab"},
	}

	for _, tc := range tests {
		result := validate.TruncateString(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("TruncateString(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
		}
	}
}

// TestMaxLengthEnforcement tests that long strings can be truncated.
func TestMaxLengthEnforcement(t *testing.T) {
	// Very long input
	longInput := strings.Repeat("a", 10000)

	// Use TruncateString to enforce limit
	result := validate.TruncateString(longInput, 4096)

	if len(result) > 4096 {
		t.Errorf("Truncated string too long: %d bytes", len(result))
	}
}

// TestStripControlChars tests control character stripping.
func TestStripControlChars(t *testing.T) {
	input := "hello\x00\x1b\x7fworld"
	result := validate.StripControlChars(input)

	if strings.Contains(result, "\x00") || strings.Contains(result, "\x1b") {
		t.Error("StripControlChars should remove control characters")
	}
	if !strings.Contains(result, "hello") || !strings.Contains(result, "world") {
		t.Error("StripControlChars should preserve normal text")
	}
}

// TestSafeFilename tests safe filename generation.
func TestSafeFilename(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"normal.txt", false},
		{"file/with/slashes.txt", false},
		{"file:with:colons.txt", false},
		{"  spaces  ", false},
	}

	for _, tc := range tests {
		result := validate.SafeFilename(tc.input)
		if result == "" && tc.input != "" {
			t.Errorf("SafeFilename(%q) should not be empty", tc.input)
		}
		// Should not contain unsafe chars
		if strings.ContainsAny(result, "/\\:*?\"<>|") {
			t.Errorf("SafeFilename(%q) = %q, contains unsafe chars", tc.input, result)
		}
	}
}

// TestEmptyAndWhitespace tests edge cases.
func TestEmptyAndWhitespace(t *testing.T) {
	// Empty string
	result := validate.SanitizeString("")
	if result != "" {
		t.Errorf("SanitizeString(\"\") = %q, want \"\"", result)
	}

	// Whitespace-only - SanitizeString keeps spaces
	result = validate.SanitizeString("   ")
	if result != "   " {
		t.Errorf("SanitizeString(\"   \") = %q, want \"   \"", result)
	}

	// Project name should trim whitespace
	result = validate.SanitizeProjectName("   ")
	if result != "" {
		t.Errorf("SanitizeProjectName(\"   \") = %q, want \"\"", result)
	}
}

// TestUnicodePreserved tests that unicode is preserved.
func TestUnicodePreserved(t *testing.T) {
	inputs := []string{
		"café",
		"日本語",
		"emoji 😀", // note: emoji may be filtered by SanitizeString
	}

	for _, input := range inputs {
		// SanitizeProjectName preserves unicode (just removes control chars)
		result := validate.SanitizeProjectName(input)
		if result == "" && input != "" {
			t.Errorf("SanitizeProjectName(%q) should preserve unicode content", input)
		}
	}
}
