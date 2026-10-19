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
	Check(scope.C, *users.LoginRequest) error
	Intermediate(scope.C, string) (users.Identity, *conceal.Text, error)
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

func (s *usersql) Intermediate(ctx scope.C, username string) (users.Identity, *conceal.Text, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, nil, xerr
	}
	defer done()

	const statement = `
	select
		users.id,
		authorize.intermediate
	from authorize
		join users on users.id = authorize.users_id
	where
		users.username = ?`

	type result struct {
		id           users.Identity
		intermediate *conceal.Text
	}

	f := func(sf litesql.ScanFunc) (result, error) {
		var id users.Identity
		var intermediate string
		err := sf(&id, &intermediate)
		return result{
			id:           id,
			intermediate: conceal.New(intermediate),
		}, err
	}

	r, qerr := litesql.QueryRow(ctx, tx, f, statement, username)
	if qerr != nil {
		return 0, nil, qerr
	}

	return r.id, r.intermediate, nil
}

func (s *usersql) Create(ctx scope.C, request *users.CreateUserRequest, expect *conceal.Bytes) (users.Identity, error) {
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
		intermediate,
		expectation
	) values (
		?, ?, ?
	)`

	if _, serr := s.ldb.ExecID(ctx, tx, statement2,
		userID,
		request.Intermediate.Unveil(),
		expect.Unveil(),
	); serr != nil {
		return litesql.ExecFailure, fmt.Errorf("unable to create shibboleth: %w", serr)
	}

	return users.Identity(userID), tx.Commit()
}

func (s *usersql) Delete(ctx scope.C, user users.Identity) error {
	// TODO: implement me
	_ = ctx
	_ = user
	return nil
}

func (s *usersql) Check(ctx scope.C, request *users.LoginRequest) error {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const statement = `
	select
		authorize.intermediate,
		authorize.expectation
	from authorize
		join users on users.id = authorize.users_id
	where
		users.username = ?`

	type inputs struct {
		intermediate *conceal.Text
		expectation  *conceal.Bytes
	}

	f := func(sf litesql.ScanFunc) (inputs, error) {
		var intermediate string
		var expectation []byte
		err := sf(&intermediate, &expectation)
		return inputs{
			intermediate: conceal.New(intermediate),
			expectation:  conceal.NewBytes(expectation),
		}, err
	}

	credentials, cerr := litesql.QueryRow(ctx, tx, f, statement, request.Username)
	if cerr != nil {
		s.log.W.Fmt("unable to query user authorization: %v", cerr)
		return ErrIncorrectCredentials
	}

	// recompute the secret from the passphrase and intermediate, then decrypt
	// the stored expectation and verify it matches the litmus test
	secret := encrypt.Combine(request.Passphrase, credentials.intermediate)
	vault := encrypt.New(&encrypt.Inputs{
		Secret:       secret,
		Intermediate: credentials.intermediate,
	})

	plaintext, derr := vault.Decrypt(credentials.expectation)
	if derr != nil {
		s.log.W.Fmt("unable to decrypt litmus test: %v", derr)
		return ErrIncorrectCredentials
	}

	if !plaintext.Equal(encrypt.Litmus) {
		s.log.W.Fmt("passphrase comparison does not match for %s", request.Username)
		return ErrIncorrectCredentials
	}

	return nil
}
