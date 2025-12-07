package search

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================
// Test Group 1: Valid Keyword Parsing
// ============================================================

func TestParseKeywords_SingleKeyword(t *testing.T) {
	keywords, err := ParseKeywords("Go", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go"}, keywords)
}

func TestParseKeywords_MultipleKeywords(t *testing.T) {
	keywords, err := ParseKeywords("Go React", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "React"}, keywords)
}

func TestParseKeywords_MultipleSpaces(t *testing.T) {
	keywords, err := ParseKeywords("Go  React", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "React"}, keywords)
}

func TestParseKeywords_LeadingTrailingSpaces(t *testing.T) {
	keywords, err := ParseKeywords("  Go React  ", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "React"}, keywords)
}

func TestParseKeywords_ThreeKeywords(t *testing.T) {
	keywords, err := ParseKeywords("Go React TypeScript", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "React", "TypeScript"}, keywords)
}

func TestParseKeywords_FiveKeywords(t *testing.T) {
	keywords, err := ParseKeywords("Go React TypeScript Python Rust", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "React", "TypeScript", "Python", "Rust"}, keywords)
}

func TestParseKeywords_MixedSpacing(t *testing.T) {
	keywords, err := ParseKeywords("  Go   React  TypeScript  ", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "React", "TypeScript"}, keywords)
}

func TestParseKeywords_TabsAndSpaces(t *testing.T) {
	keywords, err := ParseKeywords("Go\tReact\t\tTypeScript", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "React", "TypeScript"}, keywords)
}

func TestParseKeywords_NewlinesAndSpaces(t *testing.T) {
	keywords, err := ParseKeywords("Go\nReact\n\nTypeScript", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "React", "TypeScript"}, keywords)
}

// ============================================================
// Test Group 2: Empty Input Validation
// ============================================================

