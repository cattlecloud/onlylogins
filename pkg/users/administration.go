// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package users

import (
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"github.com/hashicorp/go-set/v3"
)

type Store interface {
	Sync(scope.C, set.Collection[Identity], set.Collection[Identity]) error
	ListStanding(scope.C, Standing) (set.Collection[Identity], error)
}

type Council interface {
	Sync(scope.C)
	IsAdmin(Identity) bool
}

func NewCouncil(admins []Identity, store Store) Council {
	return &council{
		log:    ulog.New("users/council"),
		admins: set.From(admins),
		store:  store,
	}
}

type council struct {
	log    *ulog.Log
	admins *set.Set[Identity]
	store  Store
}

func (c *council) Sync(ctx scope.C) {
	existing, lerr := c.store.ListStanding(ctx, Admin)
	if lerr != nil {
		c.log.E.Fmt("unable to list admin users: %v", lerr)
		return
	}

	add := c.admins.Difference(existing)
	remove := existing.Difference(c.admins)

	if err := c.store.Sync(ctx, add, remove); err != nil {
		c.log.E.Fmt("unable to sync admin users: %v", err)
		return
	}

	c.log.I.Fmt("admins sync complete")
}

func (c *council) IsAdmin(id Identity) bool {
	return c.admins.Contains(id)
}
