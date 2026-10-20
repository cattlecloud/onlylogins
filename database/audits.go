package database

import (
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/lang"
)

type Audits interface {
	// metrics
	Count(scope.C) (int, int, error)

	// track login attempts
	Record(scope.C, *users.LoginRequest, bool) error
	RecordBogus(scope.C, *users.LoginRequest) error
}

type auditsql struct {
	log *ulog.Log
	ldb *litesql.LiteDB
}

func (s *store) Audit() Audits {
	return &auditsql{
		log: ulog.New("database/audits"),
		ldb: s.ldb,
	}
}

func (s *auditsql) Count(ctx scope.C) (int, int, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, 0, fmt.Errorf("unable to count audit records: %w", xerr)
	}
	defer done()

	var logins int
	if err := s.ldb.QueryRow(ctx, tx, "select count(id) from logins").Scan(&logins); err != nil {
		return 0, 0, fmt.Errorf("unable to count logins: %w", err)
	}

	var bogus int
	if err := s.ldb.QueryRow(ctx, tx, "select count(id) from bogus").Scan(&bogus); err != nil {
		return 0, 0, fmt.Errorf("unable to count bogus: %w", err)
	}

	return logins, bogus, nil
}

func (s *auditsql) Record(ctx scope.C, request *users.LoginRequest, success bool) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to record login audit: %w", xerr)
	}
	defer done()

	const statement = `
	insert into logins (
		users_id, ip, ua, success
	)
	select
		users.id, ?, ?, ?
	from users
	where
		users.username = ?`

	if _, err := s.ldb.ExecID(ctx, tx, statement,
		request.IP,
		request.UserAgent,
		lang.Maybe(success, 1, 0),
		request.Username,
	); err != nil {
		s.log.E.Fmt("unable to insert login audit: %v", err)
		return fmt.Errorf("unable to record login audit: %w", err)
	}

	return tx.Commit()
}

func (s *auditsql) RecordBogus(ctx scope.C, request *users.LoginRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to record login audit: %w", xerr)
	}
	defer done()

	const statement = `
	insert into bogus (
		ip, ua
	)
	values ( ?, ? )`

	if _, err := s.ldb.ExecID(ctx, tx, statement,
		request.IP,
		request.UserAgent,
	); err != nil {
		s.log.E.Fmt("unable to insert bogus login audit: %v", err)
		return fmt.Errorf("unable to record login audit: %w", err)
	}

	return tx.Commit()
}
