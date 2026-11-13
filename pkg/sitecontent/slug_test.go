// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"My Folder":         "my-folder",
		"my folder":         "my-folder",
		"  spaced  ":        "spaced",
		"UPPER CASE":        "upper-case",
		"a-b--c":            "a-b--c",
		"Amazon.com":        "amazoncom",
		"café":              "caf",
		"100% Done":         "100-done",
		"---":               "",
		"日本語":               "",
		"":                  "",
		"multiple   spaces": "multiple-spaces",
	}

	for name, want := range cases {
		must.Eq(t, want, Slugify(name), must.Sprintf("name=%q", name))
	}
}

func TestValidateSlug(t *testing.T) {
	t.Parallel()

	must.True(t, ValidateSlug("miscellaneous"))
	must.True(t, ValidateSlug("my-folder-2"))
	must.True(t, ValidateSlug("abc123"))

	must.False(t, ValidateSlug(""))
	must.False(t, ValidateSlug("My Folder"))
	must.False(t, ValidateSlug("UPPER"))
	must.False(t, ValidateSlug("under_score"))
	must.False(t, ValidateSlug("dot.dot"))
}

func TestValidateName(t *testing.T) {
	t.Parallel()

	// valid display names
	must.True(t, ValidateName("work"))
	must.True(t, ValidateName("My Folder"))
	must.True(t, ValidateName("my-folder"))
	must.True(t, ValidateName("café"))
	must.True(t, ValidateName("Amazon.com"))
	must.True(t, ValidateName("   padded   "))

	// empty / no alphanumeric / too long
	must.False(t, ValidateName(""))
	must.False(t, ValidateName("   "))
	must.False(t, ValidateName("***"))
	must.False(t, ValidateName("日本語"))
	must.False(t, ValidateName("abcdefghijklmnopqrstuvwxyz123456"))
}
