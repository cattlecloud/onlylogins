package payments

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/stripe/stripe-go/v85"
)

// ProcessorStore is the portion of the database needed to apply stripe
// events to the local subscription state.
type ProcessorStore interface {
	GetUserByCustomer(scope.C, string) (users.Identity, error)
	GetUserBySubscription(scope.C, string) (users.Identity, error)
	GetSubscription(scope.C, users.Identity) (*StripeSubscription, error)
	CurrentPlan(scope.C, users.Identity) (records.PlansEnum, error)
	UpsertSubscription(scope.C, *UpdateStripeSubscription) error
}

// Processor applies stripe webhook events to the local subscription state.
//
// Processing is independent of delivery order; the user is resolved from the
// stripe objects themselves, and the current state of a subscription is always
// read from the stripe API. This makes the processor safe to run again for an
// event which was delivered (or attempted) more than once.
type Processor struct {
	log    *ulog.Log
	store  ProcessorStore
	client Client
}

// NewProcessor creates a Processor for applying stripe events.
func NewProcessor(store ProcessorStore, client Client) *Processor {
	return &Processor{
		log:    ulog.New("stripe-proc"),
		store:  store,
		client: client,
	}
}

// Dispatch applies a single verified stripe event to the local state.
//
// Events which are not relevant to subscriptions are ignored. Any error
// returned indicates the event could not be applied, and it is safe to attempt
// the same event again later.
func (p *Processor) Dispatch(ctx scope.C, event stripe.Event) error {
	switch event.Type {
	case "checkout.session.completed":
		return p.handleCheckoutCompleted(ctx, event)

	case "customer.subscription.created", "customer.subscription.updated":
		return p.handleSubscriptionUpsert(ctx, event)

	case "customer.subscription.deleted":
		return p.handleSubscriptionDeleted(ctx, event)

	case "invoice.paid":
		return p.handleInvoicePaid(ctx, event)

	default:
		p.log.D.Fmt("skipping webhook: type=%s id=%s", event.Type, event.ID)
		return nil
	}
}

func (p *Processor) handleCheckoutCompleted(ctx scope.C, event stripe.Event) error {
	p.log.I.Fmt("got webhook: type=%s id=%s", event.Type, event.ID)

	raw, derr := eventData(event)
	if derr != nil {
		return derr
	}

	var session stripe.CheckoutSession
	if err := json.Unmarshal(raw, &session); err != nil {
		return fmt.Errorf("failed to parse checkout session: %w, event: %s", err, event.ID)
	}

	if session.Subscription == nil || session.Subscription.ID == "" {
		return fmt.Errorf("failed checkout: missing subscription id, event: %s", event.ID)
	}
	subscriptionID := session.Subscription.ID

	sub, ferr := p.client.FetchSubscription(ctx, subscriptionID)
	if ferr != nil {
		return fmt.Errorf("failed checkout: unable to fetch subscription: %w", ferr)
	}
	if sub == nil {
		return fmt.Errorf("failed checkout: subscription %s not found, event: %s", subscriptionID, event.ID)
	}

	userID, uerr := p.resolveCheckoutUser(ctx, &session, sub)
	if uerr != nil {
		return fmt.Errorf("failed checkout: %w, event: %s", uerr, event.ID)
	}

	priceID := subscriptionPriceID(sub)
	if priceID == "" {
		priceID = session.Metadata["price_id"]
	}
	if priceID == "" {
		priceID = extractPriceID(&session)
	}
	if priceID == "" {
		return fmt.Errorf("failed checkout: missing price id, user: %d, event: %s", userID, event.ID)
	}

	planID, ok := p.client.PlanFromPriceID(priceID)
	if !ok {
		return fmt.Errorf("failed checkout: no such local plan for price, user: %d, event: %s, id %q", userID, event.ID, priceID)
	}

	p.log.I.Fmt("confirming checkout: user=%d subscription=%s price=%s", userID, subscriptionID, priceID)

	update := &UpdateStripeSubscription{
		User: userID,
		Plan: planID,
		Sub: &StripeSubscription{
			SubscriptionID:   subscriptionID,
			PriceID:          priceID,
			Status:           string(sub.Status),
			CurrentPeriodEnd: currentPeriodEnd(sub),
		},
	}

	if err := update.Validate(); err != nil {
		return fmt.Errorf("failed checkout: stripe subscription validation: %w", err)
	}

	if err := p.store.UpsertSubscription(ctx, update); err != nil {
		return fmt.Errorf("failed checkout: unable to upsert subscription: %w", err)
	}

	p.log.I.Fmt("updated subscription %s for user %d", subscriptionID, userID)
	return nil
}

