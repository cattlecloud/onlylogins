// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

// openEnv bundles the test environment shared by all open-endpoint tests:
// a single in-memory store, a session cache, a router pointing at the
// share-open handler with a specific vault and `now`, and a pre-seeded share.
type openEnv struct {
	router   *mux.Router
	scache   *mocks.Sessions
	uid      records.SharesUnique
	fullText string
}

// newOpenEnv builds an openEnv with one share pre-seeded into the store the
// handler is going to read from. They share the same vault so decryption works.
func newOpenEnv(t *testing.T, nowForHandler func() time.Time, secret string) *openEnv {
	t.Helper()

	store := mockdb.Open(t)
	vault := mocks.NewTestCrypt().Vault()

	uid := seedShare(t, store, vault, secret)

	scache := mocks.NewSessions()
	router := mux.NewRouter()

	router.Handle((&shareOpenHTTP{}).Path(), &shareOpenHTTP{
		log:   ulog.New("test"),
		store: store,
		vault: vault,
		now:   nowForHandler,
		html:  ui.Load("share_open.html"),
	})

	return &openEnv{
		router:   router,
		scache:   scache,
		uid:      uid,
		fullText: secret,
	}
}

// seedShare inserts a single share into the given store, returning its uid.
func seedShare(t *testing.T, store database.Store, vault encrypt.Vault, secret string) records.SharesUnique {
	return seedShareAt(t, store, vault, secret, time.Now())
}

// seedShareAt is like seedShare but uses an explicit creation timestamp.
// Useful for testing the expiration path independently of `time.Now`.
func seedShareAt(t *testing.T, store database.Store, vault encrypt.Vault, secret string, creation time.Time) records.SharesUnique {
	t.Helper()

	ctx := t.Context()
	uid, err := store.Shares(vault).Create(ctx, &sitecontent.CreateShareRequest{
		User:   mocks.Nobody,
		Secret: conceal.New(secret),
		TTL:    900 * time.Second,
		Now:    creation,
	})
	must.NoError(t, err)
	return uid
}

func TestShareOpen_ServeHTTP_not_logged_in(t *testing.T) {
	t.Parallel()

	env := newOpenEnv(t, time.Now, "a top secret hush-hush thing")
	ss := session(env.router, env.scache)

	r := requestGet(t, "/share/open/"+string(env.uid), nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `<div class="middle shareopen">`)
	must.StrContains(t, body, env.fullText)
}

func TestShareOpen_ServeHTTP_not_found(t *testing.T) {
	t.Parallel()

	env := newOpenEnv(t, time.Now, "a top secret hush-hush thing")
	_ = env // share is irrelevant; we exercise the missing-share branch
	ss := session(env.router, env.scache)

	r := requestGet(t, "/share/open/nonexistent", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code) // render the missing-share page, not 404
	must.StrContains(t, w.Body.String(), "This share does not exist.")
}

func TestShareOpen_ServeHTTP_already_consumed(t *testing.T) {
	t.Parallel()

	env := newOpenEnv(t, time.Now, "a top secret hush-hush thing")
	ss := session(env.router, env.scache)

	// consume once
	r := requestGet(t, "/share/open/"+string(env.uid), nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), env.fullText)

	// consume a second time - should now be missing
	r2 := requestGet(t, "/share/open/"+string(env.uid), nil)
	w2 := httptest.NewRecorder()
	ss.ServeHTTP(w2, r2)
	must.Eq(t, http.StatusOK, w2.Code)
	must.StrContains(t, w2.Body.String(), "This share does not exist.")
}

func TestShareOpen_ServeHTTP_expired(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	vault := mocks.NewTestCrypt().Vault()

	// seed a share at a known time, then point the handler clock past the
	// TTL so the expiration branch fires.
	creation := time.Date(2026, time.November, 2, 13, 0, 0, 0, time.UTC)
	uid := seedShareAt(t, store, vault, "an expired share", creation)

	scache := mocks.NewSessions()
	router := mux.NewRouter()

	router.Handle((&shareOpenHTTP{}).Path(), &shareOpenHTTP{
		log:   ulog.New("test"),
		store: store,
		vault: vault,
		now:   func() time.Time { return creation.Add(901 * time.Second) },
		html:  ui.Load("share_open.html"),
	})

	ss := session(router, scache)
	r := requestGet(t, "/share/open/"+string(uid), nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), "This share does not exist.")
}