func TestParseKeywords_EmptyString(t *testing.T) {
	keywords, err := ParseKeywords("", 10, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
	assert.Contains(t, err.Error(), "keywords cannot be empty")
}

func TestParseKeywords_WhitespaceOnly(t *testing.T) {
	keywords, err := ParseKeywords("   ", 10, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
	assert.Contains(t, err.Error(), "keywords cannot be empty")
}

func TestParseKeywords_TabsOnly(t *testing.T) {
	keywords, err := ParseKeywords("\t\t\t", 10, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
	assert.Contains(t, err.Error(), "keywords cannot be empty")
}

func TestParseKeywords_NewlinesOnly(t *testing.T) {
	keywords, err := ParseKeywords("\n\n\n", 10, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
	assert.Contains(t, err.Error(), "keywords cannot be empty")
}

func TestParseKeywords_MixedWhitespaceOnly(t *testing.T) {
	keywords, err := ParseKeywords("  \t\n  \t\n  ", 10, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
	assert.Contains(t, err.Error(), "keywords cannot be empty")
}

// ============================================================
// Test Group 3: Keyword Count Validation
// ============================================================

func TestParseKeywords_TooManyKeywords(t *testing.T) {
	// 11 keywords with maxCount=10
	input := "k1 k2 k3 k4 k5 k6 k7 k8 k9 k10 k11"
	keywords, err := ParseKeywords(input, 10, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
	assert.Contains(t, err.Error(), "too many keywords")
	assert.Contains(t, err.Error(), "got 11")
	assert.Contains(t, err.Error(), "maximum 10")
}

func TestParseKeywords_ExactlyMaxCount(t *testing.T) {
	// Exactly 10 keywords with maxCount=10
	input := "k1 k2 k3 k4 k5 k6 k7 k8 k9 k10"
	keywords, err := ParseKeywords(input, 10, 100)
	assert.NoError(t, err)
	assert.Len(t, keywords, 10)
}

func TestParseKeywords_OneBelowMaxCount(t *testing.T) {
	// 9 keywords with maxCount=10
	input := "k1 k2 k3 k4 k5 k6 k7 k8 k9"
	keywords, err := ParseKeywords(input, 10, 100)
	assert.NoError(t, err)
	assert.Len(t, keywords, 9)
}

func TestParseKeywords_MaxCount1(t *testing.T) {
	// Edge case: maxCount=1
	keywords, err := ParseKeywords("Go", 1, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go"}, keywords)

	// Should error with 2 keywords
	keywords, err = ParseKeywords("Go React", 1, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
}

func TestParseKeywords_MaxCount2(t *testing.T) {
	// Edge case: maxCount=2
	keywords, err := ParseKeywords("Go React", 2, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "React"}, keywords)

	// Should error with 3 keywords
	keywords, err = ParseKeywords("Go React TypeScript", 2, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
}

// ============================================================
// Test Group 4: Keyword Length Validation
// ============================================================

func TestParseKeywords_KeywordTooLong(t *testing.T) {
	// 101 characters (exceeds maxLength=100)
	longKeyword := strings.Repeat("a", 101)
	keywords, err := ParseKeywords(longKeyword, 10, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
	assert.Contains(t, err.Error(), "exceeds maximum length")
	assert.Contains(t, err.Error(), "100 characters")
}

func TestParseKeywords_ExactlyMaxLength(t *testing.T) {
	// Exactly 100 characters
	keyword := strings.Repeat("a", 100)
	keywords, err := ParseKeywords(keyword, 10, 100)
	assert.NoError(t, err)
	assert.Len(t, keywords, 1)
	assert.Equal(t, keyword, keywords[0])
}

func TestParseKeywords_OneBelowMaxLength(t *testing.T) {
	// 99 characters
	keyword := strings.Repeat("a", 99)
	keywords, err := ParseKeywords(keyword, 10, 100)
	assert.NoError(t, err)
	assert.Len(t, keywords, 1)
	assert.Equal(t, keyword, keywords[0])
}

func TestParseKeywords_MultipleKeywords_OneTooLong(t *testing.T) {
	// First keyword is fine, second is too long
	longKeyword := strings.Repeat("b", 101)
	input := "Go " + longKeyword
	keywords, err := ParseKeywords(input, 10, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

func TestParseKeywords_MaxLength1(t *testing.T) {
	// Edge case: maxLength=1
	keywords, err := ParseKeywords("a", 10, 1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a"}, keywords)

	// Should error with 2 characters
	keywords, err = ParseKeywords("ab", 10, 1)
	assert.Error(t, err)
	assert.Nil(t, keywords)
}

func TestParseKeywords_MaxLength5(t *testing.T) {
	// maxLength=5
	keywords, err := ParseKeywords("short tiny", 10, 5)
	assert.NoError(t, err)
	assert.Equal(t, []string{"short", "tiny"}, keywords)

	// Should error with 6 characters
	keywords, err = ParseKeywords("toolong", 10, 5)
	assert.Error(t, err)
	assert.Nil(t, keywords)
}

// ============================================================
// Test Group 5: Unicode and Special Characters
// ============================================================

func TestParseKeywords_UnicodeKeywords(t *testing.T) {
	keywords, err := ParseKeywords("Go 日本語 中文", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "日本語", "中文"}, keywords)
}

func TestParseKeywords_EmojiKeywords(t *testing.T) {
	keywords, err := ParseKeywords("🚀 🎉 😊", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"🚀", "🎉", "😊"}, keywords)
}

func TestParseKeywords_MixedUnicodeASCII(t *testing.T) {
	keywords, err := ParseKeywords("Go React 日本語 TypeScript 中文", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "React", "日本語", "TypeScript", "中文"}, keywords)
}

func TestParseKeywords_UnicodeLength(t *testing.T) {
	// Japanese keyword with 3 characters (should count as 3, not byte length)
	keywords, err := ParseKeywords("日本語", 10, 3)
	assert.NoError(t, err)
	assert.Equal(t, []string{"日本語"}, keywords)

	// Should error with 4 Japanese characters when maxLength=3
	keywords, err = ParseKeywords("日本語語", 10, 3)
	assert.Error(t, err)
	assert.Nil(t, keywords)
}

func TestParseKeywords_SpecialCharacters(t *testing.T) {
	// Special characters should be allowed
	keywords, err := ParseKeywords("C++ Node.js @typescript #golang", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"C++", "Node.js", "@typescript", "#golang"}, keywords)
}

func TestParseKeywords_Hyphenated(t *testing.T) {
	keywords, err := ParseKeywords("test-driven full-stack micro-services", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"test-driven", "full-stack", "micro-services"}, keywords)
}

func TestParseKeywords_Underscores(t *testing.T) {
	keywords, err := ParseKeywords("snake_case camelCase PascalCase", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"snake_case", "camelCase", "PascalCase"}, keywords)
}

// ============================================================
// Test Group 6: Error Messages
// ============================================================

func TestParseKeywords_ErrorMessage_Empty(t *testing.T) {
	_, err := ParseKeywords("", 10, 100)
	assert.Error(t, err)
	assert.Equal(t, "keywords cannot be empty", err.Error())
}

func TestParseKeywords_ErrorMessage_TooManyKeywords(t *testing.T) {
	input := "k1 k2 k3 k4 k5 k6"
	_, err := ParseKeywords(input, 5, 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many keywords")
	assert.Contains(t, err.Error(), "got 6")
	assert.Contains(t, err.Error(), "maximum 5")
}

func TestParseKeywords_ErrorMessage_KeywordTooLong(t *testing.T) {
	keyword := strings.Repeat("x", 11)
	_, err := ParseKeywords(keyword, 10, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length")
	assert.Contains(t, err.Error(), "10 characters")
	assert.Contains(t, err.Error(), keyword)
}

// ============================================================
// Test Group 7: Edge Cases and Boundary Tests
// ============================================================

func TestParseKeywords_VeryLongInput(t *testing.T) {
	// 1000 short keywords (should error)
	parts := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		parts[i] = "k"
	}
	input := strings.Join(parts, " ")
	keywords, err := ParseKeywords(input, 10, 100)
	assert.Error(t, err)
	assert.Nil(t, keywords)
	assert.Contains(t, err.Error(), "too many keywords")
}

func TestParseKeywords_SingleCharacterKeywords(t *testing.T) {
	keywords, err := ParseKeywords("a b c d e", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, keywords)
}

func TestParseKeywords_NumericKeywords(t *testing.T) {
	keywords, err := ParseKeywords("123 456 789", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"123", "456", "789"}, keywords)
}

func TestParseKeywords_MixedAlphanumeric(t *testing.T) {
	keywords, err := ParseKeywords("v1.2.3 node16 es2022", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"v1.2.3", "node16", "es2022"}, keywords)
}

func TestParseKeywords_CasePreservation(t *testing.T) {
	// Should preserve original case
	keywords, err := ParseKeywords("Go REACT typescript", 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Go", "REACT", "typescript"}, keywords)
}

// ============================================================
// Test Group 8: Consistency and Nil Errors
// ============================================================

func TestParseKeywords_NilError(t *testing.T) {
	// Valid input should return nil error, not a zero-value error
	keywords, err := ParseKeywords("Go React", 10, 100)
	assert.Nil(t, err)
	assert.NotNil(t, keywords)
}

func TestParseKeywords_EmptySliceNeverReturned(t *testing.T) {
	// Function should never return empty slice with nil error
	// If input is effectively empty, should return error with nil slice
	_, err := ParseKeywords("   ", 10, 100)
	assert.Error(t, err)
}

// ============================================================
// Test Group 9: Realistic Use Cases
// ============================================================

func TestParseKeywords_RealisticSearchQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "tech stack search",
			input:    "Go PostgreSQL Docker",
			expected: []string{"Go", "PostgreSQL", "Docker"},
		},
		{
			name:     "framework search",
			input:    "React TypeScript Next.js",
			expected: []string{"React", "TypeScript", "Next.js"},
		},
		{
			name:     "cloud services",
			input:    "AWS Lambda DynamoDB",
			expected: []string{"AWS", "Lambda", "DynamoDB"},
		},
		{
			name:     "programming concepts",
			input:    "concurrency goroutines channels",
			expected: []string{"concurrency", "goroutines", "channels"},
		},
		{
			name:     "architecture terms",
			input:    "microservices event-driven CQRS",
			expected: []string{"microservices", "event-driven", "CQRS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords, err := ParseKeywords(tt.input, 10, 100)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, keywords)
		})
	}
}

func TestParseKeywords_TaskPlanLimits(t *testing.T) {
	// Test with actual limits from task plan: maxCount=10, maxLength=100
	t.Run("within limits", func(t *testing.T) {
		input := "k1 k2 k3 k4 k5 k6 k7 k8 k9 k10"
		keywords, err := ParseKeywords(input, 10, 100)
		assert.NoError(t, err)
		assert.Len(t, keywords, 10)
	})

	t.Run("exceed count limit", func(t *testing.T) {
		input := "k1 k2 k3 k4 k5 k6 k7 k8 k9 k10 k11"
		_, err := ParseKeywords(input, 10, 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many keywords")
	})

	t.Run("exceed length limit", func(t *testing.T) {
		longKeyword := strings.Repeat("a", 101)
		_, err := ParseKeywords(longKeyword, 10, 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})
}
