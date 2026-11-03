// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
	"github.com/stripe/stripe-go/v85"
)

func newPortalHarness(t *testing.T, client payments.Client) (database.Store, *mux.Router, *mocks.Sessions) {
	t.Helper()

	store := mockdb.Open(t)
	handler := &stripePortalAPI{
		log:         ulog.New("test"),
		store:       store,
		client:      client,
		environment: env,
	}

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle(handler.Path(), sessions(handler, scache)).Methods(http.MethodPost)
	return store, router, scache
}

func TestStripePortal_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, routes, _ := newPortalHarness(t, &mocks.StripeClient{})

	request := requestPost(t, "/internal/patreon/portal", nil, nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusSeeOther, w.Code)
	must.Eq(t, "/login", w.Header().Get("Location"))
}

func TestStripePortal_FreeUser(t *testing.T) {
	t.Parallel()

	store, routes, scache := newPortalHarness(t, &mocks.StripeClient{})
	uid := newTestUser(t, store, "portal-free")
	cookie := scache.Create(uid, time.Minute)

	request := requestPost(t, "/internal/patreon/portal", cookie, nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusSeeOther, w.Code)
	must.Eq(t, "/patreon", w.Header().Get("Location"))
}

func TestStripePortal_NoCustomer(t *testing.T) {
	t.Parallel()

	store, routes, scache := newPortalHarness(t, &mocks.StripeClient{})
	uid := newTestUser(t, store, "portal-no-customer")
	upgradeToPatreonWithoutCustomer(t, store, uid)
	cookie := scache.Create(uid, time.Minute)

	request := requestPost(t, "/internal/patreon/portal", cookie, nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusInternalServerError, w.Code)
}

func TestStripePortal_Success(t *testing.T) {
	t.Parallel()

	store, routes, scache := newPortalHarness(t, &mocks.StripeClient{
		MockBillingPortalSession: &stripe.BillingPortalSession{
			URL: "https://billing.stripe.test/session",
		},
	})
	uid := newTestUser(t, store, "portal-success")
	setPatreonPlan(t, store, uid)
	cookie := scache.Create(uid, time.Minute)

	request := requestPost(t, "/internal/patreon/portal", cookie, nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusSeeOther, w.Code)
	must.Eq(t, "https://billing.stripe.test/session", w.Header().Get("Location"))
}

func upgradeToPatreonWithoutCustomer(t *testing.T, store database.Store, uid users.Identity) {
	t.Helper()

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	err := store.Stripe().UpsertSubscription(ctx, &payments.UpdateStripeSubscription{
		User: uid,
		Plan: records.PlansPatreon,
		Sub: &payments.StripeSubscription{
			SubscriptionID:   "sub_no_customer",
			PriceID:          "price_patreon",
			Status:           "active",
			CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour).Unix(),
		},
	})
	must.NoError(t, err)
}
