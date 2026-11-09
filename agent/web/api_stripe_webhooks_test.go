// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/payments"
	"github.com/shoenig/test/must"
	"github.com/stripe/stripe-go/v85"
)

func webhookEvent(kind, raw string) stripe.Event {
	return stripe.Event{
		ID:   "evt_test123",
		Type: stripe.EventType(kind),
		Data: &stripe.EventData{Raw: json.RawMessage(raw)},
	}
}

func activeSubscription(id string) *stripe.Subscription {
	return subscriptionWithStatus(id, stripe.SubscriptionStatusActive)
}

func subscriptionWithStatus(id string, status stripe.SubscriptionStatus) *stripe.Subscription {
	return &stripe.Subscription{
		ID:     id,
		Status: status,
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour).Unix()},
			},
		},
	}
}

func subscriptionWithPrice(id, priceID string) *stripe.Subscription {
	sub := subscriptionWithStatus(id, stripe.SubscriptionStatusActive)
	sub.Items.Data[0].Price = &stripe.Price{ID: priceID}
	return sub
}

func newWebhookHarness(t *testing.T, client payments.Client) (database.Store, *stripeWebhookAPI) {
	t.Helper()

	store := mockdb.Open(t)
	handler := &stripeWebhookAPI{
		log:   ulog.New("test"),
		store: store,
	}
	setWebhookClient(handler, client)
	return store, handler
}

// setWebhookClient replaces the stripe client of the handler, and the
// processor which uses it.
func setWebhookClient(handler *stripeWebhookAPI, client payments.Client) {
	handler.client = client
	handler.processor = payments.NewProcessor(handler.store.Stripe(), client)
}

