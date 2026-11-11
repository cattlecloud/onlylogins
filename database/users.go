// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

// Sentinels related to user account management, surfaced to the login
// handler so it can audit username-probing attempts (bogus table) vs.
// password-guessing attempts (logins table) and so it can present a
// uniform "incorrect credentials" message regardless of which kind of
// failure occurred (to prevent username enumeration).
var (
	ErrIncorrectCredentials = errors.New("incorrect credentials")
	ErrUsernameTaken        = errors.New("username is already taken")
	ErrUserNotFound         = errors.New("user not found")
)

type Users interface {
	// metrics
	Count(scope.C) (int, error)

	// user management
	Create(scope.C, *users.CreateUserRequest, *conceal.Bytes) (users.Identity, error)
	Delete(scope.C, users.Identity) error
	Lookup(scope.C, users.Identity) (*users.Profile, error)
	Profile(scope.C, users.Identity) (*sitecontent.Profile, error)
	Moderate(scope.C, users.Identity) (*sitecontent.UserModeration, error)
	Check(scope.C, *users.LoginRequest) (users.Identity, *conceal.Bytes, error)
	List(scope.C) ([]*users.Profile, error)
}

type usersql struct {
	log *ulog.Log
	ldb *litesql.LiteDB
}

func (s *store) Users() Users {
	return &usersql{
		log: ulog.New("database/users"),
		ldb: s.ldb,
	}
}

func (s *usersql) Count(ctx scope.C) (int, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, xerr
	}
	defer done()

	const statement = `select count(id) from users`

	f := func(scan litesql.ScanFunc) (int, error) {
		var i int
		err := scan(&i)
		return i, err
	}

	return litesql.QueryRow(ctx, tx, f, statement)
}

func (s *usersql) Lookup(ctx scope.C, id users.Identity) (*users.Profile, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		id,
		username,
		email,
		standings_id
	from users
	where id = ?`

	f := func(scan litesql.ScanFunc) (*users.Profile, error) {
		var profile users.Profile
		err := scan(
			&profile.ID,
			&profile.Username,
			&profile.Email,
			&profile.Standing,
		)
		return &profile, err
	}

	return litesql.QueryRow(ctx, tx, f, statement, id)
}

func (s *usersql) Profile(ctx scope.C, id users.Identity) (*sitecontent.Profile, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		users.username,
		plans.id,
		plans.label,
		plans.price,
		plans.docs_limit,
		(select count(d.id)
		 from documents d
		 join folders f on f.id = d.folders_id
		 where f.users_id = users.id)
	from users
		join plans on users.plans_id = plans.id
	where
		users.id = ?`

	f := func(scan litesql.ScanFunc) (*sitecontent.Profile, error) {
		profile := new(sitecontent.Profile)
		if err := scan(
			&profile.Username,
			&profile.Usage.Plan.ID,
			&profile.Usage.Plan.Label,
			&profile.Usage.Plan.MonthlyPrice,
			&profile.Usage.Plan.DocumentsLimit,
			&profile.Usage.Documents,
		); err != nil {
			return nil, err
		}
		profile.Usage.User = id
		return profile, nil
	}

	return litesql.QueryRow(ctx, tx, f, statement, id)
}

