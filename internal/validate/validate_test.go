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
