// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package sitecontent

// Profile contains the information displayed on a user's profile page.
//
// Usage carries the number of documents the user has created along with the
// limit imposed by their plan.
type Profile struct {
	Username string
	Usage    Usage
}
