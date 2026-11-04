package database

import (
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/users"
)

type Stripe interface {
	// customers
	GetCustomer(scope.C, users.Identity) (string, bool, error)
	SetCustomer(scope.C, users.Identity, string) error

	// subscriptions
	GetSubscription(scope.C, users.Identity) (*payments.StripeSubscription, error)
	GetUserBySubscription(scope.C, string) (users.Identity, error)
	UpsertSubscription(scope.C, *payments.UpdateStripeSubscription) error
	DowngradeExpired(scope.C) (int, error)
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
		return fmt.Errorf("stripe customers set: %w", err)
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

	uid, err := litesql.QueryRow(ctx, tx, f, statement, subscriptionID)
	if err != nil {
		return 0, fmt.Errorf("no user found for subscription %q", subscriptionID)
	}

	return uid, nil
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
