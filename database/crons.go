// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"fmt"
	"time"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/crons"
)

type Crons interface {
	Register(scope.C, string) error
	Get(scope.C, string) (*crons.CurrentStatus, error)
	Latest(scope.C, string) (*crons.RunResult, error)
	List(scope.C) ([]*crons.RunResult, error)
	Start(scope.C, string) error
	Stop(scope.C, string, *crons.RunMetric) error
	Reset(scope.C, string) error
	Prune(scope.C, time.Duration) (int, error)
}

type cronsql struct {
	log *ulog.Log
	ldb *litesql.LiteDB
}

func (s *store) Crons() Crons {
	return &cronsql{
		log: ulog.New("database/crons"),
		ldb: s.ldb,
	}
}

func (s *cronsql) Register(ctx scope.C, name string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const statement = `
	insert or ignore into crons (
		label
	) values (
		?
	)`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOneOrZero, statement, name); err != nil {
		return fmt.Errorf("unable to register cronjob: %s: %w", name, err)
	}

	return tx.Commit()
}

func (s *cronsql) List(ctx scope.C) ([]*crons.RunResult, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		crons.id,
		crons.label,
		cron_states.label,
		cron_runs.code,
		cron_runs.started,
		cron_runs.stopped
	from crons
		join cron_states on crons.states_id = cron_states.id
		join cron_runs on cron_runs.id = (
			select max(id) from cron_runs where crons_id = crons.id
		)
	order by crons.id asc`

	return litesql.QueryRows(ctx, tx, scanCronRunResult, statement)
}

func (s *cronsql) Latest(ctx scope.C, name string) (*crons.RunResult, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		crons.id,
		crons.label,
		cron_states.label,
		cron_runs.code,
		cron_runs.started,
		cron_runs.stopped
	from crons
		join cron_states on crons.states_id = cron_states.id
		join cron_runs on cron_runs.id = (
			select max(id) from cron_runs where crons_id = crons.id
		)
	where crons.label = ?`

	return litesql.QueryRow(ctx, tx, scanCronRunResult, statement, name)
}

func scanCronRunResult(scan litesql.ScanFunc) (*crons.RunResult, error) {
	r := new(crons.RunResult)
	var start, stop int64

	if err := scan(
		&r.ID,
		&r.Label,
		&r.State,
		&r.Code,
		&start,
		&stop,
	); err != nil {
		return nil, err
	}

	r.Start = time.Unix(start, 0)
	r.Stop = time.Unix(stop, 0)

	return r, nil
}

func scanCronCurrentStatus(scan litesql.ScanFunc) (*crons.CurrentStatus, error) {
	c := new(crons.CurrentStatus)

	if err := scan(
		&c.ID,
		&c.Label,
		&c.State,
	); err != nil {
		return nil, err
	}

	return c, nil
}

func (s *cronsql) Get(ctx scope.C, name string) (*crons.CurrentStatus, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		crons.id,
		crons.label,
		cron_states.label
	from crons
		join cron_states on crons.states_id = cron_states.id
	where
		crons.label = ?`

	return litesql.QueryRow(ctx, tx, scanCronCurrentStatus, statement, name)
}

func (s *cronsql) Start(ctx scope.C, name string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const statement = `
	update crons set
		states_id = 2,
		modification = unixepoch('now')
	where
		label = ?`

	if _, err := s.ldb.ExecID(ctx, tx, statement, name); err != nil {
		return fmt.Errorf("unable to start cron: %s: %w", name, err)
	}

	return tx.Commit()
}

func (s *cronsql) Stop(ctx scope.C, name string, metric *crons.RunMetric) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const update = `
	update crons set
		states_id = 3,
		modification = unixepoch('now')
	where
		label = ?`

	if _, err := s.ldb.ExecID(ctx, tx, update, name); err != nil {
		return fmt.Errorf("unable to stop cron: %s: %w", name, err)
	}

	const insert = `
	insert into cron_runs (
		crons_id,
		code,
		started,
		stopped
	) values (
		(select id from crons where label = ?), ?, ?, ?
	)`

	if _, err := s.ldb.ExecID(
		ctx, tx, insert,
		name, metric.Code, metric.Start.Unix(), metric.Stop.Unix(),
	); err != nil {
		return fmt.Errorf("unable to record cron run: %s: %w", name, err)
	}

	return tx.Commit()
}

func (s *cronsql) Reset(ctx scope.C, name string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const statement = `
	update crons set
		states_id = 3,
		modification = unixepoch('now')
	where
		label = ?`

	if _, err := s.ldb.ExecID(ctx, tx, statement, name); err != nil {
		return fmt.Errorf("unable to reset cron: %s: %w", name, err)
	}

	return tx.Commit()
}

func (s *cronsql) Prune(ctx scope.C, maxAge time.Duration) (int, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return litesql.TxFailure, xerr
	}
	defer done()

	cutoff := time.Now().Add(-maxAge).Unix()

	const countSQL = `select count(id) from cron_runs where started < ?`
	count, qerr := litesql.QueryRow(ctx, tx, scanCount, countSQL, cutoff)
	if qerr != nil {
		return litesql.ExecFailure, fmt.Errorf("unable to count cron runs to prune: %w", qerr)
	}

	const deleteSQL = `delete from cron_runs where started < ?`
	if err := s.ldb.Exec(ctx, tx, count, deleteSQL, cutoff); err != nil {
		return litesql.ExecFailure, fmt.Errorf("unable to prune cron runs: %w", err)
	}

	return count, tx.Commit()
}
