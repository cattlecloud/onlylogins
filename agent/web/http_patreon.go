// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/agent/web/pages"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type patreonContent struct {
	Free    *sitecontent.Plan
	Patreon *sitecontent.Plan
	Usage   *sitecontent.Usage

	IsPatreon   bool
	Enabled     bool
	Success     bool
	Canceled    bool
	PeriodLabel string
	PeriodDate  string
}

type patreonHTTP struct {
	log         *ulog.Log
	environment runtime.Environment
	store       database.Store
	payments    payments.Client
	html        *template.Template
}

func (h *patreonHTTP) Path() string {
	return "/patreon"
}

func (h *patreonHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *patreonHTTP) Metric() string {
	return "patreon"
}

func (h *patreonHTTP) Consumer() consumer {
	return normal
}

func (h *patreonHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *patreonHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return perr
	}

	plans, lerr := h.store.Plans().List(ctx)
	if lerr != nil {
		return fmt.Errorf("unable to list plans: %w", lerr)
	}

	content := &patreonContent{
		Free:     findPlan(plans, records.PlansFree),
		Patreon:  findPlan(plans, records.PlansPatreon),
		Enabled:  h.enabled(),
		Success:  r.URL.Query().Get("success") == "true",
		Canceled: r.URL.Query().Get("canceled") == "true",
	}

	if content.Free == nil || content.Patreon == nil {
		return errors.New("unable to load plan definitions")
	}

	if session.Active() {
		usage, uerr := h.store.Plans().Usage(ctx, session.Identity())
		if uerr != nil {
			return fmt.Errorf("unable to get plan usage: %w", uerr)
		}
		content.Usage = usage
		content.IsPatreon = usage.Plan.ID == records.PlansPatreon

		if content.IsPatreon {
			h.getSubscription(ctx, session.Identity(), content)
		}
	}

	head := &pages.Head{
		Title:       format.PageTitle("Patreon"),
		Styles:      []string{"patreon.css"},
		Canonical:   h.environment.Canonical("%s", h.Path()),
		Description: h.description(),
		Profile:     profile,
	}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, content, nil).Execute(w, session)
}

func (h *patreonHTTP) enabled() bool {
	return h.payments != nil && h.payments.PatreonProductID() != ""
}

func (h *patreonHTTP) getSubscription(ctx scope.C, uid users.Identity, content *patreonContent) {
	sub, err := h.store.Stripe().GetSubscription(ctx, uid)
	if err != nil {
		h.log.D.Fmt("no stripe subscription found for user %d: %v", uid, err)
		return
	}

	if sub.CurrentPeriodEnd > 0 {
		content.PeriodDate = format.DateTime(time.Unix(sub.CurrentPeriodEnd, 0))
		switch sub.Status {
		case "canceled", "unpaid":
			content.PeriodLabel = "Ends"
		default:
			content.PeriodLabel = "Renews"
		}
	}
}

func findPlan(plans []*sitecontent.Plan, id records.PlansEnum) *sitecontent.Plan {
	for _, plan := range plans {
		if plan.ID == id {
			return plan
		}
	}
	return nil
}

func (h *patreonHTTP) description() string {
	const text = `
	Support OnlyLogins on Patreon and store thousands of login credentials,
	passwords, notes, and other secrets in your encrypted vault.`
	return format.PageDescription(text)
}
