// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type Plans interface {
	Get(scope.C, records.PlansEnum) (*sitecontent.Plan, error)
	List(scope.C) ([]*sitecontent.Plan, error)
	Usage(scope.C, users.Identity) (*sitecontent.Usage, error)
}

type plansql struct {
	log *ulog.Log
	ldb *litesql.LiteDB
}

func (s *store) Plans() Plans {
	return &plansql{
		log: ulog.New("database/plans"),
		ldb: s.ldb,
	}
}

func scanPlan(scan litesql.ScanFunc) (*sitecontent.Plan, error) {
	p := new(sitecontent.Plan)
	err := scan(
		&p.ID,
		&p.Label,
		&p.MonthlyPrice,
		&p.DocumentsLimit,
	)
	return p, err
}

func (s *plansql) Get(ctx scope.C, plan records.PlansEnum) (*sitecontent.Plan, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to lookup plan: %w", xerr)
	}
	defer done()

	const statement = `
	select
		plans.id,
		plans.label,
		plans.price,
		plans.docs_limit
	from plans
	where
		plans.id = ?`

	return litesql.QueryRow(ctx, tx, scanPlan, statement, plan)
}

func (s *plansql) List(ctx scope.C) ([]*sitecontent.Plan, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to list plans: %w", xerr)
	}
	defer done()

	const statement = `
	select
		plans.id,
		plans.label,
		plans.price,
		plans.docs_limit
	from plans
	order by plans.id asc`

	return litesql.QueryRows(ctx, tx, scanPlan, statement)
}

func (s *plansql) Usage(ctx scope.C, user users.Identity) (*sitecontent.Usage, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to get usage: %w", xerr)
	}
	defer done()

	const statement = `
	select
		plans.id,
		plans.label,
		plans.price,
		plans.docs_limit,
		(select count(documents.id) from documents join folders on documents.folders_id = folders.id where folders.users_id = users.id)
	from users
		join plans on users.plans_id = plans.id
	where
		users.id = ?`

	f := func(scan litesql.ScanFunc) (*sitecontent.Usage, error) {
		u := new(sitecontent.Usage)
		u.User = user
		err := scan(
			&u.Plan.ID,
			&u.Plan.Label,
			&u.Plan.MonthlyPrice,
			&u.Plan.DocumentsLimit,
			&u.Documents,
		)
		return u, err
	}

	return litesql.QueryRow(ctx, tx, f, statement, user)
}
