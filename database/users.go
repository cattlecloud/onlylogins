package database

import (
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

type Users interface {
	// metrics
	Count(scope.C) (int, error)

	// user management
	Create(scope.C, *users.CreateUserRequest, *conceal.Bytes) (users.Identity, error)
	Delete(scope.C, users.Identity) error
	Lookup(scope.C, users.Identity) (*users.Profile, error)
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
