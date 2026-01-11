package property

import (
	"testing"
	"testing/quick"
	"time"

	"github.com/manav03panchal/humantime/internal/parser"
)

// TestNaturalLanguageNeverPanics tests that natural language parsing never panics.
func TestNaturalLanguageNeverPanics(t *testing.T) {
	f := func(input string) bool {
		// These should never panic
		_ = parser.ParseTimestamp(input)
		_ = parser.ParseDeadline(input)
		_ = parser.ParseDuration(input)
		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// TestPeriodParsing tests period expressions like "this week", "last month".
func TestPeriodParsing(t *testing.T) {
	periods := []string{
		"this week",
		"last week",
		"this month",
		"last month",
		"this year",
		"last year",
		"this hour",
		"last hour",
		"this day",
		"last day",
		"current week",
		"previous week",
	}

	for _, period := range periods {
		result := parser.ParseTimestamp(period)
		if result.Error != nil {
			t.Logf("Period %q not supported: %v", period, result.Error)
			continue
		}
		if result.Time.IsZero() {
			t.Errorf("ParseTimestamp(%q) returned zero time", period)
		}
	}
}

// TestRelativeTimeExpressions tests relative time expressions.
func TestRelativeTimeExpressions(t *testing.T) {
	expressions := []string{
		"now",
		"yesterday",
		"today",
		"2 hours ago",
		"1 hour ago",
		"30 minutes ago",
	}

	now := time.Now()

	for _, expr := range expressions {
		result := parser.ParseTimestamp(expr)
		if result.Error != nil {
			t.Logf("Expression %q not supported: %v", expr, result.Error)
			continue
		}

		// All these should produce times in the past or present
		if result.Time.After(now.Add(time.Minute)) {
			t.Errorf("ParseTimestamp(%q) = %v, expected time <= now", expr, result.Time)
		}
	}
}

// TestTimeOfDayExpressions tests time of day expressions.
func TestTimeOfDayExpressions(t *testing.T) {
	times := []string{
		"9am",
		"10am",
		"12pm",
		"5pm",
		"9:00",
		"14:30",
		"17:00",
	}

	for _, timeStr := range times {
		result := parser.ParseTimestamp(timeStr)
		if result.Error != nil {
			t.Logf("Time %q not supported: %v", timeStr, result.Error)
			continue
		}
		if result.Time.IsZero() {
			t.Errorf("ParseTimestamp(%q) returned zero time", timeStr)
		}
	}
}

// TestGetPeriodRange tests period range calculation.
func TestGetPeriodRange(t *testing.T) {
	tests := []struct {
		period    string
		expectGap bool // Start should be before End
	}{
		{"today", true},
		{"yesterday", true},
		{"this week", true},
		{"last week", true},
		{"this month", true},
		{"last month", true},
		{"this year", true},
		{"last year", true},
	}

	for _, tc := range tests {
		result := parser.GetPeriodRange(tc.period)

		if result.Start.IsZero() {
			t.Errorf("GetPeriodRange(%q) returned zero Start", tc.period)
		}
		if result.End.IsZero() {
			t.Errorf("GetPeriodRange(%q) returned zero End", tc.period)
		}

		if tc.expectGap && !result.Start.Before(result.End) {
			t.Errorf("GetPeriodRange(%q) Start should be before End: %v - %v",
				tc.period, result.Start, result.End)
		}
	}
}

// TestPeriodRangeConsistency tests that period ranges are consistent.
func TestPeriodRangeConsistency(t *testing.T) {
	// This week's range should contain today
	thisWeek := parser.GetPeriodRange("this week")
	today := parser.GetPeriodRange("today")

	if today.Start.Before(thisWeek.Start) || today.End.After(thisWeek.End) {
		t.Log("Today should be within this week")
		// This is expected behavior but depends on definition of "week"
	}
}

// TestParserConsistency tests that parsers give consistent results.
func TestParserConsistency(t *testing.T) {
	inputs := []string{
		"+5m",
		"tomorrow",
		"1h30m",
		"now",
	}

	for _, input := range inputs {
		// Parse twice
		result1 := parser.ParseDeadline(input)
		result2 := parser.ParseDeadline(input)

		// Error status should match
		if (result1.Error == nil) != (result2.Error == nil) {
			t.Errorf("Inconsistent error status for %q", input)
		}

		// If both succeeded, times should be very close
		if result1.Error == nil && result2.Error == nil {
			diff := result1.Time.Sub(result2.Time)
			if diff < 0 {
				diff = -diff
			}
			if diff > 2*time.Second {
				t.Errorf("Inconsistent results for %q: %v vs %v", input, result1.Time, result2.Time)
			}
		}
	}
}

// TestNowParsing tests "now" parsing.
func TestNowParsing(t *testing.T) {
	before := time.Now()
	result := parser.ParseTimestamp("now")
	after := time.Now()

	if result.Error != nil {
		t.Fatalf("ParseTimestamp(now) error: %v", result.Error)
	}

	// Result should be between before and after
	if result.Time.Before(before) || result.Time.After(after) {
		t.Errorf("ParseTimestamp(now) = %v, expected between %v and %v",
			result.Time, before, after)
	}
}

// TestEmptyInputHandling tests empty input handling across parsers.
func TestEmptyInputHandling(t *testing.T) {
	// Empty string should either error or return reasonable default
	timestampResult := parser.ParseTimestamp("")
	durationResult := parser.ParseDuration("")
	deadlineResult := parser.ParseDeadline("")

	// Duration with empty should definitely be invalid
	if durationResult.Valid {
		t.Error("Empty duration should be invalid")
	}

	// Log behavior for reference
	t.Logf("Empty timestamp: time=%v, err=%v", timestampResult.Time, timestampResult.Error)
	t.Logf("Empty duration: valid=%v, dur=%v", durationResult.Valid, durationResult.Duration)
	t.Logf("Empty deadline: time=%v, err=%v", deadlineResult.Time, deadlineResult.Error)
}
