// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package users

import (
	"regexp"
	"strings"

	"github.com/hashicorp/go-set/v3"
)

// AllowUsername returns whether s is an allowable username. A good username
// must be in the form <letter>-<letters/numbers>.
//
// Certain admin-related words are specifically forbidden.
func AllowUsername(s string) bool {
	lower := strings.ToLower(s)
	if forbiddenUsernames.Contains(lower) {
		return false
	}

	return usernameRe.MatchString(s)
}

var forbiddenUsernames = set.From([]string{
	"admin", "admins", "administrator", "administrators",
	"administration", "administrations",
	"mod", "mods", "moderator", "moderators", "moderation", "moderations",
	"webmaster", "webmasters",
	"owner", "owners",
})

var usernameRe = regexp.MustCompile(`^[a-zA-Z]+[a-zA-Z0-9-_]+$`)
