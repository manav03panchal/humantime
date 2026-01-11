package fuzz

import (
	"testing"

	"github.com/manav03panchal/humantime/internal/logging"
)

// FuzzMaskURL tests URL masking with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzMaskURL -fuzztime=30s
func FuzzMaskURL(f *testing.F) {
	seeds := []string{
		"https://example.com/webhook/secret",
		"http://localhost:8080",
		"https://api.com/hook?token=secret123",
		"ftp://invalid.com",
		"not-a-url",
		string(make([]byte, 10000)),
		"",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// MaskURL should never panic
		_ = logging.MaskURL(input)
	})
}

// FuzzMaskValue tests value masking with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzMaskValue -fuzztime=30s
func FuzzMaskValue(f *testing.F) {
	seeds := []string{
		"secret123",
		"very-long-secret-value-that-should-be-masked",
		"",
		"a",
		string(make([]byte, 10000)),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// MaskValue should never panic
		_ = logging.MaskValue(input)
	})
}

// FuzzIsSensitiveField tests sensitive field detection with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzIsSensitiveField -fuzztime=30s
func FuzzIsSensitiveField(f *testing.F) {
	seeds := []string{
		"token",
		"password",
		"api_key",
		"name",
		"project",
		"",
		"TOKEN",
		"pAsSwOrD",
		string(make([]byte, 1000)),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// IsSensitiveField should never panic
		_ = logging.IsSensitiveField(input)
	})
}

// FuzzMaskString tests string masking with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzMaskString -fuzztime=30s
func FuzzMaskString(f *testing.F) {
	seeds := []string{
		"Check https://api.example.com/webhook for status",
		"No URLs here",
		"Multiple https://a.com and https://b.com URLs",
		"",
		string(make([]byte, 10000)),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// MaskString should never panic
		_ = logging.MaskString(input)
	})
}

// FuzzMaskPartial tests partial masking with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzMaskPartial -fuzztime=30s
func FuzzMaskPartial(f *testing.F) {
	f.Add("secret12345", 3)
	f.Add("short", 100)
	f.Add("", 5)
	f.Add("test", 0)
	f.Add("test", -1)
	f.Add(string(make([]byte, 10000)), 50)

	f.Fuzz(func(t *testing.T, input string, showChars int) {
		// MaskPartial should never panic
		if showChars < 0 {
			showChars = 0
		}
		_ = logging.MaskPartial(input, showChars)
	})
}
