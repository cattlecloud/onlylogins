// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"fmt"
	"time"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/lang"
)

type Stripe interface {
	// customers
	GetCustomer(scope.C, users.Identity) (string, bool, error)
	GetUserByCustomer(scope.C, string) (users.Identity, error)
	SetCustomer(scope.C, users.Identity, string) error

	// subscriptions
	GetSubscription(scope.C, users.Identity) (*payments.StripeSubscription, error)
	GetUserBySubscription(scope.C, string) (users.Identity, error)
	UpsertSubscription(scope.C, *payments.UpdateStripeSubscription) error
	DowngradeExpired(scope.C) (int, error)
	CurrentPlan(scope.C, users.Identity) (records.PlansEnum, error)

	// webhook event inbox
	RecordEvent(scope.C, *payments.StripeEvent) (bool, error)
	GetEvent(scope.C, string) (*payments.StripeEvent, error)
	MarkEventDone(scope.C, string) error
	MarkEventFailed(scope.C, string, string) error
	MarkEventDead(scope.C, string, string) error
	RetryableEvents(scope.C, time.Time, int) ([]*payments.StripeEvent, error)
	ExpireEvents(scope.C, time.Time) (int, error)
	ProblemEvents(scope.C, time.Time, int) ([]*payments.StripeEvent, error)
	PruneEvents(scope.C, time.Duration) (int, error)
}

func (s *store) Stripe() Stripe {
	return &stripesql{ldb: s.ldb}
}

type stripesql struct {
	ldb *litesql.LiteDB
}

func (s *stripesql) GetCustomer(ctx scope.C, uid users.Identity) (string, bool, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return "", false, xerr
	}
	defer done()

	const statement = `
	select customer_id from stripe_customers
	where
		users_id = ?`

	f := func(scan litesql.ScanFunc) (string, error) {
		var customer string
		return customer, scan(&customer)
	}

	customer, err := litesql.QueryRow(ctx, tx, f, statement, uid)
	if err != nil {
		return "", false, nil
	}

	return customer, true, nil
}

func (s *stripesql) GetUserByCustomer(ctx scope.C, customerID string) (users.Identity, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, xerr
	}
	defer done()

	const statement = `
	select users_id from stripe_customers
	where
		customer_id = ?`

	f := func(scan litesql.ScanFunc) (users.Identity, error) {
		var uid users.Identity
		return uid, scan(&uid)
	}

	return litesql.QueryRow(ctx, tx, f, statement, customerID)
}

func (s *stripesql) SetCustomer(ctx scope.C, uid users.Identity, customerID string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const statement = `
	insert into stripe_customers (
		users_id, customer_id
	) values (?, ?)
	on conflict(users_id) do update set
		customer_id = excluded.customer_id`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement, uid, customerID); err != nil {
		return fmt.Errorf("unable to set stripe customer: %w", err)
	}

	return tx.Commit()
}

func scanStripeSubscription(scan litesql.ScanFunc) (*payments.StripeSubscription, error) {
	s := new(payments.StripeSubscription)
	err := scan(
		&s.SubscriptionID,
		&s.PriceID,
		&s.Status,
		&s.CurrentPeriodEnd,
	)
	return s, err
}

