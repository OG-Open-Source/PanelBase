package models

import (
	"fmt"
	"strings"
	"time"
)

// RFC3339Time wraps time.Time to force RFC3339 format in JSON.
type RFC3339Time time.Time

// MarshalJSON implements the json.Marshaler interface.
// The time is formatted using the time.RFC3339 layout (second precision).
func (t RFC3339Time) MarshalJSON() ([]byte, error) {
	// Handle zero time explicitly to return null?
	// Or return ""? Let's format the zero time as well for consistency unless required otherwise.
	timeStr := fmt.Sprintf("\"%s\"", time.Time(t).UTC().Format(time.RFC3339))
	return []byte(timeStr), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The time is expected to be in RFC3339 or RFC3339Nano format (for compatibility).
func (t *RFC3339Time) UnmarshalJSON(data []byte) error {
	// Trim quotes and handle null/empty string
	str := strings.Trim(string(data), "\"")
	if str == "null" || str == "" {
		*t = RFC3339Time(time.Time{}) // Set zero value
		return nil
	}

	// Try parsing RFC3339 first (target format)
	parsedTime, err := time.Parse(time.RFC3339, str)
	if err != nil {
		// If RFC3339 fails, try RFC3339Nano (for backward compatibility with existing data)
		parsedTime, err = time.Parse(time.RFC3339Nano, str)
		if err != nil {
			return fmt.Errorf("invalid time format, expected RFC3339 or RFC3339Nano: %s, error: %w", str, err)
		}
	}

	// Assign the parsed time (implicitly UTC if Z or offset was present)
	*t = RFC3339Time(parsedTime)
	return nil
}

// Time returns the underlying time.Time value.
// Useful for accessing time.Time methods directly.
func (t RFC3339Time) Time() time.Time {
	return time.Time(t)
}

// IsZero checks if the underlying time is the zero value.
func (t RFC3339Time) IsZero() bool {
	return time.Time(t).IsZero()
}
