// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package records

import "cattlecloud.net/go/litesql"

type (
	// table plans
	PlansEnum litesql.ID
)

type (
	// table folders
	FoldersAuto litesql.ID

	// table documents
	DocumentsAuto litesql.ID

	// table weblinks
	WeblinksAuto litesql.ID

	// table usernames
	UsernamesAuto litesql.ID

	// table passwords
	PasswordsAuto litesql.ID

	// table keyvals
	KeyvalsAuto litesql.ID

	// table notes
	NotesAuto litesql.ID
)
