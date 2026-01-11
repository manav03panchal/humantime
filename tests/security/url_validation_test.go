package security

import (
	"testing"

	"github.com/manav03panchal/humantime/internal/validate"
)

func TestURLValidation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Valid URLs
		{"valid https", "https://discord.com/webhook/123", false},
		{"valid https with path", "https://hooks.slack.com/services/T00/B00/xxx", false},
		{"valid localhost http", "http://localhost:8080/webhook", false},
		{"valid localhost https", "https://localhost:8080/webhook", false},
		{"valid 127.0.0.1 http", "http://127.0.0.1:8080/webhook", false},

		// Invalid schemes
		{"no scheme", "discord.com/webhook", true},
		{"ftp scheme", "ftp://discord.com/webhook", true},
		{"javascript scheme", "javascript:alert(1)", true},
		{"file scheme", "file:///etc/passwd", true},

		// HTTP on non-localhost (should fail)
		{"http on external", "http://discord.com/webhook", true},
		{"http on external ip", "http://1.2.3.4/webhook", true},

		// Internal IPs (SSRF protection)
		{"internal 10.x", "https://10.0.0.1/webhook", true},
		{"internal 172.16.x", "https://172.16.0.1/webhook", true},
		{"internal 192.168.x", "https://192.168.1.1/webhook", true},
		{"internal 127.x non-localhost", "https://127.0.0.2/webhook", true},

		// Empty and too long
		{"empty url", "", true},
		{"too long url", "https://example.com/" + string(make([]byte, 2100)), true},

		// Malformed
		{"malformed url", "https://", true},
		{"spaces in url", "https://example .com/webhook", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.URL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("URL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestSIDValidation(t *testing.T) {
	tests := []struct {
		name    string
		sid     string
		wantErr bool
	}{
		// Valid SIDs
		{"simple", "myproject", false},
		{"with-dash", "my-project", false},
		{"with_underscore", "my_project", false},
		{"with.dot", "my.project", false},
		{"with-numbers", "project123", false},
		{"mixed", "my-project_v1.2", false},
		{"single-char", "a", false},

		// Invalid SIDs
		{"empty", "", true},
		{"starts-with-dash", "-project", true},
		{"starts-with-dot", ".project", true},
		{"starts-with-underscore", "_project", true},
		{"too-long", "this-is-a-very-long-project-name-that-exceeds-32-characters", true},
		{"with-spaces", "my project", true},
		{"with-special", "my@project", true},
		{"with-slash", "my/project", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.SID(tt.sid)
			if (err != nil) != tt.wantErr {
				t.Errorf("SID(%q) error = %v, wantErr %v", tt.sid, err, tt.wantErr)
			}
		})
	}
}

func TestHexColorValidation(t *testing.T) {
	tests := []struct {
		name    string
		color   string
		wantErr bool
	}{
		// Valid colors
		{"valid uppercase", "#FF5733", false},
		{"valid lowercase", "#ff5733", false},
		{"valid mixed", "#Ff5733", false},
		{"empty (allowed)", "", false},

		// Invalid colors
		{"no hash", "FF5733", true},
		{"too short", "#FFF", true},
		{"too long", "#FF5733FF", true},
		{"invalid chars", "#GGGGGG", true},
		{"with spaces", "# FF5733", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.HexColor(tt.color)
			if (err != nil) != tt.wantErr {
				t.Errorf("HexColor(%q) error = %v, wantErr %v", tt.color, err, tt.wantErr)
			}
		})
	}
}

func TestProjectNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		project string
		wantErr bool
	}{
		{"valid short", "work", false},
		{"valid with spaces", "My Project", false},
		{"valid with emoji", "Work 🎯", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 200)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.ProjectName(tt.project)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProjectName(%q) error = %v, wantErr %v", tt.project, err, tt.wantErr)
			}
		})
	}
}
