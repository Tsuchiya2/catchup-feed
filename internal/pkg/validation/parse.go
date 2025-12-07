package validation

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDateISO8601 parses a date string in ISO 8601 format.
// Supports: "2024-01-01" (date only) and "2024-01-01T10:00:00Z" (with time)
// Returns nil and no error for empty input (optional date).
// Returns error for invalid format.
//
// Parameters:
//   - input: Date string in ISO 8601 format or empty string
//
// Returns:
//   - *time.Time: Parsed date (nil if input is empty)
//   - error: nil if valid, descriptive error otherwise
//
// Supported formats:
//   - Date only: "2024-01-01"
//   - RFC3339: "2024-01-01T10:00:00Z"
//   - RFC3339 with timezone: "2024-01-01T10:00:00+09:00"
//
// Example:
//
//	date, err := ParseDateISO8601("2024-01-01")
//	if err != nil {
//	    log.Error("Invalid date: %v", err)
//	}
func ParseDateISO8601(input string) (*time.Time, error) {
	// Empty input is valid (optional field)
	if input == "" {
		return nil, nil
	}

	// Try date-only format first (2024-01-01)
	t, err := time.Parse("2006-01-02", input)
	if err == nil {
		return &t, nil
	}

	// Try RFC3339 format (2024-01-01T10:00:00Z)
	t, err = time.Parse(time.RFC3339, input)
	if err == nil {
		return &t, nil
	}

	// If both fail, return error
	return nil, fmt.Errorf("invalid date format '%s': expected ISO 8601 format (e.g., '2024-01-01' or '2024-01-01T10:00:00Z')", input)
}

// ValidateEnum validates if value is one of the allowed values.
// Returns error with field name if value is not in allowed list.
// Empty value returns nil (optional field).
//
// Parameters:
//   - value: Value to validate
//   - allowed: List of allowed values
//   - fieldName: Name of the field (for error messages)
//
// Returns:
//   - error: nil if valid, descriptive error otherwise
//
// Validation rules:
//   - Empty value is valid (optional field)
//   - Value must exactly match one of the allowed values (case-sensitive)
//   - Allowed list must not be empty
//
// Example:
//
//	err := ValidateEnum("RSS", []string{"RSS", "Webflow"}, "source_type")
//	if err != nil {
//	    log.Error("Invalid source type: %v", err)
//	}
func ValidateEnum(value string, allowed []string, fieldName string) error {
	// Empty value is valid (optional field)
	if value == "" {
		return nil
	}

	// Validate that allowed list is not empty
	if len(allowed) == 0 {
		return fmt.Errorf("allowed values list cannot be empty for field '%s'", fieldName)
	}

	// Check if value is in allowed list
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}

	// Value not found in allowed list
	return fmt.Errorf("invalid value '%s' for field '%s': must be one of [%s]",
		value, fieldName, strings.Join(allowed, ", "))
}

// ParseBool parses a boolean string.
// Accepts: "true", "false", "1", "0", "" (empty = nil)
// Returns pointer to bool, nil for empty input.
//
// Parameters:
//   - input: Boolean string or empty string
//
// Returns:
//   - *bool: Parsed boolean (nil if input is empty)
//   - error: nil if valid, descriptive error otherwise
//
// Accepted values:
//   - true: "true", "1"
//   - false: "false", "0"
//   - nil (optional): "" (empty string)
//
// Example:
//
//	enabled, err := ParseBool("true")
//	if err != nil {
//	    log.Error("Invalid boolean: %v", err)
//	}
//	if enabled != nil && *enabled {
//	    // enabled is true
//	}
func ParseBool(input string) (*bool, error) {
	// Empty input is valid (optional field)
	if input == "" {
		return nil, nil
	}

	// Use standard library strconv.ParseBool
	// Accepts: "1", "t", "T", "TRUE", "true", "True", "0", "f", "F", "FALSE", "false", "False"
	b, err := strconv.ParseBool(input)
	if err != nil {
		return nil, fmt.Errorf("invalid boolean value '%s': expected 'true', 'false', '1', or '0'", input)
	}

	return &b, nil
}
