// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import (
	"testing"

	"cattlecloud.net/onlylogins/pkg/sitecontent"
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

func TestFolders_Create(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	f := db.Folders()

	t.Run("create", func(t *testing.T) {
		request := &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "work",
			Icon:  sitecontent.Icon(0),
		}
		err := f.Create(ctx, request)
		must.NoError(t, err)

		count, cerr := f.Count(ctx)
		must.NoError(t, cerr)
		must.Eq(t, testUsersCount+1, count)

		list, lerr := f.List(ctx, nobody)
		must.NoError(t, lerr)
		must.SliceLen(t, 2, list)
		must.Eq(t, "miscellaneous", list[0].Label) // alphabetically first
		must.Eq(t, "work", list[1].Label)
	})

	t.Run("duplicate label", func(t *testing.T) {
		request := &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "work",
			Icon:  sitecontent.Icon(0),
		}
		err := f.Create(ctx, request)
		must.Error(t, err)
		must.Error(t, err)
	})
}
