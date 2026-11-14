// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"testing"

	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
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
	must.Eq(t, "miscellaneous", list[0].Label)
	must.True(t, list[0].Protected)
}

func TestFolders_Create(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	f := db.Folders()

	t.Run("create", func(t *testing.T) {
		request := &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "work",
			Slug:  "work",
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
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}
		err := f.Create(ctx, request)
		must.Error(t, err)
		must.Error(t, err)
	})

	t.Run("duplicate slug", func(t *testing.T) {
		// "Home" and "home" differ as labels but slugify to the same value;
		// the second must be rejected as a duplicate
		must.NoError(t, f.Create(ctx, &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "Home",
			Slug:  "home",
			Icon:  sitecontent.Icon(0),
		}))
		err := f.Create(ctx, &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "home",
			Slug:  "home",
			Icon:  sitecontent.Icon(0),
		})
		must.Error(t, err)
	})
}

func TestFolders_Delete(t *testing.T) {
	t.Parallel()

	t.Run("happy path cascades children", func(t *testing.T) {
		db, ctx := openTestDB(t)
		f := db.Folders()
		crypt := mocks.NewTestCrypt()

		// create the "work" folder
		must.NoError(t, f.Create(ctx, &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))

		// add a document inside it, with one of every kind of child entry;
		// the existing per-kind insert helpers hardcode the misc folder, so
		// duplicate the call sites inline here to target "work" instead
		_, err := db.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User:   nobody,
			Folder: "work",
			Label:  "github",
			Slug:   "github",
		})
		must.NoError(t, err)

		ct, cerr := crypt.Vault().Encrypt(conceal.New("https://example.com"))
		must.NoError(t, cerr)
		_, err = db.Documents().CreateWeblink(ctx, &sitecontent.CreateWeblinkRequest{
			User:      nobody,
			Folder:    "work",
			Document:  "github",
			URL:       "https://example.com",
			Encrypted: ct,
		})
		must.NoError(t, err)

		cu, cerr := crypt.Vault().Encrypt(conceal.New("alice"))
		must.NoError(t, cerr)
		_, err = db.Documents().CreateUsername(ctx, &sitecontent.CreateUsernameRequest{
			User:      nobody,
			Folder:    "work",
			Document:  "github",
			Username:  "alice",
			Encrypted: cu,
		})
		must.NoError(t, err)

		cp, cerr := crypt.Vault().Encrypt(conceal.New("s3cret!"))
		must.NoError(t, cerr)
		_, err = db.Documents().CreatePassword(ctx, &sitecontent.CreatePasswordRequest{
			User:      nobody,
			Folder:    "work",
			Document:  "github",
			Encrypted: cp,
		})
		must.NoError(t, err)

		cn, cerr := crypt.Vault().Encrypt(conceal.New("hi"))
		must.NoError(t, cerr)
		_, err = db.Documents().CreateNote(ctx, &sitecontent.CreateNoteRequest{
			User:      nobody,
			Folder:    "work",
			Document:  "github",
			Encrypted: cn,
		})
		must.NoError(t, err)

		// delete the folder; cascade should also wipe the document and all
		// of its children
		err = f.Delete(ctx, &sitecontent.DeleteFolderRequest{
			User: nobody,
			Slug: "work",
		})
		must.NoError(t, err)

		// folder listing is now back to just the default "miscellaneous"
		count, cerr := f.Count(ctx)
		must.NoError(t, cerr)
		must.Eq(t, testUsersCount, count)

		// and the work/github document is gone
		doc, gerr := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: nobody, Folder: "work", Document: "github",
		})
		must.Error(t, gerr)
		must.Nil(t, doc)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		f := db.Folders()

		must.NoError(t, f.Create(ctx, &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))
		// inserting bob creates his own system "miscellaneous" folder via the
		// default-folder trigger, so the global folder count goes up by 2
		bob := insertUser(t, ctx, db, "bob")
		_ = bob

		err := f.Delete(ctx, &sitecontent.DeleteFolderRequest{
			User: bob,
			Slug: "work",
		})
		must.Error(t, err)

		// "work" must still exist under nobody; read back via List and find it
		list, lerr := f.List(ctx, nobody)
		must.NoError(t, lerr)
		var found bool
		for _, folder := range list {
			if folder.Label == "work" {
				found = true
				break
			}
		}
		must.True(t, found)
	})

	t.Run("unknown folder", func(t *testing.T) {
		db, ctx := openTestDB(t)
		f := db.Folders()

		err := f.Delete(ctx, &sitecontent.DeleteFolderRequest{
			User: nobody,
			Slug: "no-such-folder",
		})
		must.Error(t, err)
	})

	t.Run("system folder", func(t *testing.T) {
		db, ctx := openTestDB(t)
		f := db.Folders()

		// attempting to delete the default "miscellaneous" system folder must
		// fail; the SQL `protected = 0` filter refuses to delete protected rows
		err := f.Delete(ctx, &sitecontent.DeleteFolderRequest{
			User: nobody,
			Slug: "miscellaneous",
		})
		must.Error(t, err)

		count, cerr := f.Count(ctx)
		must.NoError(t, cerr)
		must.Eq(t, testUsersCount, count)
	})
}

