// Package validate provides input validation helpers for the Humantime CLI.
package validate

import (
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/manav03panchal/humantime/internal/errors"
)

const (
	// MaxSIDLength is the maximum length for a simplified ID.
	MaxSIDLength = 32
	// MaxURLLength is the maximum length for a URL.
	MaxURLLength = 2048
	// MaxProjectNameLength is the maximum length for a project name.
	MaxProjectNameLength = 128
	// MaxNoteLength is the maximum length for a note.
	MaxNoteLength = 4096
)

// sidRegex validates simplified IDs (alphanumeric, dashes, underscores, periods).
var sidRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// SID validates a simplified ID.
func SID(sid string) error {
	if sid == "" {
		return errors.NewUserError("SID cannot be empty", "Provide a valid identifier")
	}
	if len(sid) > MaxSIDLength {
		return errors.NewUserErrorWithField("sid", sid,
			"SID too long",
			"SIDs must be 32 characters or fewer")
	}
	if !sidRegex.MatchString(sid) {
		return errors.NewUserErrorWithField("sid", sid,
			"Invalid SID format",
			"SIDs must start with a letter or number and contain only letters, numbers, dashes, underscores, or periods")
	}
	return nil
}

// ProjectName validates a project name.
func ProjectName(name string) error {
	if name == "" {
		return errors.NewUserError("Project name cannot be empty", "Provide a project name")
	}
	if utf8.RuneCountInString(name) > MaxProjectNameLength {
		return errors.NewUserErrorWithField("project", name,
			"Project name too long",
			"Project names must be 128 characters or fewer")
	}
	return nil
}

// Note validates a note/description.
func Note(note string) error {
	if utf8.RuneCountInString(note) > MaxNoteLength {
		return errors.NewUserError(
			"Note too long",
			"Notes must be 4096 characters or fewer")
	}
	return nil
}

// HexColor validates a hex color code.
func HexColor(color string) error {
	if color == "" {
		return nil // Empty is allowed (no color)
	}
	if !strings.HasPrefix(color, "#") {
		return errors.NewUserErrorWithField("color", color,
			"Invalid color format",
			"Use hex format like '#FF5733' or '#00FF00'")
	}
	hex := strings.TrimPrefix(color, "#")
	if len(hex) != 6 {
		return errors.NewUserErrorWithField("color", color,
			"Invalid color format",
			"Use 6-digit hex format like '#FF5733'")
	}
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return errors.NewUserErrorWithField("color", color,
				"Invalid hex character in color",
				"Use only hex digits (0-9, A-F)")
		}
	}
	return nil
}

// URL validates a URL for use as a webhook endpoint.
func URL(rawURL string) error {
	if rawURL == "" {
		return errors.NewUserError("URL cannot be empty", "Provide a valid URL")
	}
	if len(rawURL) > MaxURLLength {
		return errors.NewUserError("URL too long", "URLs must be 2048 characters or fewer")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return errors.NewUserErrorWithField("url", rawURL,
			"Invalid URL format",
			"Provide a valid URL starting with https://")
	}

	// Check scheme
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return errors.NewUserErrorWithField("url", rawURL,
			"Invalid URL scheme",
			"URLs must use https:// (or http:// for localhost)")
	}

	// Check hostname exists
	hostname := parsed.Hostname()
	if hostname == "" {
		return errors.NewUserErrorWithField("url", rawURL,
			"Invalid URL: missing hostname",
			"Provide a valid URL like https://example.com/webhook")
	}

	// Check for localhost (http allowed)
	isLocalhost := hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"

	// Require HTTPS for non-localhost
	if parsed.Scheme == "http" && !isLocalhost {
		return errors.NewUserErrorWithField("url", rawURL,
			"HTTP not allowed for external URLs",
			"Use https:// for security. HTTP is only allowed for localhost.")
	}

	// Check for internal IPs (SSRF protection)
	if !isLocalhost {
		if err := checkInternalIP(hostname); err != nil {
			return err
		}
	}

	return nil
}

// checkInternalIP checks if a hostname resolves to an internal IP.
func checkInternalIP(hostname string) error {
	// First check if it's a direct IP
	if ip := net.ParseIP(hostname); ip != nil {
		if isInternalIP(ip) {
			return errors.NewUserErrorWithField("url", hostname,
				"Internal IP addresses not allowed",
				"Webhook URLs must point to external services")
		}
		return nil
	}

	// Try to resolve hostname
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// DNS resolution failed - this is OK, the webhook will fail later
		return nil
	}

	for _, ip := range ips {
		if isInternalIP(ip) {
			return errors.NewUserErrorWithField("url", hostname,
				"Hostname resolves to internal IP",
				"Webhook URLs must point to external services")
		}
	}

	return nil
}

// isInternalIP checks if an IP is in a private/internal range.
func isInternalIP(ip net.IP) bool {
	// Private ranges
	privateRanges := []string{
		"10.0.0.0/8",     // RFC 1918
		"172.16.0.0/12",  // RFC 1918
		"192.168.0.0/16", // RFC 1918
		"127.0.0.0/8",    // Loopback (except explicit localhost check)
		"169.254.0.0/16", // Link-local
		"fc00::/7",       // IPv6 private
		"fe80::/10",      // IPv6 link-local
		"::1/128",        // IPv6 loopback
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// NonEmpty validates that a string is not empty.
func NonEmpty(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.NewUserError(
			field+" cannot be empty",
			"Provide a value for "+field)
	}
	return nil
}

// InRange validates that an integer is within a range.
func InRange(field string, value, min, max int) error {
	if value < min || value > max {
		return errors.NewUserErrorWithField(field, "",
			"Value out of range",
			"Must be between "+string(rune(min+'0'))+" and "+string(rune(max+'0')))
	}
	return nil
}
