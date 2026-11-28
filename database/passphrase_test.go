// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"testing"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

// insertPassphraseUser creates a user whose passphrase and derived key are
// known to the caller, returning the user id and the derived encryption key.
func insertPassphraseUser(t *testing.T, ctx scope.C, db Store, username, passphrase string) (users.Identity, *conceal.Bytes) {
	t.Helper()

	salt := encrypt.Salt()
	pass := conceal.New(passphrase)
	key := encrypt.DeriveKey(pass, salt)
	verifier := encrypt.MakeVerifier(key)

	id, err := db.Users().Create(ctx, &users.CreateUserRequest{
		Email:      username + "@example.com",
		Username:   username,
		Salt:       salt,
		Passphrase: pass,
	}, verifier)
	must.NoError(t, err)
	must.Positive(t, id)

	return id, key
}

func encryptForTest(t *testing.T, v encrypt.Cypher, plaintext string) *conceal.Bytes {
	t.Helper()

	ciphertext, err := v.Encrypt(conceal.New(plaintext))
	must.NoError(t, err)
	return ciphertext
}

// seedEncryptedContent adds one of every encrypted content type to the user's
// "github" document, encrypted with the given cypher.
func seedEncryptedContent(t *testing.T, ctx scope.C, db Store, uid users.Identity, cypher encrypt.Cypher) {
	t.Helper()

	const folder = "miscellaneous"
	const document = "github"

	insertDocument(t, ctx, db, uid, document)

	_, err := db.Documents().CreateWeblink(ctx, &sitecontent.CreateWeblinkRequest{
		User:      uid,
		Folder:    folder,
		Document:  document,
		URL:       "https://example.com",
		Encrypted: encryptForTest(t, cypher, "https://example.com"),
	})
	must.NoError(t, err)

	_, err = db.Documents().CreateUsername(ctx, &sitecontent.CreateUsernameRequest{
		User:      uid,
		Folder:    folder,
		Document:  document,
		Username:  "alice-username",
		Encrypted: encryptForTest(t, cypher, "alice-username"),
	})
	must.NoError(t, err)

	_, err = db.Documents().CreatePassword(ctx, &sitecontent.CreatePasswordRequest{
		User:      uid,
		Folder:    folder,
		Document:  document,
		Encrypted: encryptForTest(t, cypher, "hunter2hunter2hunter2"),
	})
	must.NoError(t, err)

	_, err = db.Documents().CreateKeyval(ctx, &sitecontent.CreateKeyvalRequest{
		User:      uid,
		Folder:    folder,
		Document:  document,
		Key:       "api-key",
		Encrypted: encryptForTest(t, cypher, "secret-value"),
	})
	must.NoError(t, err)

	_, err = db.Documents().CreateNote(ctx, &sitecontent.CreateNoteRequest{
		User:      uid,
		Folder:    folder,
		Document:  document,
		Encrypted: encryptForTest(t, cypher, "some note text"),
	})
	must.NoError(t, err)
}

// passphraseUpdateCount returns the number of rows in the passphraseupdates
// audit table.
func passphraseUpdateCount(t *testing.T, db Store, ctx scope.C) int {
	t.Helper()

	st, ok := db.(*store)
	must.True(t, ok)

	tx, done, xerr := st.ldb.StartRead(ctx)
	must.NoError(t, xerr)
	defer done()

	count, err := litesql.QueryRow(ctx, tx, scanCount, `select count(id) from passphraseupdates`)
	must.NoError(t, err)
	return count
}

func TestUsers_ChangePassphrase(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)

	const oldPass = "correct horse battery staple"
	const newPass = "brand new battery horse staple"

	uid, oldKey := insertPassphraseUser(t, ctx, db, "alice", oldPass)
	oldCypher := encrypt.New(oldKey)

	seedEncryptedContent(t, ctx, db, uid, oldCypher)

	must.Zero(t, passphraseUpdateCount(t, db, ctx))

	request := &users.ChangePassphraseRequest{
		Previous:   conceal.New(oldPass),
		Passphrase: conceal.New(newPass),
		Confirm:    conceal.New(newPass),
	}
	must.NoError(t, db.Users().ChangePassphrase(ctx, uid, request))

	// the old passphrase no longer verifies
	_, _, err := db.Users().Check(ctx, &users.LoginRequest{
		Username:   "alice",
		Passphrase: conceal.New(oldPass),
	})
	must.ErrorIs(t, err, ErrIncorrectCredentials)

	// the new passphrase verifies and yields the re-encryption key
	_, newKey, err := db.Users().Check(ctx, &users.LoginRequest{
		Username:   "alice",
		Passphrase: conceal.New(newPass),
	})
	must.NoError(t, err)
	must.NotNil(t, newKey)

	newCypher := encrypt.New(newKey)

	// every content type is now readable with the new key and matches the
	// original plaintext
	doc, derr := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User:     uid,
		Folder:   "miscellaneous",
		Document: "github",
	})
	must.NoError(t, derr)

	must.Eq(t, "https://example.com", doc.Weblinks[0].Unlock(newCypher))
	must.Eq(t, "alice-username", doc.Usernames[0].Unlock(newCypher))
	must.Eq(t, "hunter2hunter2hunter2", doc.Passwords[0].Unlock(newCypher))
	must.Eq(t, "secret-value", doc.Keyvals[0].Unlock(newCypher))
	must.Eq(t, "some note text", doc.Note.Unlock(newCypher))

	// the old key can no longer decrypt the content
	must.Eq(t, sitecontent.ErrNoUnlock, doc.Weblinks[0].Unlock(oldCypher))

	// the change was recorded in the audit table
	must.One(t, passphraseUpdateCount(t, db, ctx))
}

func TestUsers_ChangePassphrase_WrongOldPassphrase(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)

	const oldPass = "correct horse battery staple"
	const newPass = "brand new battery horse staple"

	uid, oldKey := insertPassphraseUser(t, ctx, db, "alice", oldPass)
	oldCypher := encrypt.New(oldKey)

	seedEncryptedContent(t, ctx, db, uid, oldCypher)

	request := &users.ChangePassphraseRequest{
		Previous:   conceal.New("wrong old passphrase here"),
		Passphrase: conceal.New(newPass),
		Confirm:    conceal.New(newPass),
	}
	err := db.Users().ChangePassphrase(ctx, uid, request)
	must.ErrorIs(t, err, ErrIncorrectCredentials)

	// nothing changed: the old passphrase still verifies
	_, key, err := db.Users().Check(ctx, &users.LoginRequest{
		Username:   "alice",
		Passphrase: conceal.New(oldPass),
	})
	must.NoError(t, err)

	// and content is still encrypted with the old key
	doc, derr := db.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User:     uid,
		Folder:   "miscellaneous",
		Document: "github",
	})
	must.NoError(t, derr)
	must.Eq(t, "https://example.com", doc.Weblinks[0].Unlock(encrypt.New(key)))

	// no audit row was recorded
	must.Zero(t, passphraseUpdateCount(t, db, ctx))
}
