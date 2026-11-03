// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/users"
)

type stripeCheckoutAPI struct {
	log         *ulog.Log
	store       database.Store
	client      payments.Client
	environment runtime.Environment
}

func (h *stripeCheckoutAPI) Path() string {
	return "/internal/patreon/checkout"
}

func (h *stripeCheckoutAPI) Methods() []string {
	return []string{http.MethodPost}
}

func (h *stripeCheckoutAPI) Consumer() consumer {
	return component
}

func (h *stripeCheckoutAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const long = 1 * time.Minute // give stripe a while
	ctx, cancel := scope.WithTTL(r.Context(), long)
	defer cancel()

	serve(ctx, h.log, w, r, methods{
		http.MethodPost: h.post,
	})
}

func (h *stripeCheckoutAPI) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)
	if !session.Active() {
		return redirect(w, r, "/login", http.StatusSeeOther)
	}

	priceID := h.client.PatreonProductID()

	usage, uerr := h.store.Plans().Usage(ctx, session.Identity())
	if uerr != nil {
		return abort(h.log, w, fmt.Errorf("unable to get plan usage: %w", uerr))
	}
	if usage.Plan.ID == records.PlansPatreon {
		return redirect(w, r, "/patreon", http.StatusSeeOther)
	}

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return abort(h.log, w, perr)
	}

	user := fmt.Sprintf("%d", session.Identity())
	customerID, cerr := h.client.GetOrCreateCustomer(ctx, profile.Email, user)
	if cerr != nil {
		return abort(h.log, w, fmt.Errorf("unable to get or create customer: %w", cerr))
	}
	if serr := h.store.Stripe().SetCustomer(ctx, session.Identity(), customerID); serr != nil {
		return abort(h.log, w, fmt.Errorf("unable to persist customer %s for user %s: %w", customerID, user, serr))
	}

	checkout, serr := h.client.CreateCheckoutSession(ctx, customerID, priceID, user,
		string(h.environment.Canonical("/patreon?success=true")),
		string(h.environment.Canonical("/patreon?canceled=true")),
	)
	if serr != nil {
		return abort(h.log, w, fmt.Errorf("unable to create checkout session: %w", serr))
	}
	if checkout.URL == "" {
		return abort(h.log, w, errors.New("checkout session has no redirect url"))
	}

	return redirect(w, r, checkout.URL, http.StatusSeeOther)
}
