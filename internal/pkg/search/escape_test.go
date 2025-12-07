package search

import (
	"strings"
	"testing"
)

func TestEscapeILIKE(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Normal string
		{
			name:     "normal string",
			input:    "Go",
			expected: "%Go%",
		},
		// With percent
		{
			name:     "with percent",
			input:    "100%",
			expected: `%100\%%`,
		},
		// With underscore
		{
			name:     "with underscore",
			input:    "my_var",
			expected: `%my\_var%`,
		},
		// With backslash
		{
			name:     "with backslash",
			input:    `path\file`,
			expected: `%path\\file%`,
		},
		// All special chars
		{
			name:     "all special chars",
			input:    `%_\`,
			expected: `%\%\_\\%`,
		},
		// Empty string
		{
			name:     "empty string",
			input:    "",
			expected: "%%",
		},
		// Unicode
		{
			name:     "unicode",
			input:    "日本語",
			expected: "%日本語%",
		},
		// Multiple percent signs
		{
			name:     "multiple percent signs",
			input:    "100% off 50%",
			expected: `%100\% off 50\%%`,
		},
		// Multiple underscores
		{
			name:     "multiple underscores",
			input:    "my_var_name",
			expected: `%my\_var\_name%`,
		},
		// Multiple backslashes
		{
			name:     "multiple backslashes",
			input:    `C:\Program Files\App`,
			expected: `%C:\\Program Files\\App%`,
		},
		// Mixed special characters
		{
			name:     "mixed special characters",
			input:    `100%_off\sale`,
			expected: `%100\%\_off\\sale%`,
		},
		// Already escaped backslash (should escape again)
		{
			name:     "already escaped backslash",
			input:    `\\`,
			expected: `%\\\\%`,
		},
		// Complex pattern
		{
			name:     "complex pattern",
			input:    `SELECT * FROM table WHERE name LIKE '%_\%'`,
			expected: `%SELECT * FROM table WHERE name LIKE '\%\_\\\%'%`,
		},
		// Real-world example: PostgreSQL pattern
		{
			name:     "postgresql pattern",
			input:    `test_%_pattern`,
			expected: `%test\_\%\_pattern%`,
		},
		// Real-world example: File path
		{
			name:     "windows file path",
			input:    `C:\Users\John\Documents\file_2024.txt`,
			expected: `%C:\\Users\\John\\Documents\\file\_2024.txt%`,
		},
		// Real-world example: Percentage in text
		{
			name:     "percentage in text",
			input:    `CPU usage: 75%`,
			expected: `%CPU usage: 75\%%`,
		},
		// Edge case: Only special characters
		{
			name:     "only special characters",
			input:    `\%_`,
			expected: `%\\\%\_%`,
		},
		// Edge case: Spaces (should not be escaped)
		{
			name:     "with spaces",
			input:    `Go Programming`,
			expected: `%Go Programming%`,
		},
		// Edge case: Numbers
		{
			name:     "numbers",
			input:    `12345`,
			expected: `%12345%`,
		},
		// Edge case: Special chars at start
		{
			name:     "special chars at start",
			input:    `%start`,
			expected: `%\%start%`,
		},
		// Edge case: Special chars at end
		{
			name:     "special chars at end",
			input:    `end_`,
			expected: `%end\_%`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeILIKE(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeILIKE(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeILIKE_BackslashEscapedFirst(t *testing.T) {
	// Verify that backslash is escaped before percent and underscore
	// This prevents incorrect double-escaping
	input := `\%`
	result := EscapeILIKE(input)
	expected := `%\\\%%`

	if result != expected {
		t.Errorf("EscapeILIKE(%q) = %q, want %q (backslash should be escaped first)", input, result, expected)
	}

	// Explanation:
	// Input: \%
	// Step 1: Escape \ to \\ → \\%
	// Step 2: Escape % to \% → \\\%
	// Step 3: Wrap with % → %\\\%%
}

func TestEscapeILIKE_PerformanceCharacteristics(t *testing.T) {
	// Test with long string to verify performance is acceptable
	// strings.NewReplacer should handle this efficiently
	longInput := strings.Repeat("test_pattern%", 1000)
	result := EscapeILIKE(longInput)

	// Verify that all special characters are escaped
	expectedCount := 1000 * 2 // 1000 underscores + 1000 percents
	escapedCount := countEscapes(result)

	if escapedCount < expectedCount {
		t.Errorf("Expected at least %d escaped characters, got %d", expectedCount, escapedCount)
	}

	// Verify result is wrapped
	if !strings.HasPrefix(result, "%") || !strings.HasSuffix(result, "%") {
		t.Errorf("Result should be wrapped with %%")
	}
}

// Helper function to count escape sequences
func countEscapes(s string) int {
	count := 0
	// Count \%, \_, and \\ sequences
	count += strings.Count(s, `\%`)
	count += strings.Count(s, `\_`)
	count += strings.Count(s, `\\`)
	return count
}

func BenchmarkEscapeILIKE(b *testing.B) {
	testCases := []string{
		"Go",
		"100%",
		"my_var",
		`path\file`,
		`%_\`,
		"日本語",
		"SELECT * FROM table WHERE name LIKE '%_\\%'",
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = EscapeILIKE(tc)
			}
		})
	}
}

func BenchmarkEscapeILIKE_LongString(b *testing.B) {
	longString := strings.Repeat("test_pattern%with\\special_chars", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EscapeILIKE(longString)
	}
}