func TestFolders_Rename(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		f := db.Folders()

		must.NoError(t, f.Create(ctx, &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))

		_, err := db.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User:   nobody,
			Folder: "work",
			Label:  "github",
			Slug:   "github",
		})
		must.NoError(t, err)

		must.NoError(t, f.Rename(ctx, &sitecontent.RenameFolderRequest{
			User:    nobody,
			Slug:    "work",
			Label:   "home",
			NewSlug: "home",
		}))

		// the folder now exists under its new label and slug
		list, lerr := f.List(ctx, nobody)
		must.NoError(t, lerr)
		var found bool
		for _, folder := range list {
			if folder.Label == "home" && folder.Slug == "home" {
				found = true
				break
			}
		}
		must.True(t, found)

		// the old slug no longer resolves
		_, gerr := f.Get(ctx, nobody, "work")
		must.Error(t, gerr)

		// documents follow the folder id, so they resolve via the new slug
		doc, derr := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: nobody, Folder: "home", Document: "github",
		})
		must.NoError(t, derr)
		must.Eq(t, "github", doc.Slug)
	})

	t.Run("case-only rename", func(t *testing.T) {
		db, ctx := openTestDB(t)
		f := db.Folders()

		must.NoError(t, f.Create(ctx, &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "Work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))

		// renaming to the same slug (case change of the label) succeeds
		must.NoError(t, f.Rename(ctx, &sitecontent.RenameFolderRequest{
			User:    nobody,
			Slug:    "work",
			Label:   "work",
			NewSlug: "work",
		}))
	})

	t.Run("duplicate slug", func(t *testing.T) {
		db, ctx := openTestDB(t)
		f := db.Folders()

		must.NoError(t, f.Create(ctx, &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "home",
			Slug:  "home",
			Icon:  sitecontent.Icon(0),
		}))
		must.NoError(t, f.Create(ctx, &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))

		err := f.Rename(ctx, &sitecontent.RenameFolderRequest{
			User:    nobody,
			Slug:    "work",
			Label:   "home",
			NewSlug: "home",
		})
		must.Error(t, err)
	})

	t.Run("protected folder", func(t *testing.T) {
		db, ctx := openTestDB(t)
		f := db.Folders()

		err := f.Rename(ctx, &sitecontent.RenameFolderRequest{
			User:    nobody,
			Slug:    "miscellaneous",
			Label:   "other",
			NewSlug: "other",
		})
		must.Error(t, err)

		count, cerr := f.Count(ctx)
		must.NoError(t, cerr)
		must.Eq(t, testUsersCount, count)
	})

	t.Run("unknown folder", func(t *testing.T) {
		db, ctx := openTestDB(t)
		f := db.Folders()

		err := f.Rename(ctx, &sitecontent.RenameFolderRequest{
			User:    nobody,
			Slug:    "no-such-folder",
			Label:   "other",
			NewSlug: "other",
		})
		must.Error(t, err)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		f := db.Folders()

		must.NoError(t, f.Create(ctx, &sitecontent.CreateFolderRequest{
			User:  nobody,
			Label: "work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))
		bob := insertUser(t, ctx, db, "bob")
		_ = bob

		err := f.Rename(ctx, &sitecontent.RenameFolderRequest{
			User:    bob,
			Slug:    "work",
			Label:   "home",
			NewSlug: "home",
		})
		must.Error(t, err)

		// "work" must still exist under nobody
		list, lerr := f.List(ctx, nobody)
		must.NoError(t, lerr)
		var found bool
		for _, folder := range list {
			if folder.Label == "work" {
				found = true
				break
			}
		}
		must.True(t, found)
	})
}

func TestFolders_List_Slug(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	f := db.Folders()

	must.NoError(t, f.Create(ctx, &sitecontent.CreateFolderRequest{
		User:  nobody,
		Label: "Work",
		Slug:  "work",
		Icon:  sitecontent.Icon(0),
	}))

	list, err := f.List(ctx, nobody)
	must.NoError(t, err)

	var found *sitecontent.Folder
	for _, folder := range list {
		if folder.Label == "Work" {
			found = folder
			break
		}
	}
	must.NotNil(t, found)
	must.Eq(t, "work", found.Slug)
}
