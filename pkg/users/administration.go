// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package users

import (
	"cattlecloud.net/go/ulog"
	"github.com/hashicorp/go-set/v3"
)

type Council interface {
	IsAdmin(Identity) bool
}

func NewCouncil(admins []Identity) Council {
	return &council{
		log:    ulog.New("users/council"),
		admins: set.From(admins),
	}
}

type council struct {
	log    *ulog.Log
	admins *set.Set[Identity]
}

func (c *council) IsAdmin(id Identity) bool {
	return c.admins.Contains(id)
}
