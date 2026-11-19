// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package cronjobs

import (
	"time"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/hashicorp/cronexpr"
)

func NewSnapshots(
	config *crons.Config,
	dbc *database.Config,
	store database.Store,
) *Snapshots {
	return &Snapshots{
		expression: cronexpr.MustParse(config.Expression),
		store:      store,
		config:     dbc,
		log:        ulog.New("cron/db-snapshots"),
	}
}

type Snapshots struct {
	store      database.Store
	expression *cronexpr.Expression
	config     *database.Config
	log        *ulog.Log
}

func (s *Snapshots) Name() string {
	return "db-snapshots"
}

func (s *Snapshots) Schedule() *cronexpr.Expression {
	return s.expression
}

func (s *Snapshots) Start() *crons.RunMetric {
	start := time.Now()

	if err := s.create(); err != nil {
		s.log.E.Fmt("stopping with failure: %v", err)
		return crons.Failure(start)
	}

	return crons.Success(start)
}

func (s *Snapshots) create() error {
	ctx, cancel := scope.TTL(1 * time.Minute)
	defer cancel()

	progress := func(total, remaining int) {
		s.log.D.Fmt("progress %d/%d", remaining, total)
	}

	opts := &litesql.SnapshotOptions{
		Directory: s.config.Snapshots,
		Retention: 2,  // 1 + 2 more
		Step:      -1, // all at once
		Progress:  progress,
	}

	return s.store.Snaps().Create(ctx, opts)
}
