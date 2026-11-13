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

func insertWeblink(t *testing.T, ctx scope.C, store Store, user users.Identity, doc, url string, crypt mocks.TestCrypt) records.WeblinksAuto {
	t.Helper()

	ciphertext, cerr := crypt.Vault().Encrypt(conceal.New(url))
	must.NoError(t, cerr)

	id, err := store.Documents().CreateWeblink(ctx, &sitecontent.CreateWeblinkRequest{
		User:      user,
		Folder:    miscFolderOf(t, ctx, store, user),
		Document:  doc,
		URL:       url,
		Encrypted: ciphertext,
	})
	must.NoError(t, err)
	return id
}

func insertUsername(t *testing.T, ctx scope.C, store Store, user users.Identity, doc, username string, crypt mocks.TestCrypt) records.UsernamesAuto {
	t.Helper()

	ciphertext, cerr := crypt.Vault().Encrypt(conceal.New(username))
	must.NoError(t, cerr)

	id, err := store.Documents().CreateUsername(ctx, &sitecontent.CreateUsernameRequest{
		User:      user,
		Folder:    miscFolderOf(t, ctx, store, user),
		Document:  doc,
		Username:  username,
		Encrypted: ciphertext,
	})
	must.NoError(t, err)

	return id
}

func insertPassword(t *testing.T, ctx scope.C, store Store, user users.Identity, doc, password string, crypt mocks.TestCrypt) records.PasswordsAuto {
	t.Helper()

	ciphertext, cerr := crypt.Vault().Encrypt(conceal.New(password))
	must.NoError(t, cerr)

	id, err := store.Documents().CreatePassword(ctx, &sitecontent.CreatePasswordRequest{
		User:      user,
		Folder:    miscFolderOf(t, ctx, store, user),
		Document:  doc,
		Encrypted: ciphertext,
	})
	must.NoError(t, err)

	return id
}

func insertNote(t *testing.T, ctx scope.C, store Store, user users.Identity, doc, note string, crypt mocks.TestCrypt) records.NotesAuto {
	t.Helper()

	ciphertext, cerr := crypt.Vault().Encrypt(conceal.New(note))
	must.NoError(t, cerr)

	id, err := store.Documents().CreateNote(ctx, &sitecontent.CreateNoteRequest{
		User:      user,
		Folder:    miscFolderOf(t, ctx, store, user),
		Document:  doc,
		Encrypted: ciphertext,
	})
	must.NoError(t, err)

	return id
}

func TestDocuments_CreateWeblink(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", crypt)
		must.Positive(t, id)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Weblinks)
		must.Eq(t, "https://example.com", doc.Weblinks[0].Unlock(crypt.Vault()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("https://attacker.example.com"))
		must.NoError(t, cerr)

		_, err := db.Documents().CreateWeblink(ctx, &sitecontent.CreateWeblinkRequest{
			User:      bob,
			Folder:    "miscellaneous",
			Document:  "github",
			URL:       "https://attacker.example.com",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})

	t.Run("nonexistent document", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("https://nope.example.com"))
		must.NoError(t, cerr)

		_, err := db.Documents().CreateWeblink(ctx, &sitecontent.CreateWeblinkRequest{
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "nope",
			URL:       "https://nope.example.com",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})
}

func TestDocuments_UpdateWeblink(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://original.example.com", crypt)

		newCiphertext, cerr := crypt.Vault().Encrypt(conceal.New("https://updated.example.com"))
		must.NoError(t, cerr)

		uerr := db.Documents().UpdateWeblink(ctx, &sitecontent.UpdateWeblinkRequest{
			ID:        id,
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "github",
			URL:       "https://updated.example.com",
			Encrypted: newCiphertext,
		})
		must.NoError(t, uerr)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Weblinks)
		must.Eq(t, "https://updated.example.com", doc.Weblinks[0].Unlock(crypt.Vault()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", crypt)

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("https://attacker.example.com"))
		must.NoError(t, cerr)

		err := db.Documents().UpdateWeblink(ctx, &sitecontent.UpdateWeblinkRequest{
			ID:        id,
			User:      bob,
			Folder:    "miscellaneous",
			Document:  "github",
			URL:       "https://attacker.example.com",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})

	t.Run("nonexistent weblink", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("https://nowhere.example.com"))
		must.NoError(t, cerr)

		err := db.Documents().UpdateWeblink(ctx, &sitecontent.UpdateWeblinkRequest{
			ID:        records.WeblinksAuto(9999),
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "github",
			URL:       "https://nowhere.example.com",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})
}

