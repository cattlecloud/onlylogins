// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import (
	// anonymous import to embed sql file
	"database/sql"
	_ "embed"
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
)

//go:embed sql/init-db.sql
var initSQL string

//go:embed sql/init-users.sql
var initUsers string

// Init will execute the sql/init-*.sql file(s).
func (s *store) Init(ctx scope.C) error {
	tx, txclose, txerr := s.ldb.StartWrite(ctx)
	if txerr != nil {
		return txerr
	}
	defer txclose()

	// first initialize the database schema; including migrations
	if err := s.ldb.Exec(ctx, tx, litesql.ExpectAnything, initSQL); err != nil {
		return err
	}

	// next initialize the built-in users
	if err := s.initUsers(ctx, tx); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *store) initUsers(ctx scope.C, tx *sql.Tx) error {
	const statement = `select count(id) from users`
	count, cerr := litesql.QueryRow(ctx, tx, scanCount, statement)
	if cerr != nil {
		return fmt.Errorf("unable to count users: %w", cerr)
	}

	if count == 0 {
		if err := s.ldb.Exec(ctx, tx, litesql.ExpectAnything, initUsers); err != nil {
			return fmt.Errorf("unable to initialize users: %w", err)
		}
	}
	return nil
}
