// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

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

type stripePortalAPI struct {
	log         *ulog.Log
	store       database.Store
	client      payments.Client
	environment runtime.Environment
}

func (h *stripePortalAPI) Path() string {
	return "/internal/patreon/portal"
}

func (h *stripePortalAPI) Methods() []string {
	return []string{http.MethodPost}
}

func (h *stripePortalAPI) Consumer() consumer {
	return component
}

func (h *stripePortalAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const long = 1 * time.Minute // give stripe a while
	ctx, cancel := scope.WithTTL(r.Context(), long)
	defer cancel()

	serve(ctx, h.log, w, r, methods{
		http.MethodPost: h.post,
	})
}

func (h *stripePortalAPI) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)
	if !session.Active() {
		return redirect(w, r, "/login", http.StatusSeeOther)
	}

	usage, uerr := h.store.Plans().Usage(ctx, session.Identity())
	if uerr != nil {
		return abort(h.log, w, fmt.Errorf("unable to get plan usage: %w", uerr))
	}
	if usage.Plan.ID != records.PlansPatreon {
		return redirect(w, r, "/patreon", http.StatusSeeOther)
	}

	customerID, found, cerr := h.store.Stripe().GetCustomer(ctx, session.Identity())
	if cerr != nil {
		return abort(h.log, w, fmt.Errorf("unable to look up stripe customer: %w", cerr))
	}
	if !found {
		return abort(h.log, w, errors.New("no stripe customer found"))
	}

	portal, perr := h.client.CreateBillingPortalSession(ctx, customerID, string(h.environment.Canonical("/patreon")))
	if perr != nil {
		return abort(h.log, w, fmt.Errorf("unable to create billing portal session: %w", perr))
	}
	if portal.URL == "" {
		return abort(h.log, w, errors.New("billing portal session has no redirect url"))
	}

	return redirect(w, r, portal.URL, http.StatusSeeOther)
}
