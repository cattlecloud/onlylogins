// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package agent

import (
	"time"

	"cattlecloud.net/go/memc"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/httpclient"
	"cattlecloud.net/go/webtools/middles/oauth"
	"cattlecloud.net/go/webtools/middles/oauth/nonces"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/agent/cronjobs"
	"cattlecloud.net/onlylogins/agent/web"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/crons"
	"cattlecloud.net/onlylogins/pkg/crons/dispatch"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/memcache"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/sessions"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/VictoriaMetrics/metrics"
	"github.com/shoenig/ignore"
)

type OnlyLogins struct {
	log    *ulog.Log
	config *Config
}

func New(config *Config) *OnlyLogins {
	return &OnlyLogins{
		log:    ulog.New("agent"),
		config: config,
	}
}

func (l *OnlyLogins) Start(parent scope.C) error {
	l.log.I.Fmt("starting up each component ...")
	db, dberr := l.connectDB(parent)
	if dberr != nil {
		return dberr
	}
	defer ignore.Close(db)

	// metrics accumulation
	mets := metrics.NewSet()

	// create memcached client
	mc := memc.New(
		l.config.Cache.Instances,
		memc.SetDialTimeout(l.config.Cache.DialTimeout),
		memc.SetDefaultTTL(l.config.Cache.DefaultTTL),
		memc.SetIdleConnections(l.config.Cache.IdleConnections),
	)
	mcache := memcache.New(mc, mets)

	// create oauth token mint
	mint := nonces.New()

	// create an http client we can use everywhere
	httpc := httpclient.Get()

	// create the environment configuration
	env := runtime.Setup(l.config.Runtime)

	// create a cookies factory
	cookies := &oauth.CookieFactory[users.Identity]{
		Name:   "onlylogins-session",
		Secure: env.NonLocal(),
		Clock:  time.Now,
	}

	// create sessions cache with volatile + memcached backends
	scache := oauth.NewSessions(
		cookies,
		sessions.New(mcache), // our own shim between memcache and oauth
	)

	// administration council
	council := users.NewCouncil(l.config.Users.Administrators)

	// shared shares vault
	vault := encrypt.NewFromConfig(l.config.Vault)

	// stripe payments client
	stripec := payments.New(l.config.Stripe)

	// startup http web application
	webapp := web.New(l.config.HTTP, &web.Objects{
		Environment: env,
		Council:     council,
		Store:       db,
		Scache:      scache,
		HTTPC:       httpc,
		Metrics:     mets,
		Mcache:      mcache,
		Mint:        mint,
		Vault:       vault,
		Payments:    stripec,
	})
	go webapp.Start(parent)

	// startup cron framework and registrations
	l.crons(parent, db, stripec)

	// wait for overall context to complete
	<-parent.Done()
	l.log.I.Fmt("exit; goodbye!")
	return nil
}

func (l *OnlyLogins) connectDB(parent scope.C) (database.Store, error) {
	// open database file
	db, oerr := database.Open(l.config.Database)
	if oerr != nil {
		return nil, oerr
	}

	// enable short timeout
	ctx, cancel := scope.WithTTL(parent, 3*time.Second)
	defer cancel()

	// execute initialization sql statements
	if err := db.Init(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func (l *OnlyLogins) crons(parent scope.C, store database.Store, client payments.Client) {
	dc := dispatch.New(parent, store.Crons())

	register := func(job crons.Cronjob) {
		if err := dc.Register(job); err != nil {
			l.log.E.Fmt("unable to register cronjob %s: %v", job.Name(), err)
		}
	}

	register(cronjobs.NewSnapshots(
		l.config.Crons.Snapshots,
		l.config.Database,
		store,
	))

	register(cronjobs.NewDailyDBM(
		l.config.Crons.DailyDBM,
		store,
	))

	register(cronjobs.NewWeeklyDBM(
		l.config.Crons.WeeklyDBM,
		store,
	))

	register(cronjobs.NewMonthlyDBM(
		l.config.Crons.MonthlyDBM,
		store,
	))

	register(cronjobs.NewSyncSubscriptions(
		l.config.Crons.SubscriptionSync,
		store,
	))

	register(cronjobs.NewRetryStripeEvents(
		l.config.Crons.StripeEvents,
		store,
		client,
	))
}
