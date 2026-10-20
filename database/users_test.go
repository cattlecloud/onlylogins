package database

import (
	"testing"

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

	db, ctx := openTestDB(t)

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
