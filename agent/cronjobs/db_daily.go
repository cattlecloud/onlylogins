// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package cronjobs

import (
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/hashicorp/cronexpr"
)

func NewDailyDBM(config *crons.Config, store database.Store) *DailyDBM {
	return &DailyDBM{
		expression: cronexpr.MustParse(config.Expression),
		store:      store,
		log:        ulog.New("cron/db-daily"),
	}
}

type DailyDBM struct {
	store      database.Store
	expression *cronexpr.Expression
	log        *ulog.Log
}

func (c *DailyDBM) Name() string {
	return "db-daily"
}

func (c *DailyDBM) Schedule() *cronexpr.Expression {
	return c.expression
}

func (c *DailyDBM) Start() *crons.RunMetric {
	start := time.Now()

	ctx, cancel := scope.TTL(1 * time.Minute)
	defer cancel()

	// guaranteed to be expired by these times
	const (
		expireStripe = 60 * 24 * time.Hour
		expireShares = 32 * 24 * time.Hour
		expireLogins = 7 * 24 * time.Hour
	)

	// expire outdated shared secrets
	shares, serr := c.store.Shares(nil).Prune(ctx, expireShares)
	if serr != nil {
		c.log.E.Fmt("failed to prune shares: %v", serr)
		return crons.Failure(start)
	}
	c.log.I.Fmt("pruned %d old shares", shares)

	// expire login data from over a week ago
	if err := c.store.Audit().Prune(ctx, expireLogins); err != nil {
		c.log.E.Fmt("failed to prune audit records: %v", err)
		return crons.Failure(start)
	}
	c.log.I.Fmt("pruned old login and bogus attempts")

	// expire stripe events more than two months old
	events, eerr := c.store.Stripe().PruneEvents(ctx, expireStripe)
	if eerr != nil {
		c.log.E.Fmt("failed to prune stripe events: %v", eerr)
		return crons.Failure(start)
	}
	c.log.I.Fmt("pruned %d old stripe events", events)

	// run the database optimize pragma
	if err := c.store.Maint().Optimize(ctx); err != nil {
		c.log.E.Fmt("failed to optimize: %v", err)
		return crons.Failure(start)
	}
	c.log.I.Fmt("database has been optimized")

	// we are done
	return crons.Success(start)
}
