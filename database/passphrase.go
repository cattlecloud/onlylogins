// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"database/sql"
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

// authorization is the salt and verifier stored for a user, used to derive and
// verify their per-user encryption key
type authorization struct {
	salt     *conceal.Bytes
	verifier *conceal.Bytes
}

// ChangePassphrase verifies the supplied old passphrase, rotates the user's
// salt and verifier, and re-encrypts all of their content with the newly
// derived key.
//
// On a wrong old passphrase it returns ErrIncorrectCredentials. On any other
// failure the transaction is rolled back and no changes are persisted.
func (s *usersql) ChangePassphrase(ctx scope.C, id users.Identity, request *users.ChangePassphraseRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to change passphrase: %w", xerr)
	}
	defer done()

	// load the current salt and verifier
	auth, aerr := s.loadAuthorization(ctx, tx, id)
	if aerr != nil {
		return fmt.Errorf("unable to change passphrase: %w", aerr)
	}

	// verify the old passphrase before changing anything
	oldKey := encrypt.DeriveKey(request.Previous, auth.salt)
	if !encrypt.Verify(oldKey, auth.verifier) {
		return ErrIncorrectCredentials
	}

	// derive a fresh salt and key for the new passphrase
	newSalt := encrypt.Salt()
	newKey := encrypt.DeriveKey(request.Passphrase, newSalt)
	newVerifier := encrypt.MakeVerifier(newKey)

	oldCypher := encrypt.New(oldKey)
	newCypher := encrypt.New(newKey)

	// re-encrypt every piece of user-owned content with the new key

	const weblinksSQL = `
	select w.id, w.v
	from weblinks w
		join documents d on d.id = w.did
		join folders f on f.id = d.folders_id
	where f.users_id = ?`
	const weblinksUpdateSQL = `update weblinks set v = ? where id = ?`
	if err := s.reencrypt(ctx, tx, oldCypher, newCypher, weblinksSQL, weblinksUpdateSQL, id); err != nil {
		return fmt.Errorf("unable to change passphrase: %w", err)
	}

	const usernamesSQL = `
	select u.id, u.v
	from usernames u
		join documents d on d.id = u.did
		join folders f on f.id = d.folders_id
	where f.users_id = ?`
	const usernamesUpdateSQL = `update usernames set v = ? where id = ?`
	if err := s.reencrypt(ctx, tx, oldCypher, newCypher, usernamesSQL, usernamesUpdateSQL, id); err != nil {
		return fmt.Errorf("unable to change passphrase: %w", err)
	}

	const passwordsSQL = `
	select p.id, p.v
	from passwords p
		join documents d on d.id = p.did
		join folders f on f.id = d.folders_id
	where f.users_id = ?`
	const passwordsUpdateSQL = `update passwords set v = ? where id = ?`
	if err := s.reencrypt(ctx, tx, oldCypher, newCypher, passwordsSQL, passwordsUpdateSQL, id); err != nil {
		return fmt.Errorf("unable to change passphrase: %w", err)
	}

	const keyvalsSQL = `
	select k.id, k.v
	from keyvals k
		join documents d on d.id = k.did
		join folders f on f.id = d.folders_id
	where f.users_id = ?`
	const keyvalsUpdateSQL = `update keyvals set v = ? where id = ?`
	if err := s.reencrypt(ctx, tx, oldCypher, newCypher, keyvalsSQL, keyvalsUpdateSQL, id); err != nil {
		return fmt.Errorf("unable to change passphrase: %w", err)
	}

	const notesSQL = `
	select n.id, n.v
	from notes n
		join documents d on d.id = n.did
		join folders f on f.id = d.folders_id
	where f.users_id = ?`
	const notesUpdateSQL = `update notes set v = ? where id = ?`
	if err := s.reencrypt(ctx, tx, oldCypher, newCypher, notesSQL, notesUpdateSQL, id); err != nil {
		return fmt.Errorf("unable to change passphrase: %w", err)
	}

	// update the authorization material; the passphraseupdates audit trigger
	// records this change automatically
	const authorizeUpdateSQL = `
	update authorize set
		salt = ?,
		verifier = ?
	where users_id = ?`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, authorizeUpdateSQL,
		newSalt.Unveil(),
		newVerifier.Unveil(),
		id,
	); err != nil {
		return fmt.Errorf("unable to change passphrase: %w", err)
	}

	return tx.Commit()
}

// loadAuthorization loads the salt and verifier for the given user.
func (s *usersql) loadAuthorization(ctx scope.C, tx *sql.Tx, id users.Identity) (authorization, error) {
	const statement = `
	select salt, verifier
	from authorize
	where users_id = ?`

	f := func(scan litesql.ScanFunc) (authorization, error) {
		var salt, verifier []byte
		if err := scan(&salt, &verifier); err != nil {
			return authorization{}, err
		}
		return authorization{
			salt:     conceal.NewBytes(salt),
			verifier: conceal.NewBytes(verifier),
		}, nil
	}

	auth, err := litesql.QueryRow(ctx, tx, f, statement, id)
	if err != nil {
		return authorization{}, fmt.Errorf("unable to load authorization: %w", err)
	}

	return auth, nil
}

// reencrypt decrypts every row returned by selectSQL with oldCypher,
// re-encrypts it with newCypher, and writes it back via updateSQL.
func (s *usersql) reencrypt(
	ctx scope.C,
	tx *sql.Tx,
	oldCypher, newCypher encrypt.Cypher,
	selectSQL, updateSQL string,
	id users.Identity,
) error {
	type row struct {
		id int64
		v  []byte
	}

	f := func(scan litesql.ScanFunc) (row, error) {
		var r row
		err := scan(&r.id, &r.v)
		return r, err
	}

	rows, qerr := litesql.QueryRows(ctx, tx, f, selectSQL, id)
	if qerr != nil {
		return fmt.Errorf("unable to load encrypted content: %w", qerr)
	}

	for _, r := range rows {
		plaintext, derr := oldCypher.Decrypt(conceal.NewBytes(r.v))
		if derr != nil {
			return fmt.Errorf("unable to decrypt content: %w", derr)
		}

		ciphertext, cerr := newCypher.Encrypt(plaintext)
		if cerr != nil {
			return fmt.Errorf("unable to encrypt content: %w", cerr)
		}

		if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, updateSQL, ciphertext.Unveil(), r.id); err != nil {
			return fmt.Errorf("unable to update content: %w", err)
		}
	}

	return nil
}
