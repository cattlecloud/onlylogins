// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"html/template"
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/payments"
)

const (
	// stripeProblemAge is how long to wait before a failing stripe event is
	// failing before reporting it as a problem
	stripeProblemAge = 1 * time.Hour

	// stripeProblemLimit is the maximum number of stripe events shown
	stripeProblemLimit = 20
)

type moderationContent struct {
	// UsersCount is the total number of users registered
	UsersCount int

	// StripeProblems are the webhook events which could not be applied, and
	// which need attention
	StripeProblems []*payments.StripeEvent
}

type moderationHTTP struct {
	log   *ulog.Log
	store database.Store
	html  *template.Template
}

func (h *moderationHTTP) Path() string {
	return "/moderate"
}

func (h *moderationHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *moderationHTTP) Consumer() consumer {
	return moderate
}

func (h *moderationHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *moderationHTTP) get(ctx scope.C, w http.ResponseWriter, _ *http.Request) error {
	// count the users
	usersCount, ucerr := h.store.Users().Count(ctx)
	if ucerr != nil {
		return abort(h.log, w, ucerr)
	}

	// lookup stripe problems
	problems, perr := h.store.Stripe().ProblemEvents(
		ctx,
		time.Now().Add(-stripeProblemAge),
		stripeProblemLimit,
	)
	if perr != nil {
		return abort(h.log, w, perr)
	}

	content := &moderationContent{
		UsersCount:     usersCount,
		StripeProblems: problems,
	}

	if err := h.html.ExecuteTemplate(w, "layout", content); err != nil {
		h.log.E.Fmt("unable to render moderation page: %v", err)
	}

	return nil
}
