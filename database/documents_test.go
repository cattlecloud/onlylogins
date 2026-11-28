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

func insertWeblink(t *testing.T, ctx scope.C, store Store, user users.Identity, doc, url string, cypher mocks.TestCypher) records.WeblinksAuto {
	t.Helper()

	ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New(url))
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

func insertUsername(t *testing.T, ctx scope.C, store Store, user users.Identity, doc, username string, cypher mocks.TestCypher) records.UsernamesAuto {
	t.Helper()

	ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New(username))
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

func insertPassword(t *testing.T, ctx scope.C, store Store, user users.Identity, doc, password string, cypher mocks.TestCypher) records.PasswordsAuto {
	t.Helper()

	ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New(password))
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

func insertNote(t *testing.T, ctx scope.C, store Store, user users.Identity, doc, note string, cypher mocks.TestCypher) records.NotesAuto {
	t.Helper()

	ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New(note))
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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", cypher)
		must.Positive(t, id)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Weblinks)
		must.Eq(t, "https://example.com", doc.Weblinks[0].Unlock(cypher.Cypher()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("https://attacker.example.com"))
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
		cypher := mocks.NewTestCypher()

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("https://nope.example.com"))
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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://original.example.com", cypher)

		newCiphertext, cerr := cypher.Cypher().Encrypt(conceal.New("https://updated.example.com"))
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
		must.Eq(t, "https://updated.example.com", doc.Weblinks[0].Unlock(cypher.Cypher()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", cypher)

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("https://attacker.example.com"))
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
		cypher := mocks.NewTestCypher()
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("https://nowhere.example.com"))
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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", cypher)

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
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", cypher)

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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", cypher)
		must.Positive(t, id)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Usernames)
		must.Eq(t, "alice", doc.Usernames[0].Unlock(cypher.Cypher()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("mallory"))
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
		cypher := mocks.NewTestCypher()

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("stranger"))
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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", cypher)

		newCiphertext, cerr := cypher.Cypher().Encrypt(conceal.New("alice2"))
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
		must.Eq(t, "alice2", doc.Usernames[0].Unlock(cypher.Cypher()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", cypher)

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("mallory"))
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
		cypher := mocks.NewTestCypher()
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("nobody"))
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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", cypher)

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
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", cypher)

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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertPassword(t, ctx, db, mocks.Nobody, "github", "s3cret!", cypher)
		must.Positive(t, id)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 1, doc.Passwords)
		must.Eq(t, "s3cret!", doc.Passwords[0].Unlock(cypher.Cypher()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("evilpass"))
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
		cypher := mocks.NewTestCypher()

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("nope"))
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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertPassword(t, ctx, db, mocks.Nobody, "github", "oldpass", cypher)

		newCiphertext, cerr := cypher.Cypher().Encrypt(conceal.New("newpass1"))
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
		must.Eq(t, "newpass1", doc.Passwords[0].Unlock(cypher.Cypher()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertPassword(t, ctx, db, mocks.Nobody, "github", "realpass", cypher)

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("hacked!"))
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
		cypher := mocks.NewTestCypher()
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("nothing"))
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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertPassword(t, ctx, db, mocks.Nobody, "github", "secret", cypher)

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
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertPassword(t, ctx, db, mocks.Nobody, "github", "realpass", cypher)

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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertNote(t, ctx, db, mocks.Nobody, "github", "secret sauce", cypher)
		must.Positive(t, id)

		doc, err := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.NotNil(t, doc.Note)
		must.Eq(t, "secret sauce", doc.Note.Unlock(cypher.Cypher()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob-note")
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("hacker note"))
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
		cypher := mocks.NewTestCypher()

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("anything"))
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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertNote(t, ctx, db, mocks.Nobody, "github", "first", cypher)

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("second"))
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
		must.Eq(t, "first", doc.Note.Unlock(cypher.Cypher()))
	})
}

