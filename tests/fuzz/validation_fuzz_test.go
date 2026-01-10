package fuzz

import (
	"testing"

	"github.com/manav03panchal/humantime/internal/validate"
)

// FuzzSanitizeString tests string sanitization with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzSanitizeString -fuzztime=30s
func FuzzSanitizeString(f *testing.F) {
	// Seed corpus with various inputs
	seeds := []string{
		"normal text",
		"hello\x00world",
		"test\x1b[31mred",
		"café résumé",
		"日本語テスト",
		"emoji 😀🎉",
		"path/../../../etc/passwd",
		"'; DROP TABLE users;--",
		"<script>alert('xss')</script>",
		string(make([]byte, 10000)), // Large input
		"",
		"   ",
		"\t\n\r",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// SanitizeString should never panic
		_ = validate.SanitizeString(input)
	})
}

// FuzzSanitizePath tests path sanitization with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzSanitizePath -fuzztime=30s
func FuzzSanitizePath(f *testing.F) {
	seeds := []string{
		"/home/user/file.txt",
		"../../../etc/passwd",
		"..\\..\\windows\\system32",
		"./relative/path",
		"/absolute/path/to/file",
		"file\x00.txt",
		"path%2F..%2Fetc",
		"",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// SanitizePath should never panic
		_ = validate.SanitizePath(input)
	})
}

// FuzzIsPathTraversal tests path traversal detection with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzIsPathTraversal -fuzztime=30s
func FuzzIsPathTraversal(f *testing.F) {
	seeds := []string{
		"../etc/passwd",
		"/safe/path",
		"..%2F..%2Fetc",
		"....//....//etc",
		"./current",
		"",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// IsPathTraversal should never panic
		_ = validate.IsPathTraversal(input)
	})
}

// FuzzSanitizeProjectName tests project name sanitization with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzSanitizeProjectName -fuzztime=30s
func FuzzSanitizeProjectName(f *testing.F) {
	seeds := []string{
		"My Project",
		"project\x00name",
		"  trimmed  ",
		"日本語プロジェクト",
		"emoji 🚀 project",
		string(make([]byte, 10000)),
		"",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// SanitizeProjectName should never panic
		_ = validate.SanitizeProjectName(input)
	})
}

// FuzzSanitizeTag tests tag sanitization with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzSanitizeTag -fuzztime=30s
func FuzzSanitizeTag(f *testing.F) {
	seeds := []string{
		"work",
		"Work-Project",
		"  UPPERCASE  ",
		"tag@123!",
		"日本語",
		"",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// SanitizeTag should never panic
		_ = validate.SanitizeTag(input)
	})
}

// FuzzSafeFilename tests safe filename generation with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzSafeFilename -fuzztime=30s
func FuzzSafeFilename(f *testing.F) {
	seeds := []string{
		"normal.txt",
		"file/with/slashes.txt",
		"file:with:colons.txt",
		"file<>with|special?.txt",
		"\x00nullbyte.txt",
		string(make([]byte, 500)),
		"",
		".",
		"..",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// SafeFilename should never panic
		_ = validate.SafeFilename(input)
	})
}

// FuzzTruncateString tests string truncation with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzTruncateString -fuzztime=30s
func FuzzTruncateString(f *testing.F) {
	// Add input/length pairs
	f.Add("hello world", 5)
	f.Add("short", 100)
	f.Add(string(make([]byte, 10000)), 50)
	f.Add("", 10)
	f.Add("test", 0)
	f.Add("test", -1)

	f.Fuzz(func(t *testing.T, input string, maxLen int) {
		// TruncateString should never panic
		// Handle negative lengths gracefully
		if maxLen < 0 {
			maxLen = 0
		}
		_ = validate.TruncateString(input, maxLen)
	})
}
