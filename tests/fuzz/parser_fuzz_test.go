package fuzz

import (
	"testing"

	"github.com/manav03panchal/humantime/internal/parser"
)

// FuzzParseDeadline tests the deadline parser with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzParseDeadline -fuzztime=30s
func FuzzParseDeadline(f *testing.F) {
	// Seed corpus with valid inputs
	seeds := []string{
		"friday 5pm",
		"+5m",
		"+1h30m",
		"tomorrow",
		"tomorrow at 3pm",
		"in 5 minutes",
		"1 minute",
		"2 hours",
		"next monday",
		"yesterday",
		"2 hours ago",
		"9am",
		"5:30pm",
		"17:30",
		"2026-01-15",
		"2026-01-15 14:30",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// ParseDeadline should never panic, regardless of input
		result := parser.ParseDeadline(input)
		// We don't care about the result, just that it doesn't panic
		_ = result
	})
}

// FuzzParseDuration tests the duration parser with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzParseDuration -fuzztime=30s
func FuzzParseDuration(f *testing.F) {
	// Seed corpus with valid inputs
	seeds := []string{
		"1h",
		"30m",
		"1h30m",
		"90m",
		"2h",
		"45s",
		"1h30m45s",
		"1 hour",
		"30 minutes",
		"2 hours 30 minutes",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// ParseDuration should never panic, regardless of input
		_ = parser.ParseDuration(input)
	})
}

// FuzzParseTimestamp tests the timestamp parser with fuzz inputs.
// Run with: go test ./tests/fuzz/... -fuzz=FuzzParseTimestamp -fuzztime=30s
func FuzzParseTimestamp(f *testing.F) {
	// Seed corpus with valid inputs
	seeds := []string{
		"9am",
		"5pm",
		"14:30",
		"2:30pm",
		"yesterday at 3pm",
		"2 hours ago",
		"now",
		"2026-01-15",
		"2026-01-15T14:30:00",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// ParseTimestamp should never panic, regardless of input
		_ = parser.ParseTimestamp(input)
	})
}
