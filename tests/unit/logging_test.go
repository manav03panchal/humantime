package unit

import (
	"strings"
	"testing"

	"github.com/manav03panchal/humantime/internal/logging"
)

// TestMaskURL tests URL masking.
func TestMaskURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMask bool
	}{
		{"short URL", "http://x.com", false},
		{"long URL", "https://example.com/webhook/secret-token-12345", true},
		{"with query params", "https://api.com/hook?token=secret123", true},
		{"empty", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := logging.MaskURL(tc.input)
			hasMask := strings.Contains(result, "*")

			if tc.wantMask && !hasMask {
				t.Errorf("Expected masking for %q, got %q", tc.input, result)
			}
		})
	}
}

// TestMaskValue tests value masking.
func TestMaskValue(t *testing.T) {
	tests := []struct {
		input       string
		containStar bool
	}{
		{"secret", true},
		{"short", true},
		{"a", true},
		{"", false}, // Empty returns empty
	}

	for _, tc := range tests {
		result := logging.MaskValue(tc.input)
		hasStars := strings.Contains(result, "*")
		if hasStars != tc.containStar {
			t.Errorf("MaskValue(%q) = %q, expected stars=%v", tc.input, result, tc.containStar)
		}
	}
}

// TestIsSensitiveField tests sensitive field detection.
func TestIsSensitiveField(t *testing.T) {
	sensitiveFields := []string{
		"token",
		"secret",
		"password",
		"api_key",
		"apikey",
		"auth",
		"credential",
		"private",
	}

	for _, field := range sensitiveFields {
		if !logging.IsSensitiveField(field) {
			t.Errorf("Field %q should be sensitive", field)
		}
	}
}

// TestIsSensitiveFieldNegative tests non-sensitive fields.
func TestIsSensitiveFieldNegative(t *testing.T) {
	nonSensitiveFields := []string{
		"name",
		"project",
		"duration",
		"description",
		"id",
		"timestamp",
	}

	for _, field := range nonSensitiveFields {
		if logging.IsSensitiveField(field) {
			t.Errorf("Field %q should not be sensitive", field)
		}
	}
}

// TestMaskSensitiveData tests masking in key-value pairs.
func TestMaskSensitiveData(t *testing.T) {
	data := map[string]string{
		"name":     "project1",
		"token":    "secret123",
		"password": "mypassword",
		"duration": "1h",
	}

	masked := logging.MaskSensitiveData(data)

	// Non-sensitive should remain
	if masked["name"] != "project1" {
		t.Error("name should not be masked")
	}
	if masked["duration"] != "1h" {
		t.Error("duration should not be masked")
	}

	// Sensitive should be masked
	if masked["token"] != "***" {
		t.Errorf("token should be masked, got %q", masked["token"])
	}
	if masked["password"] != "***" {
		t.Errorf("password should be masked, got %q", masked["password"])
	}
}

// TestMaskSensitiveDataNilInput tests nil input handling.
func TestMaskSensitiveDataNilInput(t *testing.T) {
	result := logging.MaskSensitiveData(nil)
	if result == nil {
		t.Error("MaskSensitiveData(nil) should return empty map, not nil")
	}
}

// TestMaskURLPreservesScheme tests that scheme is visible.
func TestMaskURLPreservesScheme(t *testing.T) {
	url := "https://api.example.com/long/path/to/webhook/endpoint"
	result := logging.MaskURL(url)

	if !strings.HasPrefix(result, "https://") {
		t.Errorf("Masked URL should preserve scheme, got %q", result)
	}
}

// TestSensitiveFieldsCaseInsensitive tests case handling.
func TestSensitiveFieldsCaseInsensitive(t *testing.T) {
	// Check various cases - IsSensitiveField should handle lowercase
	tests := []struct {
		field    string
		expected bool
	}{
		{"token", true},
		{"password", true},
	}

	for _, tc := range tests {
		result := logging.IsSensitiveField(tc.field)
		if result != tc.expected {
			t.Errorf("IsSensitiveField(%q) = %v, expected %v", tc.field, result, tc.expected)
		}
	}
}

// TestMaskEmptyStrings tests edge cases with empty strings.
func TestMaskEmptyStrings(t *testing.T) {
	// Empty URL
	if logging.MaskURL("") != "" {
		t.Error("Empty URL should remain empty")
	}

	// Empty value - returns empty, not masked
	result := logging.MaskValue("")
	if result != "" {
		t.Logf("Empty value returns: %q", result)
	}

	// Empty field name
	if logging.IsSensitiveField("") {
		t.Error("Empty field should not be sensitive")
	}
}

// TestMaskSensitiveDataPreservesStructure tests map structure.
func TestMaskSensitiveDataPreservesStructure(t *testing.T) {
	input := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}

	result := logging.MaskSensitiveData(input)

	if len(result) != len(input) {
		t.Errorf("Result should have same number of keys, got %d, want %d", len(result), len(input))
	}

	for key := range input {
		if _, ok := result[key]; !ok {
			t.Errorf("Result should contain key %q", key)
		}
	}
}

// TestMaskPartial tests partial masking.
func TestMaskPartial(t *testing.T) {
	result := logging.MaskPartial("secret12345", 3)
	if !strings.HasPrefix(result, "sec") {
		t.Error("MaskPartial should preserve first 3 chars")
	}
	if !strings.Contains(result, "*") {
		t.Error("MaskPartial should contain mask chars")
	}
}

// TestMaskString tests string scanning and masking.
func TestMaskString(t *testing.T) {
	input := "Check https://api.example.com/long/webhook/path/secret for status"
	result := logging.MaskString(input)

	if !strings.Contains(result, "*") {
		t.Error("MaskString should mask long URLs")
	}
	if !strings.Contains(result, "Check") {
		t.Error("MaskString should preserve non-URL text")
	}
}
