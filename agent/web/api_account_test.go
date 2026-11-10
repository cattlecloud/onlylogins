// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/sessions/cookies"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func newAccountRouter(t *testing.T, store database.Store) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle("/internal/logout", &accountHTMX{
		log:     ulog.New("test"),
		store:   store,
		session: scache,
	})

	return router, scache
}

func TestAccountHTMX_Logout_ClearsCookies(t *testing.T) {
	t.Parallel()

	router, scache := newAccountRouter(t, mockdb.Open(t))
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

	router, scache := newAccountRouter(t, mockdb.Open(t))
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

func TestAccountHTMX_Delete_FreeUser(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newAccountRouter(t, store)
	ss := session(router, scache)

	uid := newTestUser(t, store, "delete-me")
	cookie := scache.Create(uid, 1*time.Minute)

	r := requestDelete(t, "/internal/logout", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/", w.Header().Get("HX-Redirect"))

	// the account is gone
	ctx, cancel := scope.TTL(testwait)
	t.Cleanup(cancel)

	_, err := store.Users().Lookup(ctx, uid)
	must.Error(t, err)

	// the session cookies are cleared
	cleared := make(map[string]bool)
	for _, c := range w.Result().Cookies() {
		if c.Name == cookies.Session || c.Name == cookies.Encryption {
			cleared[c.Name] = c.Value == "" && !c.Expires.IsZero()
		}
	}
	must.True(t, cleared[cookies.Session])
	must.True(t, cleared[cookies.Encryption])
}

func TestAccountHTMX_Delete_PatreonUserBlocked(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newAccountRouter(t, store)
	ss := session(router, scache)

	uid := newTestUser(t, store, "subscriber")
	setPatreonPlan(t, store, uid)
	cookie := scache.Create(uid, 1*time.Minute)

	r := requestDelete(t, "/internal/logout", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	// the request is refused while the subscription is active
	must.Eq(t, http.StatusConflict, w.Code)
	must.Eq(t, "", w.Header().Get("HX-Redirect"))

	// the account still exists
	ctx, cancel := scope.TTL(testwait)
	t.Cleanup(cancel)

	profile, err := store.Users().Lookup(ctx, uid)
	must.NoError(t, err)
	must.Eq(t, "subscriber", profile.Username)
}

func TestAccountHTMX_Delete_CanceledSubscription(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newAccountRouter(t, store)
	ss := session(router, scache)

	uid := newTestUser(t, store, "ex-subscriber")
	setPatreonPlan(t, store, uid)

	ctx, cancel := scope.TTL(testwait)
	t.Cleanup(cancel)

	// the subscription is canceled and the plan is downgraded to free
	err := store.Stripe().UpsertSubscription(ctx, &payments.UpdateStripeSubscription{
		User: uid,
		Plan: records.PlansFree,
		Sub: &payments.StripeSubscription{
			SubscriptionID:   "sub_test123",
			PriceID:          "price_patreon",
			Status:           "canceled",
			CurrentPeriodEnd: 0,
		},
	})
	must.NoError(t, err)

	cookie := scache.Create(uid, 1*time.Minute)

	r := requestDelete(t, "/internal/logout", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/", w.Header().Get("HX-Redirect"))

	// the account is gone
	_, err = store.Users().Lookup(ctx, uid)
	must.Error(t, err)
}