func TestDocuments_DeleteWeblink(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", crypt)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Weblinks)

		derr := db.Documents().DeleteWeblink(ctx, &sitecontent.DeleteWeblinkRequest{
			ID:       doc.Weblinks[0].ID,
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.NoError(t, derr)

		doc, err = db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 0, doc.Weblinks)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", crypt)

		err := db.Documents().DeleteWeblink(ctx, &sitecontent.DeleteWeblinkRequest{
			ID:       id,
			User:     bob,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.Error(t, err)
	})

	t.Run("nonexistent weblink", func(t *testing.T) {
		db, ctx := openTestDB(t)

		err := db.Documents().DeleteWeblink(ctx, &sitecontent.DeleteWeblinkRequest{
			ID:       records.WeblinksAuto(9999),
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.Error(t, err)
	})
}

func TestDocuments_CreateUsername(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", crypt)
		must.Positive(t, id)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Usernames)
		must.Eq(t, "alice", doc.Usernames[0].Unlock(crypt.Vault()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("mallory"))
		must.NoError(t, cerr)

		_, err := db.Documents().CreateUsername(ctx, &sitecontent.CreateUsernameRequest{
			User:      bob,
			Folder:    "miscellaneous",
			Document:  "github",
			Username:  "mallory",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})

	t.Run("nonexistent document", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("stranger"))
		must.NoError(t, cerr)

		_, err := db.Documents().CreateUsername(ctx, &sitecontent.CreateUsernameRequest{
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "nope",
			Username:  "stranger",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})
}

func TestDocuments_UpdateUsername(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", crypt)

		newCiphertext, cerr := crypt.Vault().Encrypt(conceal.New("alice2"))
		must.NoError(t, cerr)

		uerr := db.Documents().UpdateUsername(ctx, &sitecontent.UpdateUsernameRequest{
			ID:        id,
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "github",
			Username:  "alice2",
			Encrypted: newCiphertext,
		})
		must.NoError(t, uerr)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Usernames)
		must.Eq(t, "alice2", doc.Usernames[0].Unlock(crypt.Vault()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", crypt)

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("mallory"))
		must.NoError(t, cerr)

		err := db.Documents().UpdateUsername(ctx, &sitecontent.UpdateUsernameRequest{
			ID:        id,
			User:      bob,
			Folder:    "miscellaneous",
			Document:  "github",
			Username:  "mallory",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})

	t.Run("nonexistent username", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("nobody"))
		must.NoError(t, cerr)

		err := db.Documents().UpdateUsername(ctx, &sitecontent.UpdateUsernameRequest{
			ID:        records.UsernamesAuto(9999),
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "github",
			Username:  "nobody",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})
}

func TestDocuments_DeleteUsername(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", crypt)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Usernames)

		derr := db.Documents().DeleteUsername(ctx, &sitecontent.DeleteUsernameRequest{
			ID:       doc.Usernames[0].ID,
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.NoError(t, derr)

		doc, err = db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceEmpty(t, doc.Usernames)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", crypt)

		err := db.Documents().DeleteUsername(ctx, &sitecontent.DeleteUsernameRequest{
			ID:       id,
			User:     bob,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.Error(t, err)
	})

	t.Run("nonexistent username", func(t *testing.T) {
		db, ctx := openTestDB(t)

		err := db.Documents().DeleteUsername(ctx, &sitecontent.DeleteUsernameRequest{
			ID:       records.UsernamesAuto(9999),
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.Error(t, err)
	})
}

func TestDocuments_CreatePassword(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertPassword(t, ctx, db, mocks.Nobody, "github", "s3cret!", crypt)
		must.Positive(t, id)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Passwords)
		must.Eq(t, "s3cret!", doc.Passwords[0].Unlock(crypt.Vault()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("evilpass"))
		must.NoError(t, cerr)

		_, err := db.Documents().CreatePassword(ctx, &sitecontent.CreatePasswordRequest{
			User:      bob,
			Folder:    "miscellaneous",
			Document:  "github",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})

	t.Run("nonexistent document", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("nope"))
		must.NoError(t, cerr)

		_, err := db.Documents().CreatePassword(ctx, &sitecontent.CreatePasswordRequest{
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "nope",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})
}

func TestDocuments_UpdatePassword(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertPassword(t, ctx, db, mocks.Nobody, "github", "oldpass", crypt)

		newCiphertext, cerr := crypt.Vault().Encrypt(conceal.New("newpass1"))
		must.NoError(t, cerr)

		uerr := db.Documents().UpdatePassword(ctx, &sitecontent.UpdatePasswordRequest{
			ID:        id,
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "github",
			Encrypted: newCiphertext,
		})
		must.NoError(t, uerr)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Passwords)
		must.Eq(t, "newpass1", doc.Passwords[0].Unlock(crypt.Vault()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertPassword(t, ctx, db, mocks.Nobody, "github", "realpass", crypt)

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("hacked!"))
		must.NoError(t, cerr)

		err := db.Documents().UpdatePassword(ctx, &sitecontent.UpdatePasswordRequest{
			ID:        id,
			User:      bob,
			Folder:    "miscellaneous",
			Document:  "github",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})

	t.Run("nonexistent password", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("nothing"))
		must.NoError(t, cerr)

		err := db.Documents().UpdatePassword(ctx, &sitecontent.UpdatePasswordRequest{
			ID:        records.PasswordsAuto(9999),
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "github",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})
}

func TestDocuments_DeletePassword(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertPassword(t, ctx, db, mocks.Nobody, "github", "secret", crypt)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Passwords)

		derr := db.Documents().DeletePassword(ctx, &sitecontent.DeletePasswordRequest{
			ID:       doc.Passwords[0].ID,
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.NoError(t, derr)

		doc, err = db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 0, doc.Passwords)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertPassword(t, ctx, db, mocks.Nobody, "github", "realpass", crypt)

		err := db.Documents().DeletePassword(ctx, &sitecontent.DeletePasswordRequest{
			ID:       id,
			User:     bob,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.Error(t, err)
	})

	t.Run("nonexistent password", func(t *testing.T) {
		db, ctx := openTestDB(t)

		err := db.Documents().DeletePassword(ctx, &sitecontent.DeletePasswordRequest{
			ID:       records.PasswordsAuto(9999),
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.Error(t, err)
	})
}

func TestDocuments_CreateNote(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertNote(t, ctx, db, mocks.Nobody, "github", "secret sauce", crypt)
		must.Positive(t, id)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.NotNil(t, doc.Note)
		must.Eq(t, "secret sauce", doc.Note.Unlock(crypt.Vault()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob-note")
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("hacker note"))
		must.NoError(t, cerr)

		_, err := db.Documents().CreateNote(ctx, &sitecontent.CreateNoteRequest{
			User:      bob,
			Folder:    "miscellaneous",
			Document:  "github",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})

	t.Run("nonexistent document", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("anything"))
		must.NoError(t, cerr)

		_, err := db.Documents().CreateNote(ctx, &sitecontent.CreateNoteRequest{
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "nope",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})

	t.Run("already exists", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertNote(t, ctx, db, mocks.Nobody, "github", "first", crypt)

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("second"))
		must.NoError(t, cerr)

		_, err := db.Documents().CreateNote(ctx, &sitecontent.CreateNoteRequest{
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "github",
			Encrypted: ciphertext,
		})
		must.Error(t, err)

		// original note is preserved
		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.NotNil(t, doc.Note)
		must.Eq(t, "first", doc.Note.Unlock(crypt.Vault()))
	})
}

func TestDocuments_UpdateNote(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertNote(t, ctx, db, mocks.Nobody, "github", "oldtext", crypt)

		newCiphertext, cerr := crypt.Vault().Encrypt(conceal.New("newtext"))
		must.NoError(t, cerr)

		uerr := db.Documents().UpdateNote(ctx, &sitecontent.UpdateNoteRequest{
			ID:        id,
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "github",
			Encrypted: newCiphertext,
		})
		must.NoError(t, uerr)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.NotNil(t, doc.Note)
		must.Eq(t, "newtext", doc.Note.Unlock(crypt.Vault()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob-note")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertNote(t, ctx, db, mocks.Nobody, "github", "realtext", crypt)

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("hacked"))
		must.NoError(t, cerr)

		err := db.Documents().UpdateNote(ctx, &sitecontent.UpdateNoteRequest{
			ID:        id,
			User:      bob,
			Folder:    "miscellaneous",
			Document:  "github",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})

	t.Run("nonexistent note", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := crypt.Vault().Encrypt(conceal.New("anything"))
		must.NoError(t, cerr)

		err := db.Documents().UpdateNote(ctx, &sitecontent.UpdateNoteRequest{
			ID:        records.NotesAuto(9999),
			User:      mocks.Nobody,
			Folder:    "miscellaneous",
			Document:  "github",
			Encrypted: ciphertext,
		})
		must.Error(t, err)
	})
}

func TestDocuments_DeleteNote(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertNote(t, ctx, db, mocks.Nobody, "github", "secret", crypt)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.NotNil(t, doc.Note)

		derr := db.Documents().DeleteNote(ctx, &sitecontent.DeleteNoteRequest{
			ID:       doc.Note.ID,
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.NoError(t, derr)

		doc, err = db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.Nil(t, doc.Note)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob-note")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertNote(t, ctx, db, mocks.Nobody, "github", "realtext", crypt)

		err := db.Documents().DeleteNote(ctx, &sitecontent.DeleteNoteRequest{
			ID:       id,
			User:     bob,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.Error(t, err)
	})

	t.Run("nonexistent note", func(t *testing.T) {
		db, ctx := openTestDB(t)

		// document exists but no note row matches the bogus id
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		err := db.Documents().DeleteNote(ctx, &sitecontent.DeleteNoteRequest{
			ID:       records.NotesAuto(9999),
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.Error(t, err)
	})
}

func TestDocuments_GetNote(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertNote(t, ctx, db, mocks.Nobody, "github", "hello\nworld", crypt)

		note, err := db.Documents().GetNote(ctx, &sitecontent.GetNoteRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.NotNil(t, note)
		must.Eq(t, "hello\nworld", note.Unlock(crypt.Vault()))
	})

	t.Run("no note", func(t *testing.T) {
		db, ctx := openTestDB(t)
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		note, err := db.Documents().GetNote(ctx, &sitecontent.GetNoteRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.Nil(t, note)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		bob := insertUser(t, ctx, db, "bob-note")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertNote(t, ctx, db, mocks.Nobody, "github", "secret", crypt)

		note, err := db.Documents().GetNote(ctx, &sitecontent.GetNoteRequest{
			User: bob, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.Nil(t, note)
	})
}

func TestDocuments_Count(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	d := db.Documents()

	count, err := d.Count(ctx)
	must.NoError(t, err)
	must.Zero(t, count)
}

func TestDocuments_Resolve(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	d := db.Documents()
	id := insertDocument(t, ctx, db, mocks.Nobody, "github")

	t.Run("found", func(t *testing.T) {
		got, err := d.Resolve(ctx, &sitecontent.GetDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
		})
		must.NoError(t, err)
		must.Eq(t, id, got)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := d.Resolve(ctx, &sitecontent.GetDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "missing",
		})
		must.Error(t, err)
	})
}

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
		Slug:   sitecontent.Slugify(label),
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
			Slug:   "github",
		}
		_, err := d.Create(ctx, request)
		must.Error(t, err)
	})

	t.Run("duplicate slug", func(t *testing.T) {
		// "My Doc" and "my-doc" differ as labels but slugify to the same
		// value; the second must be rejected as a duplicate
		must.Positive(t, insertDocument(t, ctx, db, mocks.Nobody, "My Doc"))

		_, err := d.Create(ctx, &sitecontent.CreateDocumentRequest{
			User:   mocks.Nobody,
			Folder: miscFolderOf(t, ctx, db, mocks.Nobody),
			Label:  "my-doc",
			Slug:   "my-doc",
		})
		must.Error(t, err)
	})
}

func TestDocuments_Get_Slug(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	d := db.Documents()

	// create a folder (folder names may not contain spaces), then a document
	// inside it with spaces in its display name
	must.NoError(t, db.Folders().Create(ctx, &sitecontent.CreateFolderRequest{
		User:  nobody,
		Label: "WorkAccounts",
		Slug:  "workaccounts",
		Icon:  sitecontent.Icon(0),
	}))

	_, err := d.Create(ctx, &sitecontent.CreateDocumentRequest{
		User:   nobody,
		Folder: "workaccounts",
		Label:  "Git Hub",
		Slug:   "git-hub",
	})
	must.NoError(t, err)

	// the document resolves through slugs, while Label/Folder keep the display names
	doc, gerr := d.Get(ctx, &sitecontent.GetDocumentRequest{
		User:     nobody,
		Folder:   "workaccounts",
		Document: "git-hub",
	})
	must.NoError(t, gerr)
	must.Eq(t, "Git Hub", doc.Label)
	must.Eq(t, "git-hub", doc.Slug)
	must.Eq(t, "WorkAccounts", doc.Folder)
	must.Eq(t, "workaccounts", doc.FolderSlug)

	// a listing also carries the slug for URL construction
	list, lerr := d.List(ctx, &sitecontent.ListDocumentsRequest{
		User:   nobody,
		Folder: "workaccounts",
	})
	must.NoError(t, lerr)
	must.SliceLen(t, 1, list)
	must.Eq(t, "Git Hub", list[0].Label)
	must.Eq(t, "git-hub", list[0].Slug)
}

func TestDocuments_List(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	d := db.Documents()
	folder := miscFolderOf(t, ctx, db, mocks.Nobody)

	t.Run("empty", func(t *testing.T) {
		list, err := d.List(ctx, &sitecontent.ListDocumentsRequest{
			User:   mocks.Nobody,
			Folder: folder,
		})
		must.NoError(t, err)
		must.SliceEmpty(t, list)
	})

	t.Run("after inserts", func(t *testing.T) {
		id1 := insertDocument(t, ctx, db, mocks.Nobody, "github")

		// sleep before inserting second document
		time.Sleep(time.Second)

		// insert second document
		id2 := insertDocument(t, ctx, db, mocks.Nobody, "gitlab")

		list, err := d.List(ctx, &sitecontent.ListDocumentsRequest{
			User:   mocks.Nobody,
			Folder: folder,
		})
		must.NoError(t, err)
		must.SliceLen(t, 2, list)

		// most recent insert appears first
		must.Eq(t, id2, list[0].ID)
		must.Eq(t, "gitlab", list[0].Label)

		// first insert appears last
		must.Eq(t, id1, list[1].ID)
		must.Eq(t, "github", list[1].Label)
	})
}

func TestDocuments_Delete(t *testing.T) {
	t.Parallel()

	t.Run("happy path cascades children", func(t *testing.T) {
		db, ctx := openTestDB(t)
		crypt := mocks.NewTestCrypt()

		id := insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", crypt)
		insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", crypt)
		insertPassword(t, ctx, db, mocks.Nobody, "github", "s3cret!", crypt)
		insertNote(t, ctx, db, mocks.Nobody, "github", "hi", crypt)

		err := db.Documents().Delete(ctx, &sitecontent.DeleteDocumentRequest{
			ID:   id,
			User: mocks.Nobody,
		})
		must.NoError(t, err)

		// the document itself is gone
		count, cerr := db.Documents().Count(ctx)
		must.NoError(t, cerr)
		must.Zero(t, count)

		// and so are its child rows (FK on delete cascade handles this)
		doc, gerr := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.Error(t, gerr)
		must.Nil(t, doc)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)

		id := insertDocument(t, ctx, db, mocks.Nobody, "github")
		bob := insertUser(t, ctx, db, "bob")

		err := db.Documents().Delete(ctx, &sitecontent.DeleteDocumentRequest{
			ID:   id,
			User: bob,
		})
		must.Error(t, err)

		// the document is still present
		count, cerr := db.Documents().Count(ctx)
		must.NoError(t, cerr)
		must.One(t, count)
	})

	t.Run("unknown document", func(t *testing.T) {
		db, ctx := openTestDB(t)

		err := db.Documents().Delete(ctx, &sitecontent.DeleteDocumentRequest{
			ID:   424242,
			User: mocks.Nobody,
		})
		must.Error(t, err)
	})
}
