// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"net/http/httptest"
	"testing"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func TestModHome_ServeHTTP(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	routes := mux.NewRouter()
	routes.Handle("/moderate", &moderationHTTP{
		log:   ulog.New("test"),
		store: store,
		html:  ui.Mod("mod_home.html"),
	})

	request := requestGet(t, "/moderate", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<div class="modpage overview">`)
	must.StrNotContains(t, body, `stripe webhook events need attention`)
}

func TestModHome_StripeProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	created, err := store.Stripe().RecordEvent(ctx, &payments.StripeEvent{
		ID:      "evt_test_problem",
		Type:    "invoice.paid",
		Payload: []byte(`{}`),
	})
	must.NoError(t, err)
	must.True(t, created)

	must.NoError(t, store.Stripe().MarkEventDead(ctx, "evt_test_problem", "invalid event payload"))

	routes := mux.NewRouter()
	routes.Handle("/moderate", &moderationHTTP{
		log:   ulog.New("test"),
		store: store,
		html:  ui.Mod("mod_home.html"),
	})

	request := requestGet(t, "/moderate", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `stripe webhook events need attention`)
	must.StrContains(t, body, `evt_test_problem`)
	must.StrContains(t, body, `invalid event payload`)
}
