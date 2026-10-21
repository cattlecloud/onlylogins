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
	"cattlecloud.net/onlylogins/agent/web"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/memcache"
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
	})
	go webapp.Start(parent)

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
