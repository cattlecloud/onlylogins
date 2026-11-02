// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import (
	"database/sql"
	"errors"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/pkg/users"
)

// ErrQuotaExceeded indicates user may not create more of something.
var ErrQuotaExceeded = errors.New("account quota exceeded")

// ErrAccountStatus indicates user is not in good standing and may not create.
var ErrAccountStatus = errors.New("account not in good standing")

func enforcements(ldb *litesql.LiteDB) *enforcement {
	return &enforcement{
		ldb: ldb,
	}
}

type enforcement struct {
	ldb *litesql.LiteDB
}

type quota struct {
	standing users.Standing
	limit    int
	count    int
}

func (q *quota) assert() error {
	switch {
	case q.standing == users.Banned:
		return ErrAccountStatus
	case q.count >= q.limit:
		return ErrQuotaExceeded
	default:
		return nil
	}
}

func (e *enforcement) quotas(ctx scope.C, tx *sql.Tx, user users.Identity) error {
	statement := `
	select
		standings.id,
		plans.docs_limit,
		(select count(d.id)
		 from documents d
		 join folders f on f.id = d.folders_id
		 where f.users_id = users.id)
	from users
		join standings on users.standings_id = standings.id
		join plans on users.plans_id = plans.id
	where
		users.id = ?`

	f := func(scan litesql.ScanFunc) (*quota, error) {
		q := new(quota)
		err := scan(
			&q.standing,
			&q.limit,
			&q.count,
		)
		return q, err
	}

	q, err := litesql.QueryRow(ctx, tx, f, statement, user)
	if err != nil {
		return err
	}

	return q.assert()
}