func (s *stripesql) GetSubscription(ctx scope.C, uid users.Identity) (*payments.StripeSubscription, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		subscription_id,
		price_id,
		status,
		current_period_end
	from 
		stripe_subscriptions
	where
		users_id = ?`

	return litesql.QueryRow(ctx, tx, scanStripeSubscription, statement, uid)
}

func (s *stripesql) UpsertSubscription(ctx scope.C, update *payments.UpdateStripeSubscription) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const subscriptionSQL = `
	insert into stripe_subscriptions
		(users_id, subscription_id, price_id, status, current_period_end)
	values (?, ?, ?, ?, ?)
	on conflict(users_id) do update set
		subscription_id = excluded.subscription_id,
		price_id = excluded.price_id,
		status = excluded.status,
		current_period_end = excluded.current_period_end`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, subscriptionSQL,
		update.User,
		update.Sub.SubscriptionID,
		update.Sub.PriceID,
		update.Sub.Status,
		update.Sub.CurrentPeriodEnd,
	); err != nil {
		return fmt.Errorf("stripe subscriptions upsert: %w", err)
	}

	const planSQL = `
	update users set
		plans_id = ?,
		modification = unixepoch('now')
	where
		id = ?`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, planSQL,
		update.Plan,
		update.User,
	); err != nil {
		return fmt.Errorf("strip update plan: %w", err)
	}

	return tx.Commit()
}

func (s *stripesql) GetUserBySubscription(ctx scope.C, subscriptionID string) (users.Identity, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, xerr
	}
	defer done()

	const statement = `
	select users_id from stripe_subscriptions
	where
		subscription_id = ?`

	f := func(scan litesql.ScanFunc) (users.Identity, error) {
		var uid users.Identity
		return uid, scan(&uid)
	}

	return litesql.QueryRow(ctx, tx, f, statement, subscriptionID)
}

func (s *stripesql) DowngradeExpired(ctx scope.C) (int, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return 0, xerr
	}
	defer done()

	const countSQL = `
	select count(users_id) from stripe_subscriptions
	where
		status in ('canceled', 'unpaid')
		and current_period_end < unixepoch('now')`

	scanner := func(scan litesql.ScanFunc) (int, error) {
		var n int
		if err := scan(&n); err != nil {
			return 0, err
		}
		return n, nil
	}

	count, qerr := litesql.QueryRow(ctx, tx, scanner, countSQL)
	if qerr != nil {
		return 0, fmt.Errorf("stripe count expired: %w", qerr)
	}

	const updateSQL = `
	update users set plans_id = ?, modification = unixepoch('now')
	where id in (
		select ss.users_id
		from stripe_subscriptions ss
		where ss.status in ('canceled', 'unpaid')
		and ss.current_period_end < unixepoch('now')
	)`

	if err := s.ldb.Exec(ctx, tx, count, updateSQL, records.PlansFree); err != nil {
		return 0, fmt.Errorf("stripe downgrade expired: %w", err)
	}

	return count, tx.Commit()
}

func scanStripeEvent(scan litesql.ScanFunc) (*payments.StripeEvent, error) {
	event := new(payments.StripeEvent)
	var livemode int
	var received int64
	err := scan(
		&event.ID,
		&event.Type,
		&livemode,
		&event.Created,
		&received,
		&event.Payload,
		&event.Status,
		&event.Attempts,
		&event.LastError,
	)
	event.Livemode = livemode == 1
	event.Received = time.Unix(received, 0)
	return event, err
}

// RecordEvent persists a verified webhook event. It returns true when the event
// was newly recorded, and false when the event was already in the inbox.
func (s *stripesql) RecordEvent(ctx scope.C, event *payments.StripeEvent) (bool, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return false, xerr
	}
	defer done()

	const statement = `
	insert into stripe_events (
		event_id, etype, livemode, created, received, payload
	) values (?, ?, ?, ?, ?, ?)
	on conflict(event_id) do nothing`

	livemode := lang.Maybe(event.Livemode, 1, 0)

	// force received time to now if not set
	received := event.Received
	if received.IsZero() {
		received = time.Now()
	}

	// execute directly so we can observe whether the insert actually happened
	result, ierr := tx.ExecContext(ctx, statement,
		event.ID,
		event.Type,
		livemode,
		event.Created,
		received.Unix(),
		event.Payload,
	)
	if ierr != nil {
		return false, fmt.Errorf("unable to record stripe event: %w", ierr)
	}

	affected, aerr := result.RowsAffected()
	if aerr != nil {
		return false, fmt.Errorf("unable to lookup stripe event insertion rows affected: %w", aerr)
	}

	return affected == 1, tx.Commit()
}

func (s *stripesql) GetEvent(ctx scope.C, eventID string) (*payments.StripeEvent, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		event_id,
		etype,
		livemode,
		created,
		received,
		payload,
		status,
		attempts,
		last_error
	from
		stripe_events
	where
		event_id = ?`

	return litesql.QueryRow(ctx, tx, scanStripeEvent, statement, eventID)
}

func (s *stripesql) MarkEventDone(ctx scope.C, eventID string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const statement = `
	update stripe_events set
		status = ?,
		attempts = attempts + 1,
		last_error = '',
		updated = unixepoch('now')
	where
		event_id = ?`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		payments.EventStatusDone,
		eventID,
	); err != nil {
		return fmt.Errorf("stripe events mark done: %w", err)
	}

	return tx.Commit()
}

func (s *stripesql) MarkEventFailed(ctx scope.C, eventID, reason string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const statement = `
	update stripe_events set
		status = ?,
		attempts = attempts + 1,
		last_error = ?,
		updated = unixepoch('now')
	where
		event_id = ?
		and status != ?`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOneOrZero, statement,
		payments.EventStatusFailed,
		reason,
		eventID,
		payments.EventStatusDone,
	); err != nil {
		return fmt.Errorf("stripe events mark failed: %w", err)
	}

	return tx.Commit()
}

