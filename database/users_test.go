// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import (
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

const (
	// testUserCount is the number of pre-defined accounts
	// we only pre-define _nobody in onlylogins
	testUsersCount = 1
)

func TestUsers_Count(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()

	count, err := u.Count(ctx)
	must.NoError(t, err)
	must.Eq(t, testUsersCount, count)
}

func insertUser(t *testing.T, ctx scope.C, store Store, username string) users.Identity {
	t.Helper()

	salt := encrypt.Salt()

	request := &users.CreateUserRequest{
		Email:         username + "@example.com",
		Username:      username,
		Salt:          salt,
		Passphrase:    conceal.New("pass+" + username),
		Notifications: true,
		Spam:          true,
	}

	key := encrypt.DeriveKey(request.Passphrase, request.Salt)
	verifier := encrypt.MakeVerifier(key)

	id, err := store.Users().Create(ctx, request, verifier)
	must.NoError(t, err)
	return id
}

func TestUsers_Create(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()

	id := insertUser(t, ctx, db, "alice")
	must.Positive(t, id)

	exp := testUsersCount + 1
	count, cerr := u.Count(ctx)
	must.NoError(t, cerr)
	must.Eq(t, exp, count)
}

func TestUsers_Lookup(t *testing.T) {
	t.Parallel()

	db, _ := openTestDB(t)

	// argon will burn a lot of time on these tests because we are minting
	// and verifying several users; give the test a long time to accommodate
	ctx, cancel := scope.WithTTL(t.Context(), 1*time.Minute)
	t.Cleanup(cancel)

	alice := insertUser(t, ctx, db, "alice")
	bob := insertUser(t, ctx, db, "bob")
	carol := insertUser(t, ctx, db, "carol")

	t.Run("lookup alice", func(t *testing.T) {
		u, err := db.Users().Lookup(ctx, alice)
		must.NoError(t, err)
		must.Eq(t, "alice", u.Username)
		must.Eq(t, "alice@example.com", u.Email)
	})

	t.Run("lookup bob", func(t *testing.T) {
		u, err := db.Users().Lookup(ctx, bob)
		must.NoError(t, err)
		must.Eq(t, "bob", u.Username)
		must.Eq(t, "bob@example.com", u.Email)
	})

	t.Run("lookup carol", func(t *testing.T) {
		u, err := db.Users().Lookup(ctx, carol)
		must.NoError(t, err)
		must.Eq(t, "carol", u.Username)
		must.Eq(t, "carol@example.com", u.Email)
	})
}

func TestUsers_List(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()

	t.Run("initial", func(t *testing.T) {
		list, err := u.List(ctx)
		must.NoError(t, err)
		must.Len(t, testUsersCount, list)

		// the nobody user comes with the schema
		first := list[0]
		must.Eq(t, users.Identity(testUsersCount), first.ID)
		must.Eq(t, "_nobody", first.Username)
		must.Eq(t, "nobody@onlylogins.invalid", first.Email)
		must.Eq(t, users.Sentinel, first.Standing)
	})

	t.Run("after inserts", func(t *testing.T) {
		alice := insertUser(t, ctx, db, "alice")
		bob := insertUser(t, ctx, db, "bob")
		carol := insertUser(t, ctx, db, "carol")

		list, err := u.List(ctx)
		must.NoError(t, err)
		must.Len(t, testUsersCount+3, list)

		// sentinel user remains first by id
		must.Eq(t, "_nobody", list[0].Username)
		must.Eq(t, users.Sentinel, list[0].Standing)

		// inserted users follow in insertion order (auto-increment ids)
		must.Eq(t, alice, list[1].ID)
		must.Eq(t, "alice", list[1].Username)
		must.Eq(t, "alice@example.com", list[1].Email)
		must.Eq(t, users.Normal, list[1].Standing)

		must.Eq(t, bob, list[2].ID)
		must.Eq(t, "bob", list[2].Username)
		must.Eq(t, users.Normal, list[2].Standing)

		must.Eq(t, carol, list[3].ID)
		must.Eq(t, "carol", list[3].Username)
		must.Eq(t, "carol@example.com", list[3].Email)
		must.Eq(t, users.Normal, list[3].Standing)
	})
}