// handleSubscriptionUpsert handles both customer.subscription.created and
// customer.subscription.updated. Handling created independently of
// checkout.session.completed means a lost checkout event can still result in
// the local subscription being created.
func (p *Processor) handleSubscriptionUpsert(ctx scope.C, event stripe.Event) error {
	raw, derr := eventData(event)
	if derr != nil {
		return derr
	}

	var eventSub stripe.Subscription
	if err := json.Unmarshal(raw, &eventSub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w, event: %s", err, event.ID)
	}

	if eventSub.ID == "" {
		return fmt.Errorf("subscription event missing id: event=%s", event.ID)
	}

	fetched, ferr := p.client.FetchSubscription(ctx, eventSub.ID)
	if ferr != nil {
		return fmt.Errorf("failed to fetch subscription %s: %w", eventSub.ID, ferr)
	}

	sub := fetched
	if sub == nil {
		sub = &eventSub
	}

	p.log.I.Fmt("got webhook: type=%s id=%s sub=%s status=%s",
		event.Type, event.ID, sub.ID, sub.Status)

	userID, uerr := p.resolveSubscriptionUser(ctx, sub, &eventSub)
	if uerr != nil {
		return fmt.Errorf("subscription %s: %w, event: %s", eventSub.ID, uerr, event.ID)
	}

	existing, _ := p.store.GetSubscription(ctx, userID)

	priceID := subscriptionPriceID(sub)
	if priceID == "" {
		priceID = subscriptionPriceID(&eventSub)
	}
	if priceID == "" && existing != nil {
		priceID = existing.PriceID
	}
	if priceID == "" {
		return fmt.Errorf("subscription %s: missing price id, user: %d, event: %s", eventSub.ID, userID, event.ID)
	}

	planID, perr := p.planFor(ctx, userID, priceID, subscriptionGrantsPlan(sub.Status))
	if perr != nil {
		return perr
	}

	update := &UpdateStripeSubscription{
		User: userID,
		Plan: planID,
		Sub: &StripeSubscription{
			SubscriptionID:   sub.ID,
			PriceID:          priceID,
			Status:           string(sub.Status),
			CurrentPeriodEnd: currentPeriodEnd(sub),
		},
	}

	if err := update.Validate(); err != nil {
		return fmt.Errorf("subscription update validation: %w", err)
	}

	if err := p.store.UpsertSubscription(ctx, update); err != nil {
		return fmt.Errorf("unable to upsert subscription: %w", err)
	}

	p.log.I.Fmt("updated subscription %s for user %d (status=%s)", sub.ID, userID, sub.Status)
	return nil
}