func TestDocuments_UpdateNote(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertNote(t, ctx, db, mocks.Nobody, "github", "oldtext", cypher)

		newCiphertext, cerr := cypher.Cypher().Encrypt(conceal.New("newtext"))
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
		must.Eq(t, "newtext", doc.Note.Unlock(cypher.Cypher()))
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob-note")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertNote(t, ctx, db, mocks.Nobody, "github", "realtext", cypher)

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("hacked"))
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
		cypher := mocks.NewTestCypher()
		insertDocument(t, ctx, db, mocks.Nobody, "github")

		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New("anything"))
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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertNote(t, ctx, db, mocks.Nobody, "github", "secret", cypher)

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
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob-note")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		id := insertNote(t, ctx, db, mocks.Nobody, "github", "realtext", cypher)

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
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertNote(t, ctx, db, mocks.Nobody, "github", "hello\nworld", cypher)

		note, err := db.Documents().GetNote(ctx, &sitecontent.GetNoteRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.NotNil(t, note)
		must.Eq(t, "hello\nworld", note.Unlock(cypher.Cypher()))
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
		cypher := mocks.NewTestCypher()

		bob := insertUser(t, ctx, db, "bob-note")
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertNote(t, ctx, db, mocks.Nobody, "github", "secret", cypher)

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
		id2 := insertDocument(t, ctx, db, mocks.Nobody, "gitlab")

		list, err := d.List(ctx, &sitecontent.ListDocumentsRequest{
			User:   mocks.Nobody,
			Folder: folder,
		})
		must.NoError(t, err)
		must.SliceLen(t, 2, list)

		// alphabetical: "github" sorts before "gitlab"
		must.Eq(t, id1, list[0].ID)
		must.Eq(t, "github", list[0].Label)

		must.Eq(t, id2, list[1].ID)
		must.Eq(t, "gitlab", list[1].Label)
	})
}

func TestDocuments_List_CaseInsensitive(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	d := db.Documents()
	folder := miscFolderOf(t, ctx, db, mocks.Nobody)

	// create documents with mixed casing; ordering must be case-insensitive,
	// so "apple" sorts before "Banana" and both sort before "Zebra"
	for _, label := range []string{"Zebra", "apple", "Banana", "cherry"} {
		insertDocument(t, ctx, db, mocks.Nobody, label)
	}

	list, err := d.List(ctx, &sitecontent.ListDocumentsRequest{
		User:   mocks.Nobody,
		Folder: folder,
	})
	must.NoError(t, err)
	must.SliceLen(t, 4, list)

	must.Eq(t, "apple", list[0].Label)
	must.Eq(t, "Banana", list[1].Label)
	must.Eq(t, "cherry", list[2].Label)
	must.Eq(t, "Zebra", list[3].Label)
}

func TestDocuments_List_HasWeblinks(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	d := db.Documents()
	cypher := mocks.NewTestCypher()
	folder := miscFolderOf(t, ctx, db, mocks.Nobody)

	insertDocument(t, ctx, db, mocks.Nobody, "empty")

	insertDocument(t, ctx, db, mocks.Nobody, "linked")
	insertWeblink(t, ctx, db, mocks.Nobody, "linked", "https://example.com", cypher)

	list, err := d.List(ctx, &sitecontent.ListDocumentsRequest{
		User:   mocks.Nobody,
		Folder: folder,
	})
	must.NoError(t, err)
	must.SliceLen(t, 2, list)

	// alphabetical: "empty" sorts before "linked"
	must.Eq(t, "empty", list[0].Label)
	must.False(t, list[0].HasWeblinks)

	must.Eq(t, "linked", list[1].Label)
	must.True(t, list[1].HasWeblinks)
}