// MarkEventDead marks an event as permanently unprocessable. Dead events are
// not retried, and are surfaced for review.
func (s *stripesql) MarkEventDead(ctx scope.C, eventID, reason string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const statement = `
	update stripe_events set
		status = ?,
		last_error = ?,
		updated = unixepoch('now')
	where
		event_id = ?
		and status != ?`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOneOrZero, statement,
		payments.EventStatusDead,
		reason,
		eventID,
		payments.EventStatusDone,
	); err != nil {
		return fmt.Errorf("stripe events mark dead: %w", err)
	}

	return tx.Commit()
}

// RetryableEvents returns the events which may still be processed; those
// which have not been applied, and which are not yet older than the given
// cutoff.
//
// Events are ordered by their last update, so the least recently attempted
// event is handled first.
func (s *stripesql) RetryableEvents(ctx scope.C, cutoff time.Time, limit int) ([]*payments.StripeEvent, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		event_id,
		etype,
		livemode,
		created,
		received,
		payload,
		status,
		attempts,
		last_error
	from
		stripe_events
	where
		status in (?, ?)
		and received >= ?
	order by
		updated asc
	limit ?`

	return litesql.QueryRows(ctx, tx, scanStripeEvent, statement,
		payments.EventStatusPending,
		payments.EventStatusFailed,
		cutoff.Unix(),
		limit,
	)
}

// ExpireEvents marks events which are older than the given cutoff as dead;
// they have been given the full opportunity to be applied.
func (s *stripesql) ExpireEvents(ctx scope.C, cutoff time.Time) (int, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return 0, xerr
	}
	defer done()

	const countSQL = `
	select count(event_id) from stripe_events
	where
		status in (?, ?)
		and received < ?`

	count, qerr := litesql.QueryRow(ctx, tx, scanCount, countSQL,
		payments.EventStatusPending,
		payments.EventStatusFailed,
		cutoff.Unix(),
	)
	if qerr != nil {
		return 0, fmt.Errorf("stripe count expired events: %w", qerr)
	}

	const updateSQL = `
	update stripe_events set
		status = ?,
		last_error = case
			when last_error = '' then ?
			else last_error
		end,
		updated = unixepoch('now')
	where
		status in (?, ?)
		and received < ?`

	if err := s.ldb.Exec(ctx, tx, count, updateSQL,
		payments.EventStatusDead,
		"retry window exceeded",
		payments.EventStatusPending,
		payments.EventStatusFailed,
		cutoff.Unix(),
	); err != nil {
		return 0, fmt.Errorf("stripe expire events: %w", err)
	}

	return count, tx.Commit()
}

// ProblemEvents returns the events which need attention; those which are dead,
// and those which have been failing for longer than the given cutoff.
func (s *stripesql) ProblemEvents(ctx scope.C, cutoff time.Time, limit int) ([]*payments.StripeEvent, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		event_id,
		etype,
		livemode,
		created,
		received,
		payload,
		status,
		attempts,
		last_error
	from
		stripe_events
	where
		status = ?
		or (status in (?, ?) and received < ?)
	order by
		received asc
	limit ?`

	return litesql.QueryRows(ctx, tx, scanStripeEvent, statement,
		payments.EventStatusDead,
		payments.EventStatusPending,
		payments.EventStatusFailed,
		cutoff.Unix(),
		limit,
	)
}

// PruneEvents removes the events which have been applied, and which are older
// than the given age. The events which have not been applied are kept, since
// they may still be retried.
func (s *stripesql) PruneEvents(ctx scope.C, maxAge time.Duration) (int, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return 0, xerr
	}
	defer done()

	cutoff := time.Now().Add(-maxAge).Unix()

	const countSQL = `
	select count(event_id) from stripe_events
	where
		status = ?
		and received < ?`

	count, qerr := litesql.QueryRow(ctx, tx, scanCount, countSQL,
		payments.EventStatusDone,
		cutoff,
	)
	if qerr != nil {
		return 0, fmt.Errorf("stripe count prunable events: %w", qerr)
	}

	const deleteSQL = `
	delete from stripe_events
	where
		status = ?
		and received < ?`

	if err := s.ldb.Exec(ctx, tx, count, deleteSQL,
		payments.EventStatusDone,
		cutoff,
	); err != nil {
		return 0, fmt.Errorf("stripe prune events: %w", err)
	}

	return count, tx.Commit()
}

// CurrentPlan returns the plan currently assigned to the given user.
func (s *stripesql) CurrentPlan(ctx scope.C, uid users.Identity) (records.PlansEnum, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, xerr
	}
	defer done()

	const statement = `
	select plans_id from users
	where
		id = ?`

	f := func(scan litesql.ScanFunc) (records.PlansEnum, error) {
		var plan records.PlansEnum
		return plan, scan(&plan)
	}

	return litesql.QueryRow(ctx, tx, f, statement, uid)
}
