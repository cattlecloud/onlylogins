// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

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
