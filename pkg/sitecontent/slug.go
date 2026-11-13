// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

import (
	"regexp"
	"strings"
)

// slugRe restricts URL path segments to lowercase letters, digits, and dashes.
// A slug is derived from a display name via Slugify, never typed directly.
var slugRe = regexp.MustCompile(`^[a-z0-9-]+$`)

// whitespaceRun matches one or more whitespace characters, which become a
// single dash during slugification.
var whitespaceRun = regexp.MustCompile(`\s+`)

// nonSlug matches any character that is not a lowercase letter, digit, or
// dash; such characters are stripped silently during slugification.
var nonSlug = regexp.MustCompile(`[^a-z0-9-]+`)

// Slugify converts a display name into a URL-safe slug. It lowercases the
// input, replaces runs of whitespace with a single dash, strips all other
// non-alphanumeric characters, and trims any leading or trailing dashes.
func Slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = whitespaceRun.ReplaceAllString(slug, "-")
	slug = nonSlug.ReplaceAllString(slug, "")
	return strings.Trim(slug, "-")
}

// ValidateSlug reports whether s is a well-formed URL path segment.
func ValidateSlug(s string) bool {
	return slugRe.MatchString(s)
}

// ValidateName reports whether name may be used as a document display name.
// Document names may contain spaces and most printable characters, but must
// not be empty (after trimming), must stay under the database length cap, and
// must contain at least one alphanumeric character so the derived slug is
// never empty. Folder names are more restrictive; see folderLabelRe.
func ValidateName(name string) bool {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) == 0 || len(trimmed) >= 32 {
		return false
	}
	return Slugify(trimmed) != ""
}
