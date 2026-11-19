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

func NewMonthlyDBM(
	config *crons.Config,
	store database.Store,
) *MonthlyDBM {
	return &MonthlyDBM{
		expression: cronexpr.MustParse(config.Expression),
		store:      store,
		log:        ulog.New("cron/db-monthly"),
	}
}

type MonthlyDBM struct {
	store      database.Store
	expression *cronexpr.Expression
	log        *ulog.Log
}

func (c *MonthlyDBM) Name() string {
	return "db-monthly"
}

func (c *MonthlyDBM) Schedule() *cronexpr.Expression {
	return c.expression
}

func (c *MonthlyDBM) Start() *crons.RunMetric {
	const expiration = 32 * 24 * time.Hour
	start := time.Now()

	ctx, cancel := scope.TTL(1 * time.Minute)
	defer cancel()

	cronruns, crerr := c.store.Crons().Prune(ctx, expiration)
	if crerr != nil {
		c.log.E.Fmt("failed to prune cron runs: %v", crerr)
		return crons.Failure(start)
	}
	c.log.I.Fmt("pruned %d old cron runs", cronruns)

	if err := c.store.Maint().Vacuum(ctx); err != nil {
		c.log.E.Fmt("failed to vacuum: %v", err)
		return crons.Failure(start)
	}

	return crons.Success(start)
}
