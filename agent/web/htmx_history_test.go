// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/gorilla/mux"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func newHistoryRouter(t *testing.T, store database.Store) *mux.Router {
	t.Helper()

	router := mux.NewRouter()
	router.Handle((&historyHTMX{}).Path(), &historyHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	return router
}

func TestHistoryHTMX_Get_LoggedIn(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router := newHistoryRouter(t, store)
	scache := mocks.NewSessions()
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	ctx, cancel := scope.TTL(testwait)
	t.Cleanup(cancel)

	audit := store.Audit()
	must.NoError(t, audit.Record(ctx, &users.LoginRequest{
		Username:   "_nobody",
		Passphrase: conceal.New("hunter2"),
		UserAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		IP:         "203.0.113.7",
	}, true))
	must.NoError(t, audit.Record(ctx, &users.LoginRequest{
		Username:   "_nobody",
		Passphrase: conceal.New("hunter2"),
		UserAgent:  "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		IP:         "198.51.100.9",
	}, false))

	r := requestGet(t, "/internal/profile/history", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, "203.0.113.7")
	must.StrContains(t, body, "198.51.100.9")
	must.StrContains(t, body, "success")
	must.StrContains(t, body, "failure")
	must.StrContains(t, body, "Chrome")
	must.StrContains(t, body, "desktop")
	must.StrContains(t, body, "phone")
}

func TestHistoryHTMX_Get_NotLoggedIn(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router := newHistoryRouter(t, store)
	scache := mocks.NewSessions()
	ss := session(router, scache)

	r := requestGet(t, "/internal/profile/history", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}
