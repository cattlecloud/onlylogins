// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/stripe/stripe-go/v85"
)

type stripeWebhookAPI struct {
	log    *ulog.Log
	store  database.Store
	client payments.Client
}

func (h *stripeWebhookAPI) Path() string {
	return "/api/stripe/webhooks"
}

func (h *stripeWebhookAPI) Methods() []string {
	return []string{http.MethodPost}
}

func (h *stripeWebhookAPI) Consumer() consumer {
	return robotic
}

func (h *stripeWebhookAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const long = 1 * time.Minute // give stripe extra time
	ctx, cancel := scope.WithTTL(r.Context(), long)
	defer cancel()

	serve(ctx, h.log, w, r, methods{
		http.MethodPost: h.post,
	})
}

func (h *stripeWebhookAPI) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	payload, perr := io.ReadAll(r.Body)
	if perr != nil {
		return abort(h.log, w, fmt.Errorf("failed to read webhook body: %w", perr))
	}

	sig := r.Header.Get("Stripe-Signature")

	event, werr := h.client.ConstructWebhookEvent(payload, sig)
	if werr != nil {
		return abort(h.log, w, fmt.Errorf("unable to construct webhook event: %w", werr))
	}

	switch event.Type {
	case "checkout.session.completed":
		if err := h.handleCheckoutCompleted(ctx, event); err != nil {
			return abort(h.log, w, err)
		}

	case "customer.subscription.updated":
		if err := h.handleSubscriptionUpdated(ctx, event); err != nil {
			return abort(h.log, w, err)
		}

	case "customer.subscription.deleted":
		if err := h.handleSubscriptionDeleted(ctx, event); err != nil {
			return abort(h.log, w, err)
		}

	case "invoice.paid":
		if err := h.handleInvoicePaid(ctx, event); err != nil {
			return abort(h.log, w, err)
		}

	default:
		h.log.D.Fmt("skipping webhook: type=%s id=%s", event.Type, event.ID)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func (h *stripeWebhookAPI) handleCheckoutCompleted(ctx scope.C, event stripe.Event) error {
	h.log.I.Fmt("got webhook: type=%s id=%s", event.Type, event.ID)

	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return fmt.Errorf("failed to parse checkout session: %w, event: %s", err, event.ID)
	}

	userIDStr, ok := session.Metadata["user_id"]
	if !ok {
		return fmt.Errorf("failed checkout: missing user_id in metadata, event: %s", event.ID)
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed checkout: invalid user_id %q: %w", userIDStr, err)
	}

	if session.Subscription == nil || session.Subscription.ID == "" {
		return fmt.Errorf("failed checkout: missing subscription id, user: %d, event: %s", userID, event.ID)
	}
	subscriptionID := session.Subscription.ID

	priceID := session.Metadata["price_id"]
	if priceID == "" {
		priceID = extractPriceID(&session)
	}
	if priceID == "" {
		return fmt.Errorf("failed checkout: missing price id, user: %d, event: %s", userID, event.ID)
	}

	h.log.I.Fmt("confirming checkout: user=%d subscription=%s price=%s", userID, subscriptionID, priceID)

	sub, ferr := h.client.FetchSubscription(ctx, subscriptionID)
	if ferr != nil {
		return fmt.Errorf("failed checkout: unable to fetch subscription: %w", ferr)
	}

	planID, ok := h.client.PlanFromPriceID(priceID)
	if !ok {
		return fmt.Errorf("failed checkout: no such local plan for price, user: %d, event: %s, id %q", userID, event.ID, priceID)
	}

	update := &payments.UpdateStripeSubscription{
		User: users.Identity(userID),
		Plan: planID,
		Sub: &payments.StripeSubscription{
			SubscriptionID:   subscriptionID,
			PriceID:          priceID,
			Status:           string(sub.Status),
			CurrentPeriodEnd: currentPeriodEnd(sub),
		},
	}

	if err := update.Validate(); err != nil {
		return fmt.Errorf("failed checkout: stripe subscription validation: %w", err)
	}

	if err := h.store.Stripe().UpsertSubscription(ctx, update); err != nil {
		return fmt.Errorf("failed checkout: unable to upsert subscription: %w", err)
	}

	h.log.I.Fmt("updated subscription %s for user %d", subscriptionID, userID)
	return nil
}

func extractPriceID(session *stripe.CheckoutSession) string {
	if session.LineItems == nil {
		return ""
	}
	if len(session.LineItems.Data) == 0 {
		return ""
	}
	line := session.LineItems.Data[0]
	if line.Price == nil {
		return ""
	}
	return line.Price.ID
}

func currentPeriodEnd(sub *stripe.Subscription) int64 {
	if sub.Items == nil {
		return 0
	}
	if len(sub.Items.Data) == 0 {
		return 0
	}
	return sub.Items.Data[0].CurrentPeriodEnd
}