func TestDocuments_ListWeblinks(t *testing.T) {
	t.Parallel()

	t.Run("ordered by id", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://first.example.com", cypher)
		insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://second.example.com", cypher)

		links, err := db.Documents().ListWeblinks(ctx, &sitecontent.ListWeblinksRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 2, links)
		must.Eq(t, "https://first.example.com", links[0].Unlock(cypher.Cypher()))
		must.Eq(t, "https://second.example.com", links[1].Unlock(cypher.Cypher()))
	})

	t.Run("no weblinks", func(t *testing.T) {
		db, ctx := openTestDB(t)

		insertDocument(t, ctx, db, mocks.Nobody, "github")

		links, err := db.Documents().ListWeblinks(ctx, &sitecontent.ListWeblinksRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceEmpty(t, links)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", cypher)
		bob := insertUser(t, ctx, db, "bob")

		links, err := db.Documents().ListWeblinks(ctx, &sitecontent.ListWeblinksRequest{
			User: bob, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceEmpty(t, links)
	})
}

func TestDocuments_List_HasPasswords(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	d := db.Documents()
	cypher := mocks.NewTestCypher()
	folder := miscFolderOf(t, ctx, db, mocks.Nobody)

	insertDocument(t, ctx, db, mocks.Nobody, "empty")

	insertDocument(t, ctx, db, mocks.Nobody, "secured")
	insertPassword(t, ctx, db, mocks.Nobody, "secured", "s3cret!", cypher)

	list, err := d.List(ctx, &sitecontent.ListDocumentsRequest{
		User:   mocks.Nobody,
		Folder: folder,
	})
	must.NoError(t, err)
	must.SliceLen(t, 2, list)

	// alphabetical: "empty" sorts before "secured"
	must.Eq(t, "empty", list[0].Label)
	must.False(t, list[0].HasPasswords)

	must.Eq(t, "secured", list[1].Label)
	must.True(t, list[1].HasPasswords)
}

func TestDocuments_ListPasswords(t *testing.T) {
	t.Parallel()

	t.Run("ordered by id", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertPassword(t, ctx, db, mocks.Nobody, "github", "first", cypher)
		insertPassword(t, ctx, db, mocks.Nobody, "github", "second", cypher)

		passwords, err := db.Documents().ListPasswords(ctx, &sitecontent.ListPasswordsRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceLen(t, 2, passwords)
		must.Eq(t, "first", passwords[0].Unlock(cypher.Cypher()))
		must.Eq(t, "second", passwords[1].Unlock(cypher.Cypher()))
	})

	t.Run("no passwords", func(t *testing.T) {
		db, ctx := openTestDB(t)

		insertDocument(t, ctx, db, mocks.Nobody, "github")

		passwords, err := db.Documents().ListPasswords(ctx, &sitecontent.ListPasswordsRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceEmpty(t, passwords)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertPassword(t, ctx, db, mocks.Nobody, "github", "s3cret!", cypher)
		bob := insertUser(t, ctx, db, "bob")

		passwords, err := db.Documents().ListPasswords(ctx, &sitecontent.ListPasswordsRequest{
			User: bob, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)
		must.SliceEmpty(t, passwords)
	})
}

func TestDocuments_Delete(t *testing.T) {
	t.Parallel()

	t.Run("happy path cascades children", func(t *testing.T) {
		db, ctx := openTestDB(t)
		cypher := mocks.NewTestCypher()

		id := insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertWeblink(t, ctx, db, mocks.Nobody, "github", "https://example.com", cypher)
		insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", cypher)
		insertPassword(t, ctx, db, mocks.Nobody, "github", "s3cret!", cypher)
		insertNote(t, ctx, db, mocks.Nobody, "github", "hi", cypher)

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

func TestDocuments_Rename(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", cypher)

		before, err := d.Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)

		// ensure the modification timestamp advances by a full second
		time.Sleep(time.Second)

		rerr := d.Rename(ctx, &sitecontent.RenameDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
			Label:    "gitlab",
			NewSlug:  "gitlab",
		})
		must.NoError(t, rerr)

		// the new slug resolves with the new label and a bumped timestamp
		after, gerr := d.Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "gitlab",
		})
		must.NoError(t, gerr)
		must.Eq(t, "gitlab", after.Label)
		must.True(t, after.Modification.After(before.Modification))

		// the child row survived the rename (keyed by document id, not slug)
		must.SliceLen(t, 1, after.Usernames)

		// the old slug no longer resolves
		_, oerr := d.Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.Error(t, oerr)
	})

	t.Run("case-only rename", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		insertDocument(t, ctx, db, mocks.Nobody, "github")

		must.NoError(t, d.Rename(ctx, &sitecontent.RenameDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
			Label:    "GitHub",
			NewSlug:  "github",
		}))

		doc, gerr := d.Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, gerr)
		must.Eq(t, "GitHub", doc.Label)
	})

	t.Run("duplicate label", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertDocument(t, ctx, db, mocks.Nobody, "gitlab")

		err := d.Rename(ctx, &sitecontent.RenameDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
			Label:    "gitlab",
			NewSlug:  "gitlab",
		})
		must.ErrorIs(t, err, ErrDocumentExists)
	})

	t.Run("duplicate slug", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		insertDocument(t, ctx, db, mocks.Nobody, "My Doc") // slug "my-doc"
		insertDocument(t, ctx, db, mocks.Nobody, "second")

		err := d.Rename(ctx, &sitecontent.RenameDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "second",
			Label:    "my-doc",
			NewSlug:  "my-doc",
		})
		must.ErrorIs(t, err, ErrDocumentExists)
	})

	t.Run("unknown document", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		err := d.Rename(ctx, &sitecontent.RenameDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "nope",
			Label:    "x",
			NewSlug:  "x",
		})
		must.Error(t, err)
	})

	t.Run("unknown folder", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		insertDocument(t, ctx, db, mocks.Nobody, "github")

		err := d.Rename(ctx, &sitecontent.RenameDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "no-such-folder",
			Document: "github",
			Label:    "gitlab",
			NewSlug:  "gitlab",
		})
		must.Error(t, err)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		bob := insertUser(t, ctx, db, "bob")

		err := d.Rename(ctx, &sitecontent.RenameDocumentRequest{
			User:     bob,
			Folder:   "miscellaneous",
			Document: "github",
			Label:    "gitlab",
			NewSlug:  "gitlab",
		})
		must.Error(t, err)
	})
}

