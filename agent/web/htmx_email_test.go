// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func newEmailRouter(t *testing.T, store database.Store) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle((&emailHTMX{}).Path(), &emailHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	return router, scache
}

func postEmailForm(t *testing.T, method, urlPath string, values url.Values) *http.Request {
	t.Helper()

	r, err := http.NewRequestWithContext(t.Context(), method, urlPath, strings.NewReader(values.Encode()))
	must.NoError(t, err)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestEmailHTMX_Get_RendersForm(t *testing.T) {
	t.Parallel()

	router, scache := newEmailRouter(t, mockdb.Open(t))
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/email/edit", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `hx-put="/internal/email/edit"`)
	must.StrContains(t, body, `name="email"`)
	must.StrContains(t, body, `value="nobody@onlylogins.invalid"`)
	must.StrContains(t, body, `<span id="error">`)
}

func TestEmailHTMX_Get_NoSession_Redirects(t *testing.T) {
	t.Parallel()

	router, scache := newEmailRouter(t, mockdb.Open(t))
	ss := session(router, scache)

	r := requestGet(t, "/internal/email/edit", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestEmailHTMX_Put_HappyPath(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newEmailRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := postEmailForm(t, http.MethodPut, "/internal/email/edit", url.Values{
		"email": {"alice.new@example.com"},
	})
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/profile", w.Header().Get("HX-Redirect"))

	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	u, err := store.Users().Lookup(ctx, mocks.Nobody)
	must.NoError(t, err)
	must.Eq(t, "alice.new@example.com", u.Email)
}

func TestEmailHTMX_Put_EmptyEmail_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newEmailRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := postEmailForm(t, http.MethodPut, "/internal/email/edit", url.Values{
		"email": {""},
	})
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, w.Body.String(), "email is too short")
}

func TestEmailHTMX_Put_MissingAt_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newEmailRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := postEmailForm(t, http.MethodPut, "/internal/email/edit", url.Values{
		"email": {"alice.example.com"},
	})
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, w.Body.String(), "email must contain @ symbol")
}

func TestEmailHTMX_Put_TooLong_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newEmailRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	long := strings.Repeat("a", 120) + "@example.com"
	r := postEmailForm(t, http.MethodPut, "/internal/email/edit", url.Values{
		"email": {long},
	})
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, w.Body.String(), "email is too long")
}

func TestEmailHTMX_Put_NoSession_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newEmailRouter(t, store)
	ss := session(router, scache)

	r := postEmailForm(t, http.MethodPut, "/internal/email/edit", url.Values{
		"email": {"alice.new@example.com"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}
