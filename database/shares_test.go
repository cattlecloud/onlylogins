// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func insertShare(
	t *testing.T, ctx scope.C, store Store,
	user users.Identity, secret string, when time.Time, cypher mocks.TestCypher,
) records.SharesUnique {
	t.Helper()

	uid, err := store.Shares(cypher.Cypher()).Create(ctx, &sitecontent.CreateShareRequest{
		User:   user,
		Secret: conceal.New(secret),
		TTL:    5 * time.Minute,
		Now:    when,
	})
	must.NoError(t, err)
	must.NotEq(t, records.SharesUnique(records.None), uid)
	must.Eq(t, 36, len(uid))

	return uid
}

func TestShares_Count(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	cypher := mocks.NewTestCypher()
	s := db.Shares(cypher.Cypher())

	count, err := s.Count(ctx)
	must.NoError(t, err)
	must.Zero(t, count)

	insertShare(t, ctx, db, mocks.Nobody, "alpha", time.Now(), cypher)
	insertShare(t, ctx, db, mocks.Nobody, "beta", time.Now(), cypher)
	insertShare(t, ctx, db, mocks.Nobody, "gamma", time.Now(), cypher)

	count, err = s.Count(ctx)
	must.NoError(t, err)
	must.Eq(t, 3, count)
}

func TestShares_Create(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	cypher := mocks.NewTestCypher()
	s := db.Shares(cypher.Cypher())

	uid, err := s.Create(ctx, &sitecontent.CreateShareRequest{
		User:   mocks.Nobody,
		Secret: conceal.New("hello world"),
		TTL:    5 * time.Minute,
		Now:    time.Now(),
	})
	must.NoError(t, err)
	must.NotEq(t, records.SharesUnique(records.None), uid)
	must.Eq(t, 36, len(uid))

	count, cerr := s.Count(ctx)
	must.NoError(t, cerr)
	must.One(t, count)
}

func TestShares_Consume(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	cypher := mocks.NewTestCypher()
	s := db.Shares(cypher.Cypher())

	uid := insertShare(t, ctx, db, mocks.Nobody, "the-secret", time.Now(), cypher)

	share, err := s.Consume(ctx, &sitecontent.ConsumeShareRequest{
		UUID: uid,
	})
	must.NoError(t, err)
	must.NotNil(t, share)
	must.Eq(t, uid, share.UUID)
	must.Eq(t, mocks.Nobody, share.User)
	must.Eq(t, "the-secret", share.Secret.Unveil())
	must.Eq(t, 5*time.Minute, share.TTL)

	// consume is destructive; the share is gone from the table afterwards
	count, cerr := s.Count(ctx)
	must.NoError(t, cerr)
	must.Zero(t, count)
}

func TestShares_Prune(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	cypher := mocks.NewTestCypher()
	s := db.Shares(cypher.Cypher())

	now := time.Now()
	_ = insertShare(t, ctx, db, mocks.Nobody, "old-secret", now.Add(-5*time.Hour), cypher)
	newUID := insertShare(t, ctx, db, mocks.Nobody, "new-secret", now, cypher)

	count, err := s.Count(ctx)
	must.NoError(t, err)
	must.Eq(t, 2, count)

	pruned, err := s.Prune(ctx, 1*time.Hour)
	must.NoError(t, err)
	must.One(t, pruned)

	count, err = s.Count(ctx)
	must.NoError(t, err)
	must.One(t, count)

	// only the recent share survives; consume it to confirm the right one
	share, err := s.Consume(ctx, &sitecontent.ConsumeShareRequest{
		UUID: newUID,
	})
	must.NoError(t, err)
	must.Eq(t, "new-secret", share.Secret.Unveil())
}