func (p *Processor) handleSubscriptionDeleted(ctx scope.C, event stripe.Event) error {
	raw, derr := eventData(event)
	if derr != nil {
		return derr
	}

	var eventSub stripe.Subscription
	if err := json.Unmarshal(raw, &eventSub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w, event: %s", err, event.ID)
	}

	if eventSub.ID == "" {
		return fmt.Errorf("subscription event missing id: event=%s", event.ID)
	}

	fetched, ferr := p.client.FetchSubscription(ctx, eventSub.ID)
	if ferr != nil {
		// A deleted subscription should still be retrievable, but the event
		// itself carries everything we need, so do not fail the downgrade.
		p.log.W.Fmt("unable to fetch deleted subscription %s: %v", eventSub.ID, ferr)
	}

	sub := fetched
	if sub == nil {
		sub = &eventSub
	}

	p.log.I.Fmt("got webhook: type=%s id=%s sub=%s status=%s",
		event.Type, event.ID, sub.ID, sub.Status)

	userID, uerr := p.resolveSubscriptionUser(ctx, sub, &eventSub)
	if uerr != nil {
		return fmt.Errorf("subscription %s: %w, event: %s", eventSub.ID, uerr, event.ID)
	}

	existing, _ := p.store.GetSubscription(ctx, userID)

	priceID := subscriptionPriceID(sub)
	if priceID == "" {
		priceID = subscriptionPriceID(&eventSub)
	}
	if priceID == "" && existing != nil {
		priceID = existing.PriceID
	}
	if priceID == "" {
		return fmt.Errorf("subscription %s: missing price id, user: %d, event: %s", eventSub.ID, userID, event.ID)
	}

	update := &UpdateStripeSubscription{
		User: userID,
		Plan: records.PlansFree,
		Sub: &StripeSubscription{
			SubscriptionID:   eventSub.ID,
			PriceID:          priceID,
			Status:           "canceled",
			CurrentPeriodEnd: 0,
		},
	}

	if err := update.Validate(); err != nil {
		return fmt.Errorf("subscription delete validation: %w", err)
	}

	if err := p.store.UpsertSubscription(ctx, update); err != nil {
		return fmt.Errorf("unable to downgrade subscription: %w", err)
	}

	p.log.I.Fmt("downgraded user %d to free plan (subscription %s deleted)", userID, eventSub.ID)
	return nil
}

func (p *Processor) handleInvoicePaid(ctx scope.C, event stripe.Event) error {
	raw, derr := eventData(event)
	if derr != nil {
		return derr
	}

	var invoice struct {
		BillingReason string `json:"billing_reason"`
		Subscription  string `json:"subscription"`
	}
	if err := json.Unmarshal(raw, &invoice); err != nil {
		return fmt.Errorf("failed to parse invoice: %w, event: %s", err, event.ID)
	}

	if invoice.BillingReason != "subscription_create" &&
		invoice.BillingReason != "subscription_cycle" {
		return nil
	}

	if invoice.Subscription == "" {
		p.log.W.Fmt("invoice paid without subscription: event=%s", event.ID)
		return nil
	}

	subID := invoice.Subscription

	fetched, ferr := p.client.FetchSubscription(ctx, subID)
	if ferr != nil {
		return fmt.Errorf("failed to fetch subscription: %w", ferr)
	}
	if fetched == nil {
		return fmt.Errorf("invoice paid: subscription %s not found, event: %s", subID, event.ID)
	}

	p.log.I.Fmt("got webhook: type=%s id=%s sub=%s reason=%s",
		event.Type, event.ID, subID, invoice.BillingReason)

	userID, uerr := p.resolveSubscriptionUser(ctx, fetched)
	if uerr != nil {
		return fmt.Errorf("invoice paid: %w, event: %s", uerr, event.ID)
	}

	existing, _ := p.store.GetSubscription(ctx, userID)

	priceID := subscriptionPriceID(fetched)
	if priceID == "" && existing != nil {
		priceID = existing.PriceID
	}
	if priceID == "" {
		return fmt.Errorf("invoice paid: missing price id, user: %d, event: %s", userID, event.ID)
	}

	// the invoice was paid, so the user is entitled to the plan
	planID, perr := p.planFor(ctx, userID, priceID, true)
	if perr != nil {
		return perr
	}

	update := &UpdateStripeSubscription{
		User: userID,
		Plan: planID,
		Sub: &StripeSubscription{
			SubscriptionID:   subID,
			PriceID:          priceID,
			Status:           string(fetched.Status),
			CurrentPeriodEnd: currentPeriodEnd(fetched),
		},
	}

	if err := update.Validate(); err != nil {
		return fmt.Errorf("invoice paid validation: %w", err)
	}

	if err := p.store.UpsertSubscription(ctx, update); err != nil {
		return fmt.Errorf("unable to extend subscription: %w", err)
	}

	p.log.I.Fmt("extended subscription %s for user %d (period_end=%d)", subID, userID, currentPeriodEnd(fetched))
	return nil
}

