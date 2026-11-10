// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

type ExportEverything struct {
	Username string          `json:"username"`
	Folders  []*ExportFolder `json:"folders"`
}

type ExportFolder struct {
	Name      string            `json:"name"`
	Documents []*ExportDocument `json:"documents"`
}

type ExportDocument struct {
	Name      string      `json:"name"`
	Weblinks  []string    `json:"weblinks,omitempty"`
	Usernames []string    `json:"usernames,omitempty"`
	Passwords []string    `json:"passwords,omitempty"`
	Keyvals   [][2]string `json:"keyvals,omitempty"`
	Note      string      `json:"note,omitempty"`
}
