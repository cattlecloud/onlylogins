package database

import (
	"fmt"
	"testing"

	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/pkg/users/oauths"
	"github.com/shoenig/test/must"
)

const (
	// testUserCount is the number of pre-defined accounts
	testUsersCount = 3
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
		Username: "bob",
		Email:    "bob@example.org",
		Subject:  "e2bd401c",
		IDP:      oauths.Testing,
	})
	must.NoError(t, err)
	must.Positive(t, id)

	exp := testUsersCount + 1
	count, cerr := u.Count(ctx)
	must.NoError(t, cerr)
	must.Eq(t, exp, count)
}

func TestUsers_Exists(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()

	t.Run("does not exist", func(t *testing.T) {
		id, exists, err := u.Exists(ctx, oauths.Google, "doesnotexist")
		must.NoError(t, err)
		must.False(t, exists)
		must.Eq(t, users.Inactive, id)
	})

	t.Run("yes exists", func(t *testing.T) {
		_, cerr := u.Create(ctx, &users.CreateUserRequest{
			Username: "bob",
			Email:    "bob@example.org",
			Subject:  "ei39dnekeld",
			IDP:      oauths.Google,
		})
		must.NoError(t, cerr)

		id, exists, err := u.Exists(ctx, oauths.Google, "ei39dnekeld")
		must.NoError(t, err)
		must.True(t, exists)
		must.Positive(t, id)
	})
}

func TestUsers_Info(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()

	id, cerr := u.Create(ctx, &users.CreateUserRequest{
		Username: "bob",
		Email:    "bob@example.org",
		Subject:  "ei39dnekeld",
		IDP:      oauths.Google,
	})
	must.NoError(t, cerr)

	info, ierr := u.Info(ctx, id)
	must.NoError(t, ierr)

	must.Eq(t, "bob", info.Username)
	must.Eq(t, "bob@example.org", info.Email)
	must.Eq(t, users.Normal, info.Standing)
	must.Eq(t, oauths.Google, info.IDP)
}

func TestUsers_List(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()

	insertUsers(t, u)

	test, terr := u.Create(ctx, &users.CreateUserRequest{
		Username: "test",
		Email:    "test@example.org",
		Subject:  "aaaaaaaaaa",
		IDP:      oauths.Testing,
	})
	must.NoError(t, terr)
	must.Positive(t, test)

	infos, ierr := u.List(ctx)
	must.NoError(t, ierr)
	must.SliceLen(t, 5, infos)
	must.Eq(t, "alice", infos[0].Username)
	must.Eq(t, "bob", infos[1].Username)
	must.Eq(t, "carl", infos[2].Username)
	must.Eq(t, "dave", infos[3].Username)
	must.Eq(t, "eve", infos[4].Username)
}

func TestUsers_SetUsername(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()

	// create initial user with name "test"
	test, terr := u.Create(ctx, &users.CreateUserRequest{
		Username: "test",
		Email:    "test@example.org",
		Subject:  "aaaaaaaaaa",
		IDP:      oauths.Testing,
	})

	must.NoError(t, terr)
	must.Positive(t, test)

	// set the username to "TheBob"
	uerr := u.SetUsername(ctx, test, "TheBob")
	must.NoError(t, uerr)

	info, ierr := u.Info(ctx, test)
	must.NoError(t, ierr)
	must.Eq(t, "TheBob", info.Username)
}

func TestUsers_ListUsernameChanges(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()

	// create initial user with name "test"
	test, terr := u.Create(ctx, &users.CreateUserRequest{
		Username: "test",
		Email:    "test@example.org",
		Subject:  "aaaaaaaaaa",
		IDP:      oauths.Testing,
	})

	must.NoError(t, terr)
	must.Positive(t, test)

	// set the username to "TheBob"
	uerr := u.SetUsername(ctx, test, "TheBob")
	must.NoError(t, uerr)

	changes, lerr := u.ListUsernameChanges(ctx, test)
	must.NoError(t, lerr)
	must.Eq(t, "TheBob", changes[0].Chosen)
	must.Eq(t, "test", changes[0].Previous)
}

func TestUsers_Profile(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()

	insertUsers(t, u)

	profile, perr := u.Profile(ctx, 2)
	must.NoError(t, perr)
	must.False(t, profile.ShowEmail)
}

func TestUsers_ProfileSetShowEmail(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()
	const user = 2

	insertUsers(t, u)

	serr := u.ProfileSetShowEmail(ctx, user, true)
	must.NoError(t, serr)

	profile, perr := u.Profile(ctx, user)
	must.NoError(t, perr)
	must.True(t, profile.ShowEmail)
}

func TestUsers_ProfileSetLocation(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()
	const user = 2

	serr := u.ProfileSetLocation(ctx, user, "Austin, TX")
	must.NoError(t, serr)

	profile, perr := u.Profile(ctx, user)
	must.NoError(t, perr)
	must.Eq(t, "Austin, TX", profile.Location)
}

func TestUsers_ProfileSetLink(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()
	const user = 2

	serr := u.ProfileSetLink(ctx, user, "https://cattlecloud.net")
	must.NoError(t, serr)

	profile, perr := u.Profile(ctx, user)
	must.NoError(t, perr)
	must.Eq(t, "https://cattlecloud.net", profile.Link)
}

func TestUsers_ProfileSetAbout(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	u := db.Users()
	const user = 2

	list, lerr := u.List(ctx)
	must.NoError(t, lerr)
	fmt.Println("begin list")
	for _, u := range list {
		fmt.Println("u", u)
	}
	fmt.Println("end list")

	serr := u.ProfileSetAbout(ctx, user, "Hello, there!")
	must.NoError(t, serr)

	profile, perr := u.Profile(ctx, user)
	must.NoError(t, perr)
	must.Eq(t, "Hello, there!", profile.About)
}