func postWebhook(t *testing.T, handler *stripeWebhookAPI, body string) *httptest.ResponseRecorder {
	t.Helper()

	r, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/stripe/webhooks", strings.NewReader(body))
	must.NoError(t, err)
	r.Header.Set("Stripe-Signature", "test_sig")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestStripeWebhook_InvalidSignature(t *testing.T) {
	t.Parallel()

	_, handler := newWebhookHarness(t, &mocks.StripeClient{
		MockEventErr: errors.New("bad signature"),
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusInternalServerError, w.Code)
}

func TestStripeWebhook_UnknownEvent(t *testing.T) {
	t.Parallel()

	_, handler := newWebhookHarness(t, &mocks.StripeClient{
		MockEvent: webhookEvent("customer.created", `{}`),
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)
}

func TestStripeWebhook_EventWithoutData(t *testing.T) {
	t.Parallel()

	_, handler := newWebhookHarness(t, &mocks.StripeClient{
		MockEvent: stripe.Event{ID: "evt_test123", Type: "invoice.paid"},
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusInternalServerError, w.Code)
}

func TestStripeWebhook_CheckoutCompleted(t *testing.T) {
	t.Parallel()

	store, handler := newWebhookHarness(t, &mocks.StripeClient{})

	uid := newTestUser(t, store, "webhook-checkout")
	raw := fmt.Sprintf(`{
		"metadata": {
			"user_id": "%d",
			"price_id": "price_patreon"
		},
		"subscription": "sub_checkout"
	}`, uid)
	setWebhookClient(handler, &mocks.StripeClient{
		MockEvent:               webhookEvent("checkout.session.completed", raw),
		MockPlanFromPriceIDEnum: records.PlansPatreon,
		MockPlanFromPriceIDBool: true,
		MockSubscription:        activeSubscription("sub_checkout"),
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, usage.Plan.ID)

	// A duplicate delivery of an already processed event must be acknowledged
	// without processing it again.
	w = postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	event, gerr := store.Stripe().GetEvent(t.Context(), "evt_test123")
	must.NoError(t, gerr)
	must.Eq(t, payments.EventStatusDone, event.Status)
	must.Eq(t, 1, event.Attempts)
}

func TestStripeWebhook_SubscriptionCreated(t *testing.T) {
	t.Parallel()

	store, handler := newWebhookHarness(t, &mocks.StripeClient{})

	uid := newTestUser(t, store, "webhook-created")
	periodEnd := time.Now().Add(30 * 24 * time.Hour).Unix()
	raw := fmt.Sprintf(`{
		"id": "sub_created",
		"status": "active",
		"customer": "cus_created",
		"metadata": {"user_id": "%d"},
		"items": {"data": [{"current_period_end": %d, "price": {"id": "price_patreon"}}]}
	}`, uid, periodEnd)
	setWebhookClient(handler, &mocks.StripeClient{
		MockEvent:               webhookEvent("customer.subscription.created", raw),
		MockPlanFromPriceIDEnum: records.PlansPatreon,
		MockPlanFromPriceIDBool: true,
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, usage.Plan.ID)

	sub, serr := store.Stripe().GetSubscription(t.Context(), uid)
	must.NoError(t, serr)
	must.Eq(t, "sub_created", sub.SubscriptionID)
	must.Eq(t, "price_patreon", sub.PriceID)
	must.Eq(t, periodEnd, sub.CurrentPeriodEnd)
}

func TestStripeWebhook_SubscriptionCreated_IncompleteDoesNotGrant(t *testing.T) {
	t.Parallel()

	store, handler := newWebhookHarness(t, &mocks.StripeClient{})

	uid := newTestUser(t, store, "webhook-created-incomplete")
	periodEnd := time.Now().Add(30 * 24 * time.Hour).Unix()
	raw := fmt.Sprintf(`{
		"id": "sub_incomplete",
		"status": "incomplete",
		"customer": "cus_incomplete",
		"metadata": {"user_id": "%d"},
		"items": {"data": [{"current_period_end": %d, "price": {"id": "price_patreon"}}]}
	}`, uid, periodEnd)
	setWebhookClient(handler, &mocks.StripeClient{
		MockEvent:               webhookEvent("customer.subscription.created", raw),
		MockPlanFromPriceIDEnum: records.PlansPatreon,
		MockPlanFromPriceIDBool: true,
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansFree, usage.Plan.ID)

	sub, serr := store.Stripe().GetSubscription(t.Context(), uid)
	must.NoError(t, serr)
	must.Eq(t, "incomplete", sub.Status)
}

func TestStripeWebhook_InvoicePaid_RecoversMissingLocalRecord(t *testing.T) {
	t.Parallel()

	store, handler := newWebhookHarness(t, &mocks.StripeClient{})

	uid := newTestUser(t, store, "webhook-invoice-recover")
	sub := subscriptionWithPrice("sub_recovered", "price_patreon")
	sub.Metadata = map[string]string{"user_id": fmt.Sprintf("%d", uid)}

	setWebhookClient(handler, &mocks.StripeClient{
		MockEvent: webhookEvent("invoice.paid",
			`{"billing_reason": "subscription_cycle", "subscription": "sub_recovered"}`,
		),
		MockPlanFromPriceIDEnum: records.PlansPatreon,
		MockPlanFromPriceIDBool: true,
		MockSubscription:        sub,
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, usage.Plan.ID)

	local, serr := store.Stripe().GetSubscription(t.Context(), uid)
	must.NoError(t, serr)
	must.Eq(t, "sub_recovered", local.SubscriptionID)
}

func TestStripeWebhook_UnresolvableEventIsRecordedAndRetried(t *testing.T) {
	t.Parallel()

	store, handler := newWebhookHarness(t, &mocks.StripeClient{})
	raw := `{"id": "sub_orphan", "status": "active"}`
	setWebhookClient(handler, &mocks.StripeClient{
		MockEvent: webhookEvent("customer.subscription.updated", raw),
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusInternalServerError, w.Code)

	event, gerr := store.Stripe().GetEvent(t.Context(), "evt_test123")
	must.NoError(t, gerr)
	must.Eq(t, payments.EventStatusFailed, event.Status)
	must.Eq(t, 1, event.Attempts)
	must.True(t, strings.Contains(event.LastError, "unable to resolve user"))

	// Make the subscription resolvable via its customer, then redeliver the
	// same event. It must be processed from the inbox rather than ignored.
	uid := newTestUser(t, store, "webhook-retry")
	must.NoError(t, store.Stripe().SetCustomer(t.Context(), uid, "cus_orphan"))

	periodEnd := time.Now().Add(30 * 24 * time.Hour).Unix()
	setWebhookClient(handler, &mocks.StripeClient{
		MockEvent:               webhookEvent("customer.subscription.updated", raw),
		MockPlanFromPriceIDEnum: records.PlansPatreon,
		MockPlanFromPriceIDBool: true,
		MockSubscription: &stripe.Subscription{
			ID:       "sub_orphan",
			Status:   stripe.SubscriptionStatusActive,
			Customer: &stripe.Customer{ID: "cus_orphan"},
			Items: &stripe.SubscriptionItemList{
				Data: []*stripe.SubscriptionItem{
					{CurrentPeriodEnd: periodEnd, Price: &stripe.Price{ID: "price_patreon"}},
				},
			},
		},
	})

	w = postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, usage.Plan.ID)

	event, gerr = store.Stripe().GetEvent(t.Context(), "evt_test123")
	must.NoError(t, gerr)
	must.Eq(t, payments.EventStatusDone, event.Status)
	must.Eq(t, 2, event.Attempts)
}

func TestStripeWebhook_SubscriptionDeleted(t *testing.T) {
	t.Parallel()

	store, handler := newWebhookHarness(t, &mocks.StripeClient{})
	uid := newTestUser(t, store, "webhook-deleted")
	setPatreonPlan(t, store, uid)

	setWebhookClient(handler, &mocks.StripeClient{
		MockEvent: webhookEvent("customer.subscription.deleted",
			`{"id": "sub_test123", "status": "canceled"}`,
		),
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansFree, usage.Plan.ID)
}

func TestStripeWebhook_InvoicePaid(t *testing.T) {
	t.Parallel()

	store, handler := newWebhookHarness(t, &mocks.StripeClient{})
	uid := newTestUser(t, store, "webhook-invoice")
	setPatreonPlan(t, store, uid)

	setWebhookClient(handler, &mocks.StripeClient{
		MockEvent: webhookEvent("invoice.paid",
			`{"billing_reason": "subscription_cycle", "subscription": "sub_test123"}`,
		),
		MockPlanFromPriceIDEnum: records.PlansPatreon,
		MockPlanFromPriceIDBool: true,
		MockSubscription:        activeSubscription("sub_test123"),
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, usage.Plan.ID)

	sub, serr := store.Stripe().GetSubscription(t.Context(), uid)
	must.NoError(t, serr)
	must.Eq(t, "sub_test123", sub.SubscriptionID)
	must.Eq(t, "active", sub.Status)
	must.Positive(t, sub.CurrentPeriodEnd)
}

func TestStripeWebhook_SubscriptionUpdated_KeepsPlan(t *testing.T) {
	t.Parallel()

	store, handler := newWebhookHarness(t, &mocks.StripeClient{})
	uid := newTestUser(t, store, "webhook-updated")
	setPatreonPlan(t, store, uid)

	setWebhookClient(handler, &mocks.StripeClient{
		MockEvent: webhookEvent("customer.subscription.updated",
			`{"id": "sub_test123", "status": "past_due"}`,
		),
		MockPlanFromPriceIDEnum: records.PlansPatreon,
		MockPlanFromPriceIDBool: true,
		MockSubscription:        subscriptionWithStatus("sub_test123", stripe.SubscriptionStatusPastDue),
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, usage.Plan.ID)

	sub, serr := store.Stripe().GetSubscription(t.Context(), uid)
	must.NoError(t, serr)
	must.Eq(t, "past_due", sub.Status)
}

func TestStripeWebhook_SubscriptionUpdated_UnknownPriceKeepsPlan(t *testing.T) {
	t.Parallel()

	store, handler := newWebhookHarness(t, &mocks.StripeClient{})
	uid := newTestUser(t, store, "webhook-unknown-price")
	setPatreonPlan(t, store, uid)

	setWebhookClient(handler, &mocks.StripeClient{
		MockEvent: webhookEvent("customer.subscription.updated",
			`{"id": "sub_test123", "status": "active"}`,
		),
		MockPlanFromPriceIDEnum: 0,
		MockPlanFromPriceIDBool: false,
		MockSubscription:        activeSubscription("sub_test123"),
	})

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, usage.Plan.ID)
}
