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

func NewSyncSubscriptions(
	config *crons.Config,
	store database.Store,
) *SyncSubscriptions {
	return &SyncSubscriptions{
		expression: cronexpr.MustParse(config.Expression),
		store:      store,
		log:        ulog.New("cron/sync-subscriptions"),
	}
}

type SyncSubscriptions struct {
	store      database.Store
	expression *cronexpr.Expression
	log        *ulog.Log
}

func (s *SyncSubscriptions) Name() string {
	return "sync-subscriptions"
}

func (s *SyncSubscriptions) Schedule() *cronexpr.Expression {
	return s.expression
}

func (s *SyncSubscriptions) Start() *crons.RunMetric {
	start := time.Now()

	ctx, cancel := scope.TTL(1 * time.Minute)
	defer cancel()

	downgraded, err := s.store.Stripe().DowngradeExpired(ctx)
	if err != nil {
		s.log.E.Fmt("failed to downgrade expired subscriptions: %v", err)
		return &crons.RunMetric{
			Code:  crons.ExitFailure,
			Start: start,
			Stop:  time.Now(),
		}
	}

	s.log.D.Fmt("downgraded %d expired subscriptions", downgraded)
	return &crons.RunMetric{
		Code:  crons.ExitSuccess,
		Start: start,
		Stop:  time.Now(),
	}
}
