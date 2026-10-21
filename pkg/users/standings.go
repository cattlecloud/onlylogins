// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package users

type Standing int

const (
	Normal    Standing = 1
	Admin     Standing = 2
	Moderator Standing = 3
	Banned    Standing = 4
	Sentinel  Standing = 5
)

func (s Standing) String() string {
	switch s {
	case Normal:
		return "Normal"
	case Admin:
		return "Admin"
	case Moderator:
		return "Moderator"
	case Banned:
		return "Banned"
	case Sentinel:
		return "Sentinel"
	default:
		panic("users: not a standing")
	}
}
