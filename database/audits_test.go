// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"testing"

	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func TestAudits_Record(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	insertUser(t, ctx, db, "alice")

	a := db.Audit()

	t.Run("success", func(t *testing.T) {
		request := &users.LoginRequest{
			Username:   "alice",
			Passphrase: conceal.New("hunter2"),
			UserAgent:  "TestUA/1.0",
			IP:         "127.0.0.1",
		}
		err := a.Record(ctx, request, true)
		must.NoError(t, err)

		logins, bogus, cerr := a.Count(ctx)
		must.NoError(t, cerr)
		must.Eq(t, 1, logins)
		must.Eq(t, 0, bogus)
	})

	t.Run("failure", func(t *testing.T) {
		request := &users.LoginRequest{
			Username:   "alice",
			Passphrase: conceal.New("hunter2"),
			UserAgent:  "TestUA/1.0",
			IP:         "127.0.0.1",
		}
		err := a.Record(ctx, request, false)
		must.NoError(t, err)

		logins, bogus, cerr := a.Count(ctx)
		must.NoError(t, cerr)
		must.Eq(t, 2, logins)
		must.Eq(t, 0, bogus)
	})
}

func TestAudits_RecordBogus(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	a := db.Audit()

	err := a.RecordBogus(ctx, &users.LoginRequest{
		UserAgent: "TestUA/1.0",
		IP:        "127.0.0.1",
	})
	must.NoError(t, err)

	logins, bogus, cerr := a.Count(ctx)
	must.NoError(t, cerr)
	must.Eq(t, 0, logins)
	must.Eq(t, 1, bogus)
}

func TestAudits_History(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	alice := insertUser(t, ctx, db, "alice")
	insertUser(t, ctx, db, "bob")

	a := db.Audit()

	record := func(username, ip string, success bool) {
		t.Helper()
		must.NoError(t, a.Record(ctx, &users.LoginRequest{
			Username:   username,
			Passphrase: conceal.New("hunter2"),
			UserAgent:  "TestUA/1.0",
			IP:         ip,
		}, success))
	}

	record("alice", "1.1.1.1", true)
	record("bob", "2.2.2.2", true)
	record("alice", "3.3.3.3", false)
	record("alice", "4.4.4.4", true)

	history, err := a.History(ctx, alice, 20)
	must.NoError(t, err)
	must.SliceLen(t, 3, history)

	// most recent first, and bob's login is excluded
	must.Eq(t, "4.4.4.4", history[0].IP)
	must.True(t, history[0].Success)
	must.Eq(t, "3.3.3.3", history[1].IP)
	must.False(t, history[1].Success)
	must.Eq(t, "1.1.1.1", history[2].IP)
	must.True(t, history[2].Success)

	// the limit is respected
	limited, err := a.History(ctx, alice, 2)
	must.NoError(t, err)
	must.SliceLen(t, 2, limited)
	must.Eq(t, "4.4.4.4", limited[0].IP)
	must.Eq(t, "3.3.3.3", limited[1].IP)
}
