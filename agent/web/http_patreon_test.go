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
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func newTestUser(t *testing.T, store database.Store, username string) users.Identity {
	t.Helper()

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	salt := encrypt.Salt()
	passphrase := conceal.New("correct horse battery staple")
	key := encrypt.DeriveKey(passphrase, salt)
	verifier := encrypt.MakeVerifier(key)

	id, err := store.Users().Create(ctx, &users.CreateUserRequest{
		Email:      username + "@example.com",
		Username:   username,
		Passphrase: passphrase,
		Salt:       salt,
	}, verifier)
	must.NoError(t, err)
	must.Positive(t, id)
	return id
}

func setPatreonPlan(t *testing.T, store database.Store, uid users.Identity) {
	t.Helper()

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	periodEnd := time.Now().Add(30 * 24 * time.Hour).Unix()
	err := store.Stripe().UpsertSubscription(ctx, &payments.UpdateStripeSubscription{
		User: uid,
		Plan: records.PlansPatreon,
		Sub: &payments.StripeSubscription{
			SubscriptionID:   "sub_test123",
			PriceID:          "price_patreon",
			Status:           "active",
			CurrentPeriodEnd: periodEnd,
		},
	})
	must.NoError(t, err)

	err = store.Stripe().SetCustomer(ctx, uid, "cus_test123")
	must.NoError(t, err)
}

func newPatreonPageHarness(t *testing.T, client payments.Client) (database.Store, *mux.Router, *mocks.Sessions) {
	t.Helper()

	store := mockdb.Open(t)
	handler := &patreonHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		payments:    client,
		html:        ui.Load("patreon.html"),
	}

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle(handler.Path(), sessions(handler, scache)).Methods(http.MethodGet)
	return store, router, scache
}

func TestPatreon_LoggedOut(t *testing.T) {
	t.Parallel()

	_, routes, _ := newPatreonPageHarness(t, &mocks.StripeClient{
		MockPatreonProductID: "price_patreon",
	})

	request := requestGet(t, "/patreon", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, body, `<div class="middle patreon">`)
	must.StrContains(t, body, "Log in")
	must.StrContains(t, body, "What the Patreon plan includes")
	must.StrNotContains(t, body, `action="/internal/patreon/checkout"`)
}

func TestPatreon_FreeUser(t *testing.T) {
	t.Parallel()

	store, routes, scache := newPatreonPageHarness(t, &mocks.StripeClient{
		MockPatreonProductID: "price_patreon",
	})

	uid := newTestUser(t, store, "free-user")
	cookie := scache.Create(uid, time.Minute)

	request := requestGet(t, "/patreon", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, body, "/ 256 documents")
	must.StrContains(t, body, `action="/internal/patreon/checkout"`)
	must.StrContains(t, body, "Current plan")
	must.StrNotContains(t, body, `action="/internal/patreon/portal"`)
}

func TestPatreon_PatreonUser(t *testing.T) {
	t.Parallel()

	store, routes, scache := newPatreonPageHarness(t, &mocks.StripeClient{
		MockPatreonProductID: "price_patreon",
	})

	uid := newTestUser(t, store, "patreon-user")
	setPatreonPlan(t, store, uid)
	cookie := scache.Create(uid, time.Minute)

	request := requestGet(t, "/patreon", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, body, "/ 8192 documents")
	must.StrContains(t, body, `action="/internal/patreon/portal"`)
	must.StrContains(t, body, "Manage subscription")
	must.StrContains(t, body, "Renews")
	must.StrNotContains(t, body, `action="/internal/patreon/checkout"`)
}

func TestPatreon_PaymentsDisabled(t *testing.T) {
	t.Parallel()

	store, routes, scache := newPatreonPageHarness(t, &mocks.StripeClient{})

	uid := newTestUser(t, store, "disabled-user")
	cookie := scache.Create(uid, time.Minute)

	request := requestGet(t, "/patreon", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, body, "Subscriptions are not configured on this instance.")
	must.StrNotContains(t, body, `action="/internal/patreon/checkout"`)
}