func (s *usersql) Moderate(ctx scope.C, id users.Identity) (*sitecontent.UserModeration, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		users.id,
		users.username,
		users.email,
		users.standings_id,
		plans.id,
		plans.label,
		plans.price,
		plans.docs_limit,
		(select count(d.id)
		 from documents d
		 join folders f on f.id = d.folders_id
		 where f.users_id = users.id)
	from users
		join plans on users.plans_id = plans.id
	where
		users.id = ?`

	f := func(scan litesql.ScanFunc) (*sitecontent.UserModeration, error) {
		um := new(sitecontent.UserModeration)
		um.Profile = new(users.Profile)
		if err := scan(
			&um.Profile.ID,
			&um.Profile.Username,
			&um.Profile.Email,
			&um.Profile.Standing,
			&um.Plan.ID,
			&um.Plan.Label,
			&um.Plan.MonthlyPrice,
			&um.Plan.DocumentsLimit,
			&um.Usage.Documents,
		); err != nil {
			return nil, err
		}
		um.Usage.User = um.Profile.ID
		um.Usage.Plan = um.Plan
		return um, nil
	}

	return litesql.QueryRow(ctx, tx, f, statement, id)
}

func (s *usersql) Create(ctx scope.C, request *users.CreateUserRequest, verifier *conceal.Bytes) (users.Identity, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return 0, fmt.Errorf("unable to create user: %w", xerr)
	}
	defer done()

	const statement = `
	insert into users (
		username,
		email
	) values ( ?, ? )`

	userID, uerr := s.ldb.ExecID(ctx, tx, statement,
		request.Username,
		request.Email,
	)
	if isUsernameTaken(uerr) {
		return litesql.ExecFailure, fmt.Errorf("unable to create user: %w", ErrUsernameTaken)
	} else if uerr != nil {
		return litesql.ExecFailure, fmt.Errorf("unable to create user: %w", uerr)
	}

	const statement2 = `
	insert into authorize (
		users_id,
		salt,
		verifier
	) values (
		?, ?, ?
	)`

	if _, serr := s.ldb.ExecID(ctx, tx, statement2,
		userID,
		request.Salt.Unveil(),
		verifier.Unveil(),
	); serr != nil {
		return litesql.ExecFailure, fmt.Errorf("unable to create verifier: %w", serr)
	}

	return users.Identity(userID), tx.Commit()
}

func (s *usersql) Delete(ctx scope.C, user users.Identity) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	// the authorize row is removed automatically via on delete cascade
	const statement = `delete from users where id = ?`
	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement, user); err != nil {
		return fmt.Errorf("unable to delete user: %w", err)
	}

	return tx.Commit()
}

// Check verifies credentials.
//
// To prevent username enumeration via response timing, when the username
// does not match a row the method still derives a key against a fixed
// dummy verifier, so the overall operation takes the same amount of time
// regardless of whether the username exists.
//
// On failure the returned error is one of:
//   - ErrUserNotFound            ("user not found")             - username did not match
//   - ErrIncorrectCredentials    ("incorrect credentials")      - wrong passphrase on a matched user
//   - other database error (wrapped)                            - infrastructure problem
//
// Callers can route ErrUserNotFound to a separate audit table (bogus)
// to track username-probing attempts distinct from password-guessing
// attempts, while still presenting a uniform "incorrect credentials"
// message to the user.
func (s *usersql) Check(ctx scope.C, request *users.LoginRequest) (users.Identity, *conceal.Bytes, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, nil, xerr
	}
	defer done()

	const statement = `
	select
		users.id,
		authorize.salt,
		authorize.verifier
	from authorize
		join users on users.id = authorize.users_id
	where
		users.username = ?`

	type inputs struct {
		id       users.Identity
		salt     *conceal.Bytes
		verifier *conceal.Bytes
	}

	f := func(scan litesql.ScanFunc) (inputs, error) {
		var id users.Identity
		var salt []byte
		var verifier []byte
		err := scan(&id, &salt, &verifier)
		return inputs{
			id:       id,
			salt:     conceal.NewBytes(salt),
			verifier: conceal.NewBytes(verifier),
		}, err
	}

	credentials, cerr := litesql.QueryRow(ctx, tx, f, statement, request.Username)
	if cerr != nil {
		s.log.W.Fmt("unable to query user authorization: %v", cerr)
		// substitute dummy credentials so the key derivation below still
		// runs, keeping the response time independent of whether the
		// username exists (timing-attack mitigation)
		credentials = inputs{
			salt:     encrypt.DummySalt,
			verifier: encrypt.DummyVerifier,
		}
	}

	// derive the key from the passphrase and salt, then verify it against
	// the stored verifier (or the dummy verifier when the username didn't
	// match, preserving timing). the verify result combined with cerr tells
	// us which kind of failure occurred, so callers can route audits
	// appropriately (e.g. bogus table for missing usernames).
	key := encrypt.DeriveKey(request.Passphrase, credentials.salt)
	if !encrypt.Verify(key, credentials.verifier) {
		switch {
		case errors.Is(cerr, sql.ErrNoRows):
			return 0, nil, ErrUserNotFound
		case cerr != nil:
			return 0, nil, fmt.Errorf("unable to check credentials: %w", cerr)
		default:
			return 0, nil, ErrIncorrectCredentials
		}
	}

	return credentials.id, key, nil
}

// isUsernameTaken reports whether err indicates that the users.username
// UNIQUE constraint failed during an insert.
func isUsernameTaken(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: users.username")
}

func (s *usersql) List(ctx scope.C) ([]*users.Profile, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to list users: %w", xerr)
	}
	defer done()

	const statement = `
	select
		id,
		username,
		email,
		standings_id
	from users
	order by id asc`

	f := func(scan litesql.ScanFunc) (*users.Profile, error) {
		var profile users.Profile
		err := scan(
			&profile.ID,
			&profile.Username,
			&profile.Email,
			&profile.Standing,
		)
		return &profile, err
	}

	return litesql.QueryRows(ctx, tx, f, statement)
}
