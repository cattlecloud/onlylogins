// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"fmt"
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/oauth"
	"cattlecloud.net/go/webtools/middles/oauth/nonces"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/memcache"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/sessions"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/ui"
	"github.com/VictoriaMetrics/metrics"
	"github.com/gorilla/mux"
)

type Application struct {
	config      *Config
	log         *ulog.Log
	environment runtime.Environment
	httpc       *http.Client
	council     users.Council
	store       database.Store
	scache      *sessions.Cache
	metrics     *metrics.Set
	mcache      memcache.Client
	mint        nonces.Mint
	vault       encrypt.Vault
	payments    payments.Client
}

func (a *Application) createRoutes(router *mux.Router) {
	a.handleStatics(router)

	a.handleAPI(router, &robotsAPI{
		log:         ulog.New("api/robots"),
		environment: a.environment,
	})

	a.handleAPI(router, &sitemapAPI{
		log:         ulog.New("api/sitemap"),
		environment: a.environment,
	})

	a.handleAPI(router, &stripeWebhookAPI{
		log:       ulog.New("internal/stripe/webhooks"),
		store:     a.store,
		client:    a.payments,
		processor: payments.NewProcessor(a.store.Stripe(), a.payments),
	})

	a.handleComponent(router, &stripeCheckoutAPI{
		log:         ulog.New("internal/stripe/checkout"),
		store:       a.store,
		client:      a.payments,
		environment: a.environment,
	})

	a.handleComponent(router, &stripePortalAPI{
		log:         ulog.New("internal/stripe/portal"),
		store:       a.store,
		client:      a.payments,
		environment: a.environment,
	})

	a.handleComponent(router, &accountHTMX{
		log:     ulog.New("internal/account"),
		store:   a.store,
		session: a.scache,
	})

	a.handleComponent(router, &weblinkHTMX{
		log:   ulog.New("htmx/weblink"),
		store: a.store,
	})

	a.handleComponent(router, &weblinkEntryHTMX{
		log:   ulog.New("htmx/weblink/entry"),
		store: a.store,
	})

	a.handleComponent(router, &usernameHTMX{
		log:   ulog.New("htmx/username"),
		store: a.store,
	})

	a.handleComponent(router, &usernameEntryHTMX{
		log:   ulog.New("htmx/username/entry"),
		store: a.store,
	})

	a.handleComponent(router, &passwordHTMX{
		log:   ulog.New("htmx/password"),
		store: a.store,
	})

	a.handleComponent(router, &passwordEntryHTMX{
		log:   ulog.New("htmx/password/entry"),
		store: a.store,
	})

	a.handleComponent(router, &keyvalHTMX{
		log:   ulog.New("htmx/keyval"),
		store: a.store,
	})

	a.handleComponent(router, &keyvalEntryHTMX{
		log:   ulog.New("htmx/keyval/entry"),
		store: a.store,
	})

	a.handleComponent(router, &noteHTMX{
		log:   ulog.New("htmx/note"),
		store: a.store,
	})

	a.handleComponent(router, &noteEntryHTMX{
		log:   ulog.New("htmx/note/entry"),
		store: a.store,
	})

	a.handleComponent(router, &renameFolderHTMX{
		log:   ulog.New("htmx/folder/rename"),
		store: a.store,
	})

	a.handleComponent(router, &renameDocumentHTMX{
		log:   ulog.New("htmx/document/rename"),
		store: a.store,
	})

	a.handleComponent(router, &migrateDocumentHTMX{
		log:   ulog.New("htmx/document/migrate"),
		store: a.store,
	})

	a.handleComponent(router, &emailHTMX{
		log:   ulog.New("htmx/email"),
		store: a.store,
	})

	a.handleComponent(router, &historyHTMX{
		log:   ulog.New("htmx/history"),
		store: a.store,
	})

	a.handleModerate(router, &moderationHTTP{
		log:   ulog.New("http/mod/home"),
		store: a.store,
		html:  ui.Mod("mod_home.html"),
	})

	a.handleModerate(router, &modUsersHTTP{
		log:   ulog.New("http/mod/users"),
		store: a.store,
		html:  ui.Mod("mod_users.html"),
	})

	a.handleModerate(router, &modUserHTTP{
		log:   ulog.New("http/mod/user"),
		store: a.store,
		html:  ui.Mod("mod_user.html"),
	})

	a.handleNormal(router, &loginHTTP{
		log:         ulog.New("http/login"),
		environment: a.environment,
		session:     a.scache,
		store:       a.store,
		html:        ui.Load("login.html"),
	})

	a.handleNormal(router, &profileHTTP{
		log:   ulog.New("http/profile"),
		store: a.store,
		html:  ui.Load("profile.html"),
	})

	a.handleNormal(router, &passphraseHTTP{
		log:     ulog.New("http/profile/passphrase"),
		store:   a.store,
		session: a.scache,
		html:    ui.Load("passphrase.html"),
	})

	a.handleNormal(router, &aboutHTTP{
		log:         ulog.New("http/about"),
		store:       a.store,
		environment: a.environment,
		html:        ui.Load("about.html"),
	})

	a.handleNormal(router, &tosHTTP{
		log:         ulog.New("http/tos"),
		store:       a.store,
		environment: a.environment,
		html:        ui.Load("tos.html"),
	})

	a.handleNormal(router, &privacyHTTP{
		log:         ulog.New("http/privacy"),
		store:       a.store,
		environment: a.environment,
		html:        ui.Load("privacy.html"),
	})

	a.handleNormal(router, &helpHTTP{
		log:         ulog.New("http/help"),
		store:       a.store,
		environment: a.environment,
		html:        ui.Load("help.html"),
	})

	a.handleNormal(router, &shareHTTP{
		log:         ulog.New("http/share"),
		store:       a.store,
		vault:       a.vault,
		environment: a.environment,
		html:        ui.Load("share.html"),
	})

	a.handleNormal(router, &shareOpenHTTP{
		log:   ulog.New("http/share/open"),
		store: a.store,
		vault: a.vault,
		now:   time.Now,
		html:  ui.Load("share_open.html"),
	})

	a.handleNormal(router, &exportHTTP{
		log:         ulog.New("http/export"),
		store:       a.store,
		environment: a.environment,
		html:        ui.Load("export.html"),
	})

	a.handleNormal(router, &patreonHTTP{
		log:         ulog.New("http/patreon"),
		store:       a.store,
		environment: a.environment,
		payments:    a.payments,
		html:        ui.Load("patreon.html"),
	})

	a.handleNormal(router, &documentHTTP{
		log:         ulog.New("document"),
		store:       a.store,
		environment: a.environment,
		html:        ui.Load("document.html"),
	})

	a.handleNormal(router, &folderHTTP{
		log:         ulog.New("http/folder"),
		store:       a.store,
		environment: a.environment,
		html:        ui.Load("folder.html"),
	})

	a.handleNormal(router, &foldersHTTP{
		log:         ulog.New("http/folders"),
		store:       a.store,
		environment: a.environment,
		html:        ui.Load("folders.html"),
	})

	// index must be listed last
	a.handleNormal(router, &indexHTTP{
		log:   ulog.New("http/index"),
		store: a.store,
		html:  ui.Load("index.html"),
	})
}