// resolveCheckoutUser determines the user for a checkout session, preferring
// the metadata set when the session was created.
func (p *Processor) resolveCheckoutUser(
	ctx scope.C,
	session *stripe.CheckoutSession,
	sub *stripe.Subscription,
) (users.Identity, error) {
	if userID, ok := metadataUserID(session.Metadata); ok {
		return userID, nil
	}
	return p.resolveSubscriptionUser(ctx, sub)
}

// resolveSubscriptionUser determines the local user for a subscription without
// requiring the local subscription row to already exist. This is what makes
// event handling independent of delivery order.
func (p *Processor) resolveSubscriptionUser(ctx scope.C, subscriptions ...*stripe.Subscription) (users.Identity, error) {
	for _, sub := range subscriptions {
		if sub == nil {
			continue
		}
		if userID, ok := metadataUserID(sub.Metadata); ok {
			return userID, nil
		}
	}

	for _, sub := range subscriptions {
		if sub == nil || sub.Customer == nil || sub.Customer.ID == "" {
			continue
		}
		userID, err := p.store.GetUserByCustomer(ctx, sub.Customer.ID)
		if err == nil {
			return userID, nil
		}
	}

	for _, sub := range subscriptions {
		if sub == nil || sub.ID == "" {
			continue
		}
		userID, err := p.store.GetUserBySubscription(ctx, sub.ID)
		if err == nil {
			return userID, nil
		}
	}

	return 0, errors.New("unable to resolve user for stripe subscription")
}

// planFor returns the local plan for the given stripe price.
//
// When the user is entitled to the paid plan, the price determines the plan.
// Otherwise the current plan of the user is retained, and the existing
// subscription maintenance handles the eventual downgrade.
func (p *Processor) planFor(
	ctx scope.C,
	userID users.Identity,
	priceID string,
	entitled bool,
) (records.PlansEnum, error) {
	if entitled {
		if planID, ok := p.client.PlanFromPriceID(priceID); ok {
			return planID, nil
		}
		p.log.E.Fmt("unrecognized price_id %q for user %d, keeping current plan", priceID, userID)
	} else {
		p.log.D.Fmt("subscription is not entitled, keeping current plan for user %d", userID)
	}

	planID, err := p.store.CurrentPlan(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("unable to get current plan for user %d: %w", userID, err)
	}
	return planID, nil
}

// eventData returns the raw object of an event, which carries the stripe
// resource the event is about.
func eventData(event stripe.Event) (json.RawMessage, error) {
	if event.Data == nil {
		return nil, fmt.Errorf("event is missing data: id=%s type=%s", event.ID, event.Type)
	}
	return event.Data.Raw, nil
}

func metadataUserID(metadata map[string]string) (users.Identity, bool) {
	raw, ok := metadata["user_id"]
	if !ok || raw == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return users.Identity(id), true
}

// subscriptionGrantsPlan reports whether a subscription status is entitled to
// the paid plan. Incomplete subscriptions (for example, while an asynchronous
// payment is still settling) must not grant access yet.
func subscriptionGrantsPlan(status stripe.SubscriptionStatus) bool {
	switch status {
	case stripe.SubscriptionStatusActive,
		stripe.SubscriptionStatusTrialing,
		stripe.SubscriptionStatusPastDue:
		return true
	default:
		return false
	}
}

func subscriptionPriceID(sub *stripe.Subscription) string {
	if sub == nil || sub.Items == nil || len(sub.Items.Data) == 0 {
		return ""
	}
	item := sub.Items.Data[0]
	if item == nil || item.Price == nil {
		return ""
	}
	return item.Price.ID
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
	if sub == nil || sub.Items == nil {
		return 0
	}
	if len(sub.Items.Data) == 0 {
		return 0
	}
	return sub.Items.Data[0].CurrentPeriodEnd
}
