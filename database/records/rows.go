// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package records

import (
	"cattlecloud.net/go/litesql"
	"github.com/shoenig/go-conceal"
)

type (
	// table plans
	PlansEnum litesql.ID

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

	// table shares
	SharesUnique UUID
)

// UUID represents a v4 unique identifier.
type UUID string

// Unique generates a new randomized UUID.
func Unique[T ~string]() T {
	return T(conceal.UUIDv4().Unveil())
}

// String returns UUID as a string.
func (u UUID) String() string {
	return string(u)
}

const (
	// None represents a zero value of UUID.
	None UUID = "00000000-0000-0000-0000-000000000000"
)

const (
	PlansFree    PlansEnum = 1
	PlansPatreon PlansEnum = 2
)
