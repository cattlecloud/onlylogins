package database

import (
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/users"
)

type Users interface {
	// metrics
	Count(scope.C) (int, error)

	// user management
	Create(scope.C, *users.CreateUserRequest) (users.Identity, error)
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

	count, err := litesql.QueryRow(ctx, tx, f, statement)
	return count, err
}

func (s *usersql) Create(ctx scope.C, request *users.CreateUserRequest) (users.Identity, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return 0, fmt.Errorf("unable to create user: %w", xerr)
	}
	defer done()

	const statement = `
	insert into users (
		username
	) values ( ? )`

	id, uerr := s.ldb.ExecID(ctx, tx, statement,
		request.Username,
	)
	if uerr != nil {
		return litesql.ExecFailure, fmt.Errorf("unable to create user: %w", uerr)
	}

	const statement2 = `
	insert into authorize (
		users_id,
		shibboleth,
		expectation
	) values (
		?, ?, ?
	)`

	xx := request.Shibboleth.Unveil()
	fmt.Println("xx is", xx)

	_, serr := s.ldb.ExecID(ctx, tx, statement2,
		id, request.Shibboleth.Unveil(), []byte("abc"),
	)
	if serr != nil {
		return litesql.ExecFailure, fmt.Errorf("unable to create shibboleth: %w", serr)
	}

	return users.Identity(id), tx.Commit()
}
