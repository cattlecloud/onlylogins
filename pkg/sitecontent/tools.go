// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

// Tool describes a one-off utility for working with the data of a user.
//
// Path is the location of the tool's own page, and Action is the label of
// the link that leads there.
type Tool struct {
	Name        string
	Description string
	Path        string
	Action      string
}

// Tools is the list of tools offered on the tools page.
//
// Each tool has a page of its own; adding one means adding an entry here
// along with the page and handler that implement it.
func Tools() []Tool {
	return []Tool{
		{
			Name:        "Export",
			Description: "Download a complete backup of your account, including every document's weblinks, usernames, passwords, key/value pairs, and notes, as one TOML file.",
			Path:        "/tools/export",
			Action:      "Export a backup",
		},
		{
			Name:        "Share",
			Description: "Create encrypted, self-destructing secret messages with a one-time link and an expiration timer. The recipient reads the message exactly once.",
			Path:        "/tools/share",
			Action:      "Share a secret",
		},
	}
}
