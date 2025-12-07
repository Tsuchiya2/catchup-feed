package search

import "strings"

// EscapeILIKE escapes special characters for PostgreSQL ILIKE patterns.
// Escapes: % (wildcard), _ (single char), \ (escape char)
// Returns the escaped string wrapped with % for partial matching.
//
// Special characters are escaped in the following order:
//  1. Backslash (\) → \\
//  2. Percent (%) → \%
//  3. Underscore (_) → \_
//
// The result is wrapped with % for ILIKE pattern matching.
//
// Examples:
//
//	EscapeILIKE("Go")           // "%Go%"
//	EscapeILIKE("100%")         // "%100\\%%"
//	EscapeILIKE("my_var")       // "%my\\_var%"
//	EscapeILIKE("path\\file")   // "%path\\\\file%"
//	EscapeILIKE("%_\\")         // "%\\%\\_\\\\%"
//	EscapeILIKE("")             // "%%"
//	EscapeILIKE("日本語")        // "%日本語%"
func EscapeILIKE(input string) string {
	// Use strings.NewReplacer for efficient replacement
	// Order matters: escape backslash first to avoid double-escaping
	replacer := strings.NewReplacer(
		`\`, `\\`, // Escape backslash first
		`%`, `\%`, // Escape percent
		`_`, `\_`, // Escape underscore
	)

	escaped := replacer.Replace(input)

	// Wrap with % for partial matching
	return "%" + escaped + "%"
}
