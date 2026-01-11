package security

import (
	"strings"
	"testing"

	"github.com/manav03panchal/humantime/internal/logging"
)

// TestWebhookURLMasking tests that webhook URLs are properly masked.
func TestWebhookURLMasking(t *testing.T) {
	sensitiveURLs := []string{
		"https://hooks.slack.com/services/T00/B00/xxxxxxxxxxxx",
		"https://discord.com/api/webhooks/123456/abcdef-secret",
		"https://api.example.com/webhook?token=secrettoken123",
		"https://notify.example.com/hook/user/long-secret-path",
	}

	for _, url := range sensitiveURLs {
		masked := logging.MaskURL(url)

		// Should contain masking
		if !strings.Contains(masked, "*") {
			t.Errorf("URL %q should be masked", url)
		}

		// Should not expose full secret parts
		if strings.Contains(masked, "secrettoken123") {
			t.Errorf("Masked URL should not contain full secrets")
		}
		if strings.Contains(masked, "xxxxxxxxxxxx") {
			t.Errorf("Masked URL should not contain full webhook identifiers")
		}
	}
}

// TestAPIKeyMasking tests API key masking.
func TestAPIKeyMasking(t *testing.T) {
	keys := []string{
		"sk_live_51J5vQ2KL2sFjH7nJ3Yw",
		"api_key_very_secret_1234567890",
		"ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	}

	for _, key := range keys {
		masked := logging.MaskValue(key)

		if masked == key {
			t.Errorf("Key %q should be masked", key)
		}
		if !strings.Contains(masked, "*") {
			t.Errorf("Masked key should contain asterisks, got %q", masked)
		}
	}
}

// TestSensitiveFieldDetection tests detection of sensitive fields.
func TestSensitiveFieldDetection(t *testing.T) {
	sensitiveFields := []string{
		"token",
		"secret",
		"password",
		"api_key",
		"apikey",
		"auth",
		"credential",
		"private_key",
		"access_token",
		"refresh_token",
		"bearer",
	}

	for _, field := range sensitiveFields {
		if !logging.IsSensitiveField(field) {
			t.Errorf("Field %q should be detected as sensitive", field)
		}
	}
}

// TestNonSensitiveFields tests that normal fields are not masked.
func TestNonSensitiveFields(t *testing.T) {
	normalFields := []string{
		"name",
		"project",
		"duration",
		"description",
		"start_time",
		"end_time",
		"entry_id",
		"tags",
	}

	for _, field := range normalFields {
		if logging.IsSensitiveField(field) {
			t.Errorf("Field %q should not be detected as sensitive", field)
		}
	}
}

// TestMaskSensitiveDataMap tests map-based masking.
func TestMaskSensitiveDataMap(t *testing.T) {
	data := map[string]string{
		"username":     "john",
		"password":     "secret123",
		"token":        "abc123xyz",
		"project_name": "my-project",
		"api_key":      "key-12345",
	}

	masked := logging.MaskSensitiveData(data)

	// Check non-sensitive preserved
	if masked["username"] != "john" {
		t.Error("username should be preserved")
	}
	if masked["project_name"] != "my-project" {
		t.Error("project_name should be preserved")
	}

	// Check sensitive masked
	if masked["password"] != "***" {
		t.Errorf("password should be masked, got %q", masked["password"])
	}
	if masked["token"] != "***" {
		t.Errorf("token should be masked, got %q", masked["token"])
	}
	if masked["api_key"] != "***" {
		t.Errorf("api_key should be masked, got %q", masked["api_key"])
	}
}

// TestURLQueryParameterMasking tests query parameter handling.
func TestURLQueryParameterMasking(t *testing.T) {
	urlsWithSecrets := []string{
		"https://api.com/hook?token=secret123&other=value",
		"https://api.com/hook?api_key=mykey&id=123",
		"https://api.com/callback?auth=bearer_token",
	}

	for _, url := range urlsWithSecrets {
		masked := logging.MaskURL(url)

		// Full URL should be masked (we mask entire URL for simplicity)
		if !strings.Contains(masked, "*") {
			t.Errorf("URL with secrets %q should be masked", url)
		}
	}
}

// TestMaskingDoesNotLeak tests that masking doesn't accidentally leak.
func TestMaskingDoesNotLeak(t *testing.T) {
	sensitiveValue := "super_secret_password_12345"

	masked := logging.MaskValue(sensitiveValue)

	// Check for any partial leakage
	if strings.Contains(masked, "password") {
		t.Error("Masked value should not contain 'password'")
	}
	if strings.Contains(masked, "12345") {
		t.Error("Masked value should not contain numeric parts")
	}
	if strings.Contains(masked, "super") {
		t.Error("Masked value should not contain any original content")
	}
}

// TestMaskingConsistency tests that same input gives same output.
func TestMaskingConsistency(t *testing.T) {
	input := "consistent_secret_value"

	result1 := logging.MaskValue(input)
	result2 := logging.MaskValue(input)

	if result1 != result2 {
		t.Errorf("MaskValue should be consistent: %q != %q", result1, result2)
	}
}

// TestMaskURLPreservesHost tests that host portion is partially visible.
func TestMaskURLPreservesHost(t *testing.T) {
	url := "https://hooks.example.com/very/long/path/to/webhook/secret"
	masked := logging.MaskURL(url)

	// Should preserve beginning for debugging
	if !strings.HasPrefix(masked, "https://") {
		t.Error("Masked URL should preserve scheme")
	}
}

// TestEmptyValueMasking tests empty value handling.
func TestEmptyValueMasking(t *testing.T) {
	empty := ""
	masked := logging.MaskValue(empty)

	// Empty values return empty (implementation specific)
	if masked != "" {
		t.Logf("Empty value returns: %q", masked)
	}
}

// TestLogMessageSanitization tests log message safety.
func TestLogMessageSanitization(t *testing.T) {
	// Ensure log messages with embedded secrets are safe
	data := map[string]string{
		"webhook_url": "https://secret.example.com/hook?token=abc",
		"auth_header": "Bearer sk_live_xxxxx",
	}

	masked := logging.MaskSensitiveData(data)

	// auth_header is definitely sensitive
	if masked["auth_header"] != "***" {
		t.Error("auth_header should be masked")
	}
}

// TestFieldNameVariations tests various field naming conventions.
func TestFieldNameVariations(t *testing.T) {
	// These should all be detected as sensitive via substring matching
	variations := []string{
		"auth_token",
		"bearer_token",
		"access_token",
		"refresh_token",
	}

	for _, field := range variations {
		if !logging.IsSensitiveField(field) {
			t.Errorf("Field %q should be detected as sensitive", field)
		}
	}
}

// TestMaskPartialPreservesPrefix tests partial masking.
func TestMaskPartialPreservesPrefix(t *testing.T) {
	value := "sk_live_abcdef123456"
	masked := logging.MaskPartial(value, 7)

	if !strings.HasPrefix(masked, "sk_live") {
		t.Error("MaskPartial should preserve first 7 characters")
	}
	if !strings.Contains(masked, "*") {
		t.Error("MaskPartial should contain mask characters")
	}
	if len(masked) < len("sk_live") {
		t.Error("Masked result should not be shorter than prefix")
	}
}

// TestMaskArgsSlice tests argument list masking.
func TestMaskArgsSlice(t *testing.T) {
	args := []any{
		"username", "john",
		"password", "secret123",
		"project", "my-project",
	}

	masked := logging.MaskArgs(args)

	// Check password was masked
	if masked[3].(string) == "secret123" {
		t.Error("password value should be masked")
	}

	// Check project was preserved
	if masked[5].(string) != "my-project" {
		t.Error("project value should be preserved")
	}
}
