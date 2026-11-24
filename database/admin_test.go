// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"testing"

	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/hashicorp/go-set/v3"
	"github.com/shoenig/test/must"
)

func TestAdmin_ListStanding(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	a := db.Admin()

	t.Run("initially empty", func(t *testing.T) {
		admins, err := a.ListStanding(ctx, users.Admin)
		must.NoError(t, err)
		must.True(t, admins.Empty())
	})
}

func TestAdmin_Sync(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	a := db.Admin()

	alice := insertUser(t, ctx, db, "alice")
	bob := insertUser(t, ctx, db, "bob")

	t.Run("promote alice", func(t *testing.T) {
		err := a.Sync(ctx, set.From([]users.Identity{alice}), set.New[users.Identity](0))
		must.NoError(t, err)

		admins, lerr := a.ListStanding(ctx, users.Admin)
		must.NoError(t, lerr)
		must.True(t, admins.Contains(alice))
		must.False(t, admins.Contains(bob))
	})

	t.Run("demote alice", func(t *testing.T) {
		err := a.Sync(ctx, set.New[users.Identity](0), set.From([]users.Identity{alice}))
		must.NoError(t, err)

		admins, lerr := a.ListStanding(ctx, users.Admin)
		must.NoError(t, lerr)
		must.True(t, admins.Empty())

		profile, perr := db.Users().Lookup(ctx, alice)
		must.NoError(t, perr)
		must.Eq(t, users.Normal, profile.Standing)
	})
}
