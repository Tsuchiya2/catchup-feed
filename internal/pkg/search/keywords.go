package search

import (
	"fmt"
	"strings"
)

// ParseKeywords parses space-separated keywords from input string.
// This function splits the input on whitespace, trims each keyword,
// filters empty strings, and validates against maxCount and maxLength limits.
//
// The function follows these steps:
//   - Splits input by whitespace using strings.Fields()
//   - Trims each keyword (whitespace removal handled by Fields)
//   - Filters empty strings (handled by Fields)
//   - Validates keyword count against maxCount
//   - Validates each keyword length against maxLength
//
// Parameters:
//   - input: Space-separated keywords string
//   - maxCount: Maximum number of keywords allowed
//   - maxLength: Maximum length for each keyword
//
// Returns:
//   - []string: List of parsed and validated keywords
//   - error: Descriptive error if validation fails
//
// Error conditions:
//   - Input is empty or whitespace-only
//   - Too many keywords (> maxCount)
//   - Any keyword exceeds maxLength
//
// Example:
//
//	keywords, err := ParseKeywords("Go React", 10, 100)
//	if err != nil {
//	    log.Error("Failed to parse keywords: %v", err)
//	}
//	// keywords = ["Go", "React"]
//
// Edge cases handled:
//   - Multiple spaces between keywords: "Go  React" → ["Go", "React"]
//   - Leading/trailing spaces: "  Go React  " → ["Go", "React"]
//   - Unicode/Japanese keywords: "Go 日本語" → ["Go", "日本語"]
func ParseKeywords(input string, maxCount int, maxLength int) ([]string, error) {
	// Validate input is not empty or whitespace-only
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("keywords cannot be empty")
	}

	// Split by whitespace (handles multiple spaces automatically)
	keywords := strings.Fields(trimmed)

	// Validate keyword count
	if len(keywords) > maxCount {
		return nil, fmt.Errorf("too many keywords: got %d, maximum %d allowed", len(keywords), maxCount)
	}

	// Validate each keyword length
	for i, keyword := range keywords {
		// strings.Fields already trimmed, but double-check for consistency
		keyword = strings.TrimSpace(keyword)
		keywords[i] = keyword

		// Validate length (use rune count for proper Unicode support)
		if len([]rune(keyword)) > maxLength {
			return nil, fmt.Errorf("keyword '%s' exceeds maximum length of %d characters", keyword, maxLength)
		}
	}

	return keywords, nil
}
