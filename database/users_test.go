package database

import (
	"testing"

	"cattlecloud.net/onlylogins/hack/mocks"
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

func TestUsers_Create(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()

	id, err := u.Create(ctx, &users.CreateUserRequest{
		Username:   "bob",
		Shibboleth: conceal.UUIDv4(),
		Passkey:    mocks.Pass,
	})
	must.NoError(t, err)
	must.Positive(t, id)

	exp := testUsersCount + 1
	count, cerr := u.Count(ctx)
	must.NoError(t, cerr)
	must.Eq(t, exp, count)
}