type Objects struct {
	Environment runtime.Environment
	Council     users.Council
	Store       database.Store
	Scache      *sessions.Cache
	HTTPC       *http.Client
	Metrics     *metrics.Set
	Mcache      memcache.Client
	Mint        nonces.Mint
	Vault       encrypt.Vault
	Payments    payments.Client
}

func New(config *Config, objects *Objects) *Application {
	return &Application{
		config:      config,
		log:         ulog.New("http"),
		environment: objects.Environment,
		httpc:       objects.HTTPC,
		council:     objects.Council,
		store:       objects.Store,
		scache:      objects.Scache,
		metrics:     objects.Metrics,
		mcache:      objects.Mcache,
		mint:        objects.Mint,
		vault:       objects.Vault,
		payments:    objects.Payments,
	}
}

func (a *Application) Start(ctx scope.C) {
	router := mux.NewRouter()
	a.createRoutes(router)

	csrf := http.NewCrossOriginProtection()
	address := fmt.Sprintf("%s:%d", a.config.Address, a.config.Port)
	server := &http.Server{
		Addr:    address,
		Handler: csrf.Handler(router),
	}

	// serve http async in goroutine
	go func() {
		_ = server.ListenAndServe()
	}()

	a.log.I.Fmt("http listening on %s", address)

	// wait for shutdown signal
	<-ctx.Done()

	// force shutdown http server
	_ = server.Shutdown(ctx)
}

type consumer int

const (
	normal consumer = iota
	component
	robotic
	moderate
)

type endpoint interface {
	Consumer() consumer
	Path() string
	ServeHTTP(http.ResponseWriter, *http.Request)
	Methods() []string
}

type web interface {
	endpoint
	Metric() string
}

type api interface {
	endpoint
}

// handleStatics is a single special case for serving embedded statics based on
// the included /ui/ directory, and includes images, css, html files, etc.
func (a *Application) handleStatics(router *mux.Router) {
	router.PathPrefix("/ui/").Handler(&middles.CacheHTTP{
		Env: a.environment,
		H:   http.StripPrefix("/ui/", http.FileServerFS(ui.FileTree)),
	})
}

func (a *Application) handleNormal(router *mux.Router, h web) {
	check(normal, h)

	// sliding session cookies + multi-level session cache (volatile + memcache)
	ss := session(a.sliding(h), a.scache)

	trace := &duration{
		sum:  a.metrics.NewSummary("ol_web_" + h.Metric()),
		next: ss,
	}

	router.Handle(h.Path(), trace).Methods(h.Methods()...)
}

func (a *Application) handleModerate(router *mux.Router, h api) {
	check(moderate, h)

	gate := &moderationGate{
		council: a.council,
		next:    h,
		log:     ulog.New("middle/moderation-gate"),
	}

	ss := session(a.sliding(gate), a.scache)

	router.Handle(h.Path(), ss).Methods(h.Methods()...)
}

func (a *Application) handleComponent(router *mux.Router, h api) {
	check(component, h)
	ss := session(a.sliding(h), a.scache)
	router.Handle(h.Path(), ss).Methods(h.Methods()...)
}

func (a *Application) handleAPI(router *mux.Router, h api) {
	check(robotic, h)

	router.Handle(h.Path(), h).Methods(h.Methods()...)
}

func session(h http.Handler, scache middles.Sessions[users.Identity]) *middles.SetSession[*oauth.CookieContent[users.Identity], users.Identity] {
	ss := &middles.SetSession[*oauth.CookieContent[users.Identity], users.Identity]{
		SessionCookieName: "onlylogins-session",
		Sessions:          scache,
		Next:              h,
	}
	return ss
}

// check ensures we have called the correct wrapping handler for the given
// endpoint - avoiding problems of incorrect security boundary, session scope,
// etc.
func check(exp consumer, h endpoint) {
	implements := h.Consumer()
	if implements != exp {
		p := h.Path()
		panic("web: incorrect handler implementation for path " + p)
	}
}
