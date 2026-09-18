package format

import (
	"fmt"
	"strings"
)

// PageDescription takes a chunk of format text and fixes spacing issues,
// then uses the given args to format the string.
//
// No truncation; must be used with known inputs (below 150 characters).
func PageDescription(text string, args ...any) string {
	fields := strings.Fields(text)
	spacing := strings.Join(fields, " ")
	return fmt.Sprintf(spacing, args...)
}

// PageDescriptionTruncate truncates the given text, which may be from an external
// source such as user input, down to fewer than 150 characters.
func PageDescriptionTruncate(text string) string {
	fields := strings.Fields(text)
	spacing := strings.Join(fields, " ")

	if len(spacing) < 150 {
		return spacing
	}

	// shorten to the first 140 characters
	return spacing[0:140] + "..."
}
