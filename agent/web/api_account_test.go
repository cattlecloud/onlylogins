// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/pkg/sessions/cookies"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func newAccountRouter(t *testing.T) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle("/internal/logout", &accountHTMX{
		log:     ulog.New("test"),
		session: scache,
	})

	return router, scache
}

func TestAccountHTMX_Logout_ClearsCookies(t *testing.T) {
	t.Parallel()

	router, scache := newAccountRouter(t)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/logout", cookie, url.Values{})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/", w.Header().Get("HX-Redirect"))

	cleared := make(map[string]bool)
	for _, c := range w.Result().Cookies() {
		if c.Name == cookies.Session || c.Name == cookies.Encryption {
			cleared[c.Name] = c.Value == "" && !c.Expires.IsZero()
		}
	}
	must.True(t, cleared[cookies.Session])
	must.True(t, cleared[cookies.Encryption])
}

func TestAccountHTMX_Logout_InvalidatesServerSession(t *testing.T) {
	t.Parallel()

	router, scache := newAccountRouter(t)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	// first request logs the user out
	r := requestPost(t, "/internal/logout", cookie, url.Values{})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusFound, w.Code)

	// replaying the old cookie must no longer produce an active session
	r2 := requestPost(t, "/internal/logout", cookie, url.Values{})
	w2 := httptest.NewRecorder()
	ss.ServeHTTP(w2, r2)

	must.Eq(t, http.StatusTemporaryRedirect, w2.Code)
	must.Eq(t, "/login", w2.Header().Get("Location"))
}
