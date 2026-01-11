package validate

import (
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// SID Tests
// =============================================================================

func TestSID(t *testing.T) {
	tests := []struct {
		name    string
		sid     string
		wantErr bool
	}{
		// Valid SIDs
		{"simple", "myproject", false},
		{"with_numbers", "project123", false},
		{"with_dash", "my-project", false},
		{"with_underscore", "my_project", false},
		{"with_period", "my.project", false},
		{"mixed", "My-Project_123.v2", false},
		{"single_char", "a", false},
		{"max_length", strings.Repeat("a", MaxSIDLength), false},

		// Invalid SIDs
		{"empty", "", true},
		{"too_long", strings.Repeat("a", MaxSIDLength+1), true},
		{"starts_with_dash", "-project", true},
		{"starts_with_period", ".project", true},
		{"starts_with_underscore", "_project", true},
		{"with_space", "my project", true},
		{"with_special", "my@project", true},
		{"with_slash", "my/project", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SID(tt.sid)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// ProjectName Tests
// =============================================================================

func TestProjectName(t *testing.T) {
	tests := []struct {
		name    string
		project string
		wantErr bool
	}{
		{"valid", "My Project", false},
		{"with_special", "Project @ Work!", false},
		{"unicode", "项目名称", false},
		{"max_length", strings.Repeat("a", MaxProjectNameLength), false},

		{"empty", "", true},
		{"too_long", strings.Repeat("a", MaxProjectNameLength+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProjectName(tt.project)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Note Tests
// =============================================================================

func TestNote(t *testing.T) {
	tests := []struct {
		name    string
		note    string
		wantErr bool
	}{
		{"empty", "", false},
		{"simple", "A simple note", false},
		{"max_length", strings.Repeat("a", MaxNoteLength), false},

		{"too_long", strings.Repeat("a", MaxNoteLength+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Note(tt.note)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// HexColor Tests
// =============================================================================

func TestHexColor(t *testing.T) {
	tests := []struct {
		name    string
		color   string
		wantErr bool
	}{
		// Valid
		{"empty", "", false},
		{"red", "#FF0000", false},
		{"green", "#00FF00", false},
		{"blue", "#0000FF", false},
		{"lowercase", "#ff5733", false},
		{"mixed_case", "#Ff5733", false},

		// Invalid
		{"no_hash", "FF0000", true},
		{"short", "#FFF", true},
		{"too_long", "#FF00000", true},
		{"invalid_chars", "#GGGGGG", true},
		{"special_chars", "#FF@@00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HexColor(tt.color)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// URL Tests
// =============================================================================

func TestURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Valid
		{"https", "https://example.com/webhook", false},
		{"https_with_port", "https://example.com:8080/webhook", false},
		{"https_with_path", "https://example.com/api/v1/webhook", false},
		{"localhost_http", "http://localhost/webhook", false},
		{"localhost_127", "http://127.0.0.1/webhook", false},
		{"localhost_ipv6", "http://[::1]/webhook", false},

		// Invalid
		{"empty", "", true},
		{"http_non_localhost", "http://example.com/webhook", true},
		{"ftp_scheme", "ftp://example.com/file", true},
		{"no_scheme", "example.com/webhook", true},
		{"too_long", "https://example.com/" + strings.Repeat("a", MaxURLLength), true},

		// Internal IPs should be rejected
		{"internal_10", "https://10.0.0.1/webhook", true},
		{"internal_172", "https://172.16.0.1/webhook", true},
		{"internal_192", "https://192.168.1.1/webhook", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := URL(tt.url)
			if tt.wantErr {
				assert.Error(t, err, "URL: %s", tt.url)
			} else {
				assert.NoError(t, err, "URL: %s", tt.url)
			}
		})
	}
}

func TestURLMissingHostname(t *testing.T) {
	err := URL("https:///path")
	assert.Error(t, err)
}

// =============================================================================
// IsInternalIP Tests
// =============================================================================

func TestIsInternalIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		internal bool
	}{
		// Internal
		{"10.x", "10.0.0.1", true},
		{"172.16.x", "172.16.0.1", true},
		{"192.168.x", "192.168.1.1", true},
		{"127.x", "127.0.0.1", true},
		{"169.254.x", "169.254.0.1", true},

		// External
		{"8.8.8.8", "8.8.8.8", false},
		{"1.1.1.1", "1.1.1.1", false},
		{"93.184.216.34", "93.184.216.34", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("invalid IP: %s", tt.ip)
			}
			result := isInternalIP(ip)
			assert.Equal(t, tt.internal, result)
		})
	}
}

// =============================================================================
// NonEmpty Tests
// =============================================================================

func TestNonEmpty(t *testing.T) {
	tests := []struct {
		field   string
		value   string
		wantErr bool
	}{
		{"name", "hello", false},
		{"name", " hello ", false},

		{"name", "", true},
		{"name", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := NonEmpty(tt.field, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// InRange Tests
// =============================================================================

func TestInRange(t *testing.T) {
	tests := []struct {
		value   int
		min     int
		max     int
		wantErr bool
	}{
		{5, 1, 10, false},
		{1, 1, 10, false},
		{10, 1, 10, false},

		{0, 1, 10, true},
		{11, 1, 10, true},
		{-1, 0, 10, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := InRange("field", tt.value, tt.min, tt.max)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Constants Tests
// =============================================================================

func TestConstants(t *testing.T) {
	assert.Equal(t, 32, MaxSIDLength)
	assert.Equal(t, 2048, MaxURLLength)
	assert.Equal(t, 128, MaxProjectNameLength)
	assert.Equal(t, 4096, MaxNoteLength)
}

// =============================================================================
// Sanitize Functions Tests
// =============================================================================

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"alphanumeric", "Hello123", "Hello123"},
		{"with_spaces", "Hello World", "Hello World"},
		{"with_allowed", "user@email.com", "user@email.com"},
		{"with_special", "Hello<script>", "Helloscript"},
		{"with_symbols", "Test!#$%", "Test"},
		{"unicode", "Héllo", "Héllo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeProjectName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "My Project", "My Project"},
		{"with_whitespace", "  My Project  ", "My Project"},
		{"with_control", "My\x00Project", "MyProject"},
		{"with_tab", "My\tProject", "MyProject"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeProjectName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeNote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "A note", "A note"},
		{"with_whitespace", "  A note  ", "A note"},
		{"with_null", "A\x00note", "Anote"},
		{"with_crlf", "A\r\nnote", "A\nnote"},
		{"with_cr", "A\rnote", "A\nnote"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeNote(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "file.txt", "file.txt"},
		{"with_dir", "dir/file.txt", "dir/file.txt"},
		{"with_traversal", "../file.txt", "file.txt"},
		{"multiple_traversal", "../../dir/file.txt", "dir/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPathTraversal(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		isTraversal bool
	}{
		{"simple", "file.txt", false},
		{"subdir", "dir/file.txt", false},
		{"dot_dir", "./file.txt", false},
		{"traversal", "../file.txt", true},
		{"deep_traversal", "../../file.txt", true},
		{"middle_traversal", "dir/../file.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPathTraversal(tt.path)
			assert.Equal(t, tt.isTraversal, result)
		})
	}
}

func TestIsWithinDirectory(t *testing.T) {
	// Use temp directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		baseDir  string
		expected bool
	}{
		{"within", tmpDir + "/subdir/file.txt", tmpDir, true},
		{"same", tmpDir, tmpDir, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWithinDirectory(tt.path, tt.baseDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeTag(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "mytag", "mytag"},
		{"uppercase", "MyTag", "mytag"},
		{"with_dash", "my-tag", "my-tag"},
		{"with_space", " my tag ", "mytag"},
		{"with_special", "my@tag!", "mytag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeTag(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripControlChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "Hello", "Hello"},
		{"with_newline", "Hello\nWorld", "Hello\nWorld"},
		{"with_tab", "Hello\tWorld", "Hello\tWorld"},
		{"with_control", "Hello\x00World", "HelloWorld"},
		{"with_bell", "Hello\x07World", "HelloWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripControlChars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short", "Hello", 10, "Hello"},
		{"exact", "Hello", 5, "Hello"},
		{"truncate", "Hello World", 8, "Hello..."},
		{"very_short_limit", "Hello", 3, "Hel"},
		{"short_limit", "Hello World", 4, "H..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "file.txt", "file.txt"},
		{"with_slash", "path/file.txt", "path_file.txt"},
		{"with_backslash", "path\\file.txt", "path_file.txt"},
		{"with_special", "file:name?.txt", "file_name_.txt"},
		{"with_dots", "...file...", "file"},
		{"with_spaces", "  file  ", "file"},
		{"with_null", "file\x00name", "filename"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeFilenameLongInput(t *testing.T) {
	longInput := strings.Repeat("a", 250)
	result := SafeFilename(longInput)
	assert.LessOrEqual(t, len(result), 200)
}