func TestDocuments_Migrate(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()
		cypher := mocks.NewTestCypher()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertUsername(t, ctx, db, mocks.Nobody, "github", "alice", cypher)

		must.NoError(t, db.Folders().Create(ctx, &sitecontent.CreateFolderRequest{
			User:  mocks.Nobody,
			Label: "Work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))

		before, err := d.Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.NoError(t, err)

		// ensure the modification timestamp advances by a full second
		time.Sleep(time.Second)

		merr := d.Migrate(ctx, &sitecontent.MigrateDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
			Target:   "work",
		})
		must.NoError(t, merr)

		// the document now resolves under the new folder with a bumped timestamp
		after, gerr := d.Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "work", Document: "github",
		})
		must.NoError(t, gerr)
		must.Eq(t, "Work", after.Folder)
		must.Eq(t, "work", after.FolderSlug)
		must.True(t, after.Modification.After(before.Modification))

		// the child row survived the migration (keyed by document id, not folder)
		must.SliceLen(t, 1, after.Usernames)

		// the old folder path no longer resolves
		_, oerr := d.Get(ctx, &sitecontent.GetDocumentRequest{
			User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
		})
		must.Error(t, oerr)
	})

	t.Run("duplicate label", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		insertDocument(t, ctx, db, mocks.Nobody, "github")

		must.NoError(t, db.Folders().Create(ctx, &sitecontent.CreateFolderRequest{
			User:  mocks.Nobody,
			Label: "Work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))
		_, cerr := d.Create(ctx, &sitecontent.CreateDocumentRequest{
			User:   mocks.Nobody,
			Folder: "work",
			Label:  "github",
			Slug:   "github",
		})
		must.NoError(t, cerr)

		err := d.Migrate(ctx, &sitecontent.MigrateDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
			Target:   "work",
		})
		must.ErrorIs(t, err, ErrDocumentExists)
	})

	t.Run("duplicate slug", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		insertDocument(t, ctx, db, mocks.Nobody, "Git Hub") // slug "git-hub"

		must.NoError(t, db.Folders().Create(ctx, &sitecontent.CreateFolderRequest{
			User:  mocks.Nobody,
			Label: "Work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))
		_, cerr := d.Create(ctx, &sitecontent.CreateDocumentRequest{
			User:   mocks.Nobody,
			Folder: "work",
			Label:  "git-hub",
			Slug:   "git-hub",
		})
		must.NoError(t, cerr)

		err := d.Migrate(ctx, &sitecontent.MigrateDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "git-hub",
			Target:   "work",
		})
		must.ErrorIs(t, err, ErrDocumentExists)
	})

	t.Run("unknown document", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		must.NoError(t, db.Folders().Create(ctx, &sitecontent.CreateFolderRequest{
			User:  mocks.Nobody,
			Label: "Work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))

		err := d.Migrate(ctx, &sitecontent.MigrateDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "nope",
			Target:   "work",
		})
		must.Error(t, err)
	})

	t.Run("unknown target folder", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		insertDocument(t, ctx, db, mocks.Nobody, "github")

		err := d.Migrate(ctx, &sitecontent.MigrateDocumentRequest{
			User:     mocks.Nobody,
			Folder:   "miscellaneous",
			Document: "github",
			Target:   "no-such-folder",
		})
		must.Error(t, err)
	})

	t.Run("wrong user", func(t *testing.T) {
		db, ctx := openTestDB(t)
		d := db.Documents()

		insertDocument(t, ctx, db, mocks.Nobody, "github")
		bob := insertUser(t, ctx, db, "bob")

		must.NoError(t, db.Folders().Create(ctx, &sitecontent.CreateFolderRequest{
			User:  bob,
			Label: "Work",
			Slug:  "work",
			Icon:  sitecontent.Icon(0),
		}))

		// bob cannot migrate nobody's document
		err := d.Migrate(ctx, &sitecontent.MigrateDocumentRequest{
			User:     bob,
			Folder:   "miscellaneous",
			Document: "github",
			Target:   "work",
		})
		must.Error(t, err)
	})
}