func (h *stripeWebhookAPI) handleSubscriptionUpdated(ctx scope.C, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w, event: %s", err, event.ID)
	}

	h.log.I.Fmt("got webhook: type=%s id=%s sub=%s status=%s",
		event.Type, event.ID, sub.ID, sub.Status)

	userID, uerr := h.store.Stripe().GetUserBySubscription(ctx, sub.ID)
	if uerr != nil {
		h.log.W.Fmt("unrecognized subscription: id=%s event=%s", sub.ID, event.ID)
		return nil
	}

	existingSub, _ := h.store.Stripe().GetSubscription(ctx, userID)
	if existingSub == nil {
		h.log.W.Fmt("subscription updated with no local record: user=%d sub=%s", userID, sub.ID)
		return nil
	}

	fetched, ferr := h.client.FetchSubscription(ctx, sub.ID)
	if ferr != nil {
		return fmt.Errorf("failed to fetch subscription: %w", ferr)
	}

	planID, perr := h.planFor(ctx, userID, existingSub.PriceID)
	if perr != nil {
		return perr
	}

	update := &payments.UpdateStripeSubscription{
		User: userID,
		Plan: planID,
		Sub: &payments.StripeSubscription{
			SubscriptionID:   sub.ID,
			PriceID:          existingSub.PriceID,
			Status:           string(sub.Status),
			CurrentPeriodEnd: currentPeriodEnd(fetched),
		},
	}

	if err := update.Validate(); err != nil {
		return fmt.Errorf("subscription update validation: %w", err)
	}

	if err := h.store.Stripe().UpsertSubscription(ctx, update); err != nil {
		return fmt.Errorf("unable to upsert subscription: %w", err)
	}

	h.log.I.Fmt("updated subscription %s for user %d (status=%s)", sub.ID, userID, sub.Status)
	return nil
}

func (h *stripeWebhookAPI) handleSubscriptionDeleted(ctx scope.C, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w, event: %s", err, event.ID)
	}

	h.log.I.Fmt("got webhook: type=%s id=%s sub=%s status=%s",
		event.Type, event.ID, sub.ID, sub.Status)

	userID, uerr := h.store.Stripe().GetUserBySubscription(ctx, sub.ID)
	if uerr != nil {
		h.log.W.Fmt("unrecognized subscription: id=%s event=%s", sub.ID, event.ID)
		return nil
	}

	existingSub, _ := h.store.Stripe().GetSubscription(ctx, userID)
	if existingSub == nil {
		h.log.W.Fmt("subscription deleted with no local record: user=%d sub=%s", userID, sub.ID)
		return nil
	}

	update := &payments.UpdateStripeSubscription{
		User: userID,
		Plan: records.PlansFree,
		Sub: &payments.StripeSubscription{
			SubscriptionID:   sub.ID,
			PriceID:          existingSub.PriceID,
			Status:           "canceled",
			CurrentPeriodEnd: 0,
		},
	}

	if err := update.Validate(); err != nil {
		return fmt.Errorf("subscription delete validation: %w", err)
	}

	if err := h.store.Stripe().UpsertSubscription(ctx, update); err != nil {
		return fmt.Errorf("unable to downgrade subscription: %w", err)
	}

	h.log.I.Fmt("downgraded user %d to free plan (subscription %s deleted)", userID, sub.ID)
	return nil
}

func (h *stripeWebhookAPI) handleInvoicePaid(ctx scope.C, event stripe.Event) error {
	var raw struct {
		BillingReason string `json:"billing_reason"`
		Subscription  string `json:"subscription"`
	}
	if err := json.Unmarshal(event.Data.Raw, &raw); err != nil {
		return fmt.Errorf("failed to parse invoice: %w, event: %s", err, event.ID)
	}

	if raw.BillingReason != "subscription_create" &&
		raw.BillingReason != "subscription_cycle" {
		return nil
	}

	if raw.Subscription == "" {
		h.log.W.Fmt("invoice paid without subscription: event=%s", event.ID)
		return nil
	}

	subID := raw.Subscription
	h.log.I.Fmt("got webhook: type=%s id=%s sub=%s reason=%s",
		event.Type, event.ID, subID, raw.BillingReason)

	userID, uerr := h.store.Stripe().GetUserBySubscription(ctx, subID)
	if uerr != nil {
		h.log.W.Fmt("unrecognized subscription on invoice: sub=%s event=%s", subID, event.ID)
		return nil
	}

	existingSub, _ := h.store.Stripe().GetSubscription(ctx, userID)
	if existingSub == nil {
		h.log.W.Fmt("invoice paid with no local record: user=%d sub=%s", userID, subID)
		return nil
	}

	fetched, ferr := h.client.FetchSubscription(ctx, subID)
	if ferr != nil {
		return fmt.Errorf("failed to fetch subscription: %w", ferr)
	}

	planID, perr := h.planFor(ctx, userID, existingSub.PriceID)
	if perr != nil {
		return perr
	}

	update := &payments.UpdateStripeSubscription{
		User: userID,
		Plan: planID,
		Sub: &payments.StripeSubscription{
			SubscriptionID:   subID,
			PriceID:          existingSub.PriceID,
			Status:           string(fetched.Status),
			CurrentPeriodEnd: currentPeriodEnd(fetched),
		},
	}

	if err := update.Validate(); err != nil {
		return fmt.Errorf("invoice paid validation: %w", err)
	}

	if err := h.store.Stripe().UpsertSubscription(ctx, update); err != nil {
		return fmt.Errorf("unable to extend subscription: %w", err)
	}

	h.log.I.Fmt("extended subscription %s for user %d (period_end=%d)", subID, userID, currentPeriodEnd(fetched))
	return nil
}

// planFor returns the local plan matching the stripe price. When the price is
// not recognized, the user's current plan is retained.
func (h *stripeWebhookAPI) planFor(ctx scope.C, userID users.Identity, priceID string) (records.PlansEnum, error) {
	if planID, ok := h.client.PlanFromPriceID(priceID); ok {
		return planID, nil
	}

	h.log.E.Fmt("unrecognized price_id %q for user %d, keeping current plan", priceID, userID)
	usage, err := h.store.Plans().Usage(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("unable to get current plan for user %d: %w", userID, err)
	}
	return usage.Plan.ID, nil
}
