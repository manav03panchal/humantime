package logging

import (
	"regexp"
	"strings"
)

const (
	// MaskChar is the character used for masking.
	MaskChar = "*"
	// URLMaskLength is how many characters to show before masking URLs.
	URLMaskLength = 30
	// DefaultMaskLength is how many mask characters to show.
	DefaultMaskLength = 3
)

// SensitiveFields contains field names that should be masked.
var SensitiveFields = map[string]bool{
	"token":          true,
	"secret":         true,
	"password":       true,
	"key":            true,
	"api_key":        true,
	"apikey":         true,
	"access_token":   true,
	"refresh_token":  true,
	"auth":           true,
	"authorization":  true,
	"bearer":         true,
	"credential":     true,
	"credentials":    true,
	"private":        true,
	"private_key":    true,
}

// urlPattern matches HTTP(S) URLs.
var urlPattern = regexp.MustCompile(`https?://[^\s"']+`)

// MaskURL masks a URL, showing only the first URLMaskLength characters.
func MaskURL(url string) string {
	if len(url) <= URLMaskLength {
		return url
	}
	return url[:URLMaskLength] + strings.Repeat(MaskChar, DefaultMaskLength)
}

// MaskValue masks a sensitive value completely.
func MaskValue(value string) string {
	if value == "" {
		return ""
	}
	return strings.Repeat(MaskChar, min(len(value), 8))
}

// MaskPartial masks a value but shows the first few characters.
func MaskPartial(value string, showChars int) string {
	if len(value) <= showChars {
		return strings.Repeat(MaskChar, len(value))
	}
	return value[:showChars] + strings.Repeat(MaskChar, DefaultMaskLength)
}

// IsSensitiveField checks if a field name indicates sensitive data.
func IsSensitiveField(fieldName string) bool {
	lower := strings.ToLower(fieldName)

	// Check exact match
	if SensitiveFields[lower] {
		return true
	}

	// Check if contains sensitive keywords
	for keyword := range SensitiveFields {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	return false
}

// MaskString scans a string for sensitive patterns and masks them.
func MaskString(s string) string {
	// Mask URLs
	s = urlPattern.ReplaceAllStringFunc(s, func(url string) string {
		// Don't mask localhost URLs
		if strings.Contains(url, "localhost") || strings.Contains(url, "127.0.0.1") {
			return url
		}
		return MaskURL(url)
	})

	return s
}

// MaskArgs masks sensitive values in a slice of logging arguments.
// Arguments are expected in key-value pairs: key1, value1, key2, value2, ...
func MaskArgs(args []any) []any {
	if len(args) < 2 {
		return args
	}

	result := make([]any, len(args))
	copy(result, args)

	for i := 0; i < len(result)-1; i += 2 {
		key, ok := result[i].(string)
		if !ok {
			continue
		}

		if IsSensitiveField(key) {
			if strVal, ok := result[i+1].(string); ok {
				result[i+1] = MaskValue(strVal)
			} else {
				result[i+1] = strings.Repeat(MaskChar, 8)
			}
		}
	}

	return result
}

// MaskMap masks sensitive values in a map.
func MaskMap(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))

	for key, value := range m {
		if IsSensitiveField(key) {
			if strVal, ok := value.(string); ok {
				result[key] = MaskValue(strVal)
			} else {
				result[key] = strings.Repeat(MaskChar, 8)
			}
		} else if strVal, ok := value.(string); ok {
			// Mask URLs in string values
			result[key] = MaskString(strVal)
		} else if nestedMap, ok := value.(map[string]any); ok {
			// Recursively mask nested maps
			result[key] = MaskMap(nestedMap)
		} else {
			result[key] = value
		}
	}

	return result
}

// SanitizeLogMessage removes or masks sensitive data from a log message.
func SanitizeLogMessage(msg string) string {
	return MaskString(msg)
}

// MaskSensitiveData masks sensitive values in a string map.
func MaskSensitiveData(m map[string]string) map[string]string {
	if m == nil {
		return make(map[string]string)
	}

	result := make(map[string]string, len(m))
	for key, value := range m {
		if IsSensitiveField(key) {
			result[key] = "***"
		} else {
			result[key] = value
		}
	}
	return result
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
