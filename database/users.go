package database

import (
	"errors"
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

var (
	ErrIncorrectCredentials = errors.New("incorrect credentials")
)

type Users interface {
	// metrics
	Count(scope.C) (int, error)

	// user management
	Create(scope.C, *users.CreateUserRequest, *conceal.Bytes) (users.Identity, error)
	Delete(scope.C, users.Identity) error
	Lookup(scope.C, users.Identity) (*users.Profile, error)
	Check(scope.C, *users.LoginRequest) (users.Identity, *conceal.Bytes, error)
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

	f := func(sf litesql.ScanFunc) (int, error) {
		var i int
		err := sf(&i)
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
		standings_id
	from users
	where id = ?`

	f := func(sf litesql.ScanFunc) (*users.Profile, error) {
		var profile users.Profile
		err := sf(&profile.ID, &profile.Username, &profile.Standing)
		return &profile, err
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
		username
	) values ( ? )`

	userID, uerr := s.ldb.ExecID(ctx, tx, statement,
		request.Username,
	)
	if uerr != nil {
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

	// remove the authorization row first (foreign key)
	const statement = `delete from authorize where users_id = ?`
	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement, user); err != nil {
		return fmt.Errorf("unable to delete authorization: %w", err)
	}

	const statement2 = `delete from users where id = ?`
	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement2, user); err != nil {
		return fmt.Errorf("unable to delete user: %w", err)
	}

	return tx.Commit()
}

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

	f := func(sf litesql.ScanFunc) (inputs, error) {
		var id users.Identity
		var salt []byte
		var verifier []byte
		err := sf(&id, &salt, &verifier)
		return inputs{
			id:       id,
			salt:     conceal.NewBytes(salt),
			verifier: conceal.NewBytes(verifier),
		}, err
	}

	credentials, cerr := litesql.QueryRow(ctx, tx, f, statement, request.Username)
	if cerr != nil {
		s.log.W.Fmt("unable to query user authorization: %v", cerr)
		// substitute dummy credentials so the key derivation still runs,
		// keeping the response time independent of whether the username exists
		credentials = inputs{
			salt:     encrypt.DummySalt,
			verifier: encrypt.DummyVerifier,
		}
	}

	// derive the key from the passphrase and salt, then verify it matches the
	// stored key check value
	key := encrypt.DeriveKey(request.Passphrase, credentials.salt)
	if !encrypt.Verify(key, credentials.verifier) {
		return 0, nil, ErrIncorrectCredentials
	}

	return credentials.id, key, nil
}
