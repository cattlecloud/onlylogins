// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import (
	"testing"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/test/must"
)

func TestDocuments_Count(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	d := db.Documents()

	count, err := d.Count(ctx)
	must.NoError(t, err)
	must.Zero(t, count)
}

// miscFolderOf returns the label of the user's default "miscellaneous" folder,
func miscFolderOf(t *testing.T, ctx scope.C, store Store, user users.Identity) string {
	t.Helper()
	list, err := store.Folders().List(ctx, user)
	must.NoError(t, err)
	must.SliceLen(t, 1, list)
	must.Eq(t, "miscellaneous", list[0].Label)
	return list[0].Label
}

func insertDocument(t *testing.T, ctx scope.C, store Store, user users.Identity, label string) records.DocumentsAuto {
	t.Helper()

	id, err := store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User:   user,
		Folder: miscFolderOf(t, ctx, store, user),
		Label:  label,
	})
	must.NoError(t, err)

	return id
}

func TestDocuments_Create(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	d := db.Documents()

	t.Run("create", func(t *testing.T) {
		id := insertDocument(t, ctx, db, mocks.Nobody, "github")
		must.Positive(t, id)

		count, cerr := d.Count(ctx)
		must.NoError(t, cerr)
		must.One(t, count)
	})

	t.Run("duplicate label", func(t *testing.T) {
		request := &sitecontent.CreateDocumentRequest{
			User:   nobody,
			Folder: miscFolderOf(t, ctx, db, mocks.Nobody),
			Label:  "github",
		}
		_, err := d.Create(ctx, request)
		must.Error(t, err)
		must.ErrorIs(t, err, ErrDocumentLabelTaken)
	})
}
