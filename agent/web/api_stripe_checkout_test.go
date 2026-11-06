// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/payments"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
	"github.com/stripe/stripe-go/v85"
)

func newCheckoutHarness(t *testing.T, client payments.Client) (database.Store, *mux.Router, *mocks.Sessions) {
	t.Helper()

	store := mockdb.Open(t)
	handler := &stripeCheckoutAPI{
		log:         ulog.New("test"),
		store:       store,
		client:      client,
		environment: env,
	}

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle(handler.Path(), session(handler, scache)).Methods(http.MethodPost)
	return store, router, scache
}

func TestStripeCheckout_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, routes, _ := newCheckoutHarness(t, &mocks.StripeClient{
		MockPatreonProductID: "price_patreon",
	})

	request := requestPost(t, "/internal/patreon/checkout", nil, nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusSeeOther, w.Code)
	must.Eq(t, "/login", w.Header().Get("Location"))
}

func TestStripeCheckout_NotConfigured(t *testing.T) {
	t.Parallel()

	store, routes, scache := newCheckoutHarness(t, &mocks.StripeClient{})
	uid := newTestUser(t, store, "not-configured")
	cookie := scache.Create(uid, time.Minute)

	request := requestPost(t, "/internal/patreon/checkout", cookie, nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusInternalServerError, w.Code)
}

func TestStripeCheckout_CustomerError(t *testing.T) {
	t.Parallel()

	store, routes, scache := newCheckoutHarness(t, &mocks.StripeClient{
		MockPatreonProductID: "price_patreon",
		MockCustomerErr:      errors.New("stripe is down"),
	})
	uid := newTestUser(t, store, "customer-error")
	cookie := scache.Create(uid, time.Minute)

	request := requestPost(t, "/internal/patreon/checkout", cookie, nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusInternalServerError, w.Code)
}

func TestStripeCheckout_Success(t *testing.T) {
	t.Parallel()

	store, routes, scache := newCheckoutHarness(t, &mocks.StripeClient{
		MockPatreonProductID: "price_patreon",
		MockCustomerID:       "cus_checkout",
		MockCheckoutSession: &stripe.CheckoutSession{
			URL: "https://checkout.stripe.test/session",
		},
	})
	uid := newTestUser(t, store, "checkout-success")
	cookie := scache.Create(uid, time.Minute)

	request := requestPost(t, "/internal/patreon/checkout", cookie, nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusSeeOther, w.Code)
	must.Eq(t, "https://checkout.stripe.test/session", w.Header().Get("Location"))

	customerID, found, err := store.Stripe().GetCustomer(t.Context(), uid)
	must.NoError(t, err)
	must.True(t, found)
	must.Eq(t, "cus_checkout", customerID)
}
