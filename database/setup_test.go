package database

import (
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/pkg/users/oauths"
	"github.com/shoenig/test/must"
)

// insertUsers inserts five default users into the database:
//   - alice: apple
//   - bob: google
//   - carl: microsoft
//   - dave: microsoft
//   - eve: google
func insertUsers(t *testing.T, u Users) map[string]users.Identity {
	ctx, cancel := scope.WithTTL(t.Context(), 10*time.Second)
	defer cancel()

	alice, aerr := u.Create(ctx, &users.CreateUserRequest{
		Username: "alice",
		Email:    "alice@example.org",
		Subject:  "neixkeim39j",
		IDP:      oauths.Apple,
	})
	must.NoError(t, aerr)
	must.Positive(t, alice)

	bob, berr := u.Create(ctx, &users.CreateUserRequest{
		Username: "bob",
		Email:    "bob@example.org",
		Subject:  "ei39dnekeld",
		IDP:      oauths.Google,
	})
	must.NoError(t, berr)
	must.Positive(t, bob)

	carl, cerr := u.Create(ctx, &users.CreateUserRequest{
		Username: "carl",
		Email:    "carl@example.org",
		Subject:  "nnuj4h9d3kni",
		IDP:      oauths.Microsoft,
	})
	must.NoError(t, cerr)
	must.Positive(t, carl)

	dave, derr := u.Create(ctx, &users.CreateUserRequest{
		Username: "dave",
		Email:    "dave@example.org",
		Subject:  "jkeidnmeidjne",
		IDP:      oauths.Microsoft,
	})
	must.NoError(t, derr)
	must.Positive(t, dave)

	eve, eerr := u.Create(ctx, &users.CreateUserRequest{
		Username: "eve",
		Email:    "eve@example.org",
		Subject:  "qkeixkencie9jns",
		IDP:      oauths.Google,
	})
	must.NoError(t, eerr)
	must.Positive(t, eve)

	return map[string]users.Identity{
		"alice": alice,
		"bob":   bob,
		"carl":  carl,
		"dave":  dave,
		"eve":   eve,
	}
}
