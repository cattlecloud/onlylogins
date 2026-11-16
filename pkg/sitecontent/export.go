// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

type ExportEverything struct {
	Username string          `toml:"username"`
	Folders  []*ExportFolder `toml:"folders"`
}

type ExportFolder struct {
	Name      string            `toml:"name"`
	Documents []*ExportDocument `toml:"documents"`
}

type ExportDocument struct {
	Name      string      `toml:"name"`
	Weblinks  []string    `toml:"weblinks,omitempty"`
	Usernames []string    `toml:"usernames,omitempty"`
	Passwords []string    `toml:"passwords,omitempty"`
	Keyvals   [][2]string `toml:"keyvals,omitempty"`
	Note      string      `toml:"note,omitempty"`
}
