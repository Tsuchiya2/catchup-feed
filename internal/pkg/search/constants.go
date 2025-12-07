// Package search provides utilities for parsing and escaping search keywords.
package search

import "time"

// Search configuration constants
const (
	// DefaultMaxKeywordCount is the maximum number of keywords allowed in a search query.
	// This limit prevents DoS attacks and excessive query complexity.
	DefaultMaxKeywordCount = 10

	// DefaultMaxKeywordLength is the maximum length (in runes) for each keyword.
	// This prevents memory exhaustion and excessive database load.
	DefaultMaxKeywordLength = 100

	// DefaultSearchTimeout is the default timeout for search queries.
	// This prevents long-running queries from blocking database connections.
	DefaultSearchTimeout = 5 * time.Second
)
