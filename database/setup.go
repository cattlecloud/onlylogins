package database

import (
	// anonymous import to embed sql file
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
	if err := s.ldb.Exec(ctx, tx, litesql.ExpectAnything, initUsers); err != nil {
		return fmt.Errorf("unable to initialize built-in users: %w", err)
	}

	return tx.Commit()
}
