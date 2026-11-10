// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package format

import (
	"fmt"
	"path/filepath"
	"strings"

	"cattlecloud.net/onlylogins/pkg/sitecontent"
)

// Icon gets the SVG associated with an icon enum.
func Icon(number sitecontent.Icon) string {
	size := sitecontent.Icon(len(sitecontent.FolderIcons))
	if number < 0 || number >= size {
		number = 0
	}
	return filepath.Join("ui/images", sitecontent.FolderIcons[number])
}

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

// PageTitle appends the site suffix to the given page title.
func PageTitle(s string) string {
	return s + " | OnlyLogins"
}
