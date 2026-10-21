package database

import (
	"testing"

	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/test/must"
)

const (
	nobody users.Identity = 1
)

func TestFolders_Count(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	f := db.Folders()

	count, err := f.Count(ctx)
	must.NoError(t, err)
	must.Eq(t, testUsersCount, count)
}

func TestFolders_List(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	f := db.Folders()

	list, err := f.List(ctx, nobody)
	must.NoError(t, err)
	must.SliceLen(t, 1, list)
}
