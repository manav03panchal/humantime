package parser

import (
	"regexp"
	"strings"
	"time"

	"github.com/markusmobius/go-dateparser"
)

// TimestampResult holds the parsed timestamp and any error.
type TimestampResult struct {
	Time  time.Time
	Error error
}

// periodRegex matches period expressions like "this week", "last month".
var periodRegex = regexp.MustCompile(`(?i)^(this|current|last|previous)\s+(hour|day|week|month|quarter|year)$`)

// ParseTimestamp parses a natural language timestamp expression.
func ParseTimestamp(input string) TimestampResult {
	input = strings.TrimSpace(input)
	if input == "" || strings.ToLower(input) == "now" {
		return TimestampResult{Time: time.Now()}
	}

	// Check for period expressions first
	if match := periodRegex.FindStringSubmatch(input); match != nil {
		return parsePeriod(match[1], match[2])
	}

	// Use go-dateparser for natural language parsing
	cfg := &dateparser.Configuration{
		CurrentTime: time.Now(),
	}

	result, err := dateparser.Parse(cfg, input)
	if err != nil {
		return TimestampResult{Error: err}
	}

	return TimestampResult{Time: result.Time}
}

// parsePeriod handles period expressions like "this week", "last month".
func parsePeriod(modifier, period string) TimestampResult {
	now := time.Now()
	modifier = strings.ToLower(modifier)
	period = strings.ToLower(period)

	var t time.Time

	switch period {
	case "hour":
		t = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
		if modifier == "last" || modifier == "previous" {
			t = t.Add(-time.Hour)
		}

	case "day":
		t = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if modifier == "last" || modifier == "previous" {
			t = t.AddDate(0, 0, -1)
		}

	case "week":
		// Go to start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday
		}
		t = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		if modifier == "last" || modifier == "previous" {
			t = t.AddDate(0, 0, -7)
		}

	case "month":
		t = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		if modifier == "last" || modifier == "previous" {
			t = t.AddDate(0, -1, 0)
		}

	case "quarter":
		quarter := (int(now.Month()) - 1) / 3
		t = time.Date(now.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, now.Location())
		if modifier == "last" || modifier == "previous" {
			t = t.AddDate(0, -3, 0)
		}

	case "year":
		t = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		if modifier == "last" || modifier == "previous" {
			t = t.AddDate(-1, 0, 0)
		}

	default:
		return TimestampResult{Time: now}
	}

	return TimestampResult{Time: t}
}

// ParseTimeRange parses a time range and returns start and end times.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// GetPeriodRange returns the start and end of a period.
func GetPeriodRange(period string) TimeRange {
	now := time.Now()
	period = strings.ToLower(period)

	var start, end time.Time

	switch {
	case strings.HasPrefix(period, "today"):
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 0, 1)

	case strings.HasPrefix(period, "yesterday"):
		start = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 0, 1)

	case strings.Contains(period, "week"):
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if strings.HasPrefix(period, "this") || strings.HasPrefix(period, "current") {
			start = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
			end = start.AddDate(0, 0, 7)
		} else { // last week
			start = time.Date(now.Year(), now.Month(), now.Day()-weekday+1-7, 0, 0, 0, 0, now.Location())
			end = start.AddDate(0, 0, 7)
		}

	case strings.Contains(period, "month"):
		if strings.HasPrefix(period, "this") || strings.HasPrefix(period, "current") {
			start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			end = start.AddDate(0, 1, 0)
		} else { // last month
			start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
			end = start.AddDate(0, 1, 0)
		}

	case strings.Contains(period, "year"):
		if strings.HasPrefix(period, "this") || strings.HasPrefix(period, "current") {
			start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
			end = start.AddDate(1, 0, 0)
		} else { // last year
			start = time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location())
			end = start.AddDate(1, 0, 0)
		}

	default:
		// Default to today
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 0, 1)
	}

	return TimeRange{Start: start, End: end}
}
