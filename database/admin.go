// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/hashicorp/go-set/v3"
)

type Admin interface {
	// manage admins
	Sync(scope.C, set.Collection[users.Identity], set.Collection[users.Identity]) error

	// manage user accounts
	ListStanding(scope.C, users.Standing) (set.Collection[users.Identity], error)
}

type adminsql struct {
	log *ulog.Log
	ldb *litesql.LiteDB
}

func (s *store) Admin() Admin {
	return &adminsql{
		log: ulog.New("database/admin"),
		ldb: s.ldb,
	}
}

func (s *adminsql) Sync(ctx scope.C, add set.Collection[users.Identity], remove set.Collection[users.Identity]) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const adminSQL = `
	update users set
		standings_id = ?,
		modification = unixepoch('now')
	where
		id = ?`

	for id := range add.Items() {
		if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, adminSQL, users.Admin, id); err != nil {
			return fmt.Errorf("unable to promote user to admin: %w", err)
		}
	}

	const normalSQL = `
	update users set
		standings_id = ?,
		modification = unixepoch('now')
	where
		id = ?`

	for id := range remove.Items() {
		if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, normalSQL, users.Normal, id); err != nil {
			return fmt.Errorf("unable to demote user to normal: %w", err)
		}
	}

	return tx.Commit()
}

func (s *adminsql) ListStanding(ctx scope.C, standing users.Standing) (set.Collection[users.Identity], error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `select id from users where standings_id = ?`

	scanID := func(scan litesql.ScanFunc) (users.Identity, error) {
		var id users.Identity
		err := scan(&id)
		return id, err
	}

	ids, qerr := litesql.QueryRows(ctx, tx, scanID, statement, standing)
	if qerr != nil {
		return nil, fmt.Errorf("unable to list standing users: %w", qerr)
	}

	return set.From(ids), nil
}
