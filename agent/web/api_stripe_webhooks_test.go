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
	return &stripe.Subscription{
		ID:     id,
		Status: stripe.SubscriptionStatusActive,
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour).Unix()},
			},
		},
	}
}

func newWebhookHarness(t *testing.T, client payments.Client) (database.Store, *stripeWebhookAPI) {
	t.Helper()

	store := mockdb.Open(t)
	handler := &stripeWebhookAPI{
		log:    ulog.New("test"),
		store:  store,
		client: client,
	}
	return store, handler
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
	handler.client = &mocks.StripeClient{
		MockEvent:               webhookEvent("checkout.session.completed", raw),
		MockPlanFromPriceIDEnum: records.PlansPatreon,
		MockPlanFromPriceIDBool: true,
		MockSubscription:        activeSubscription("sub_checkout"),
	}

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, usage.Plan.ID)
}

func TestStripeWebhook_SubscriptionDeleted(t *testing.T) {
	t.Parallel()

	store, handler := newWebhookHarness(t, &mocks.StripeClient{})
	uid := newTestUser(t, store, "webhook-deleted")
	setPatreonPlan(t, store, uid)

	handler.client = &mocks.StripeClient{
		MockEvent: webhookEvent("customer.subscription.deleted",
			`{"id": "sub_test123", "status": "canceled"}`,
		),
	}

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

	handler.client = &mocks.StripeClient{
		MockEvent: webhookEvent("invoice.paid",
			`{"billing_reason": "subscription_cycle", "subscription": "sub_test123"}`,
		),
		MockPlanFromPriceIDEnum: records.PlansPatreon,
		MockPlanFromPriceIDBool: true,
		MockSubscription:        activeSubscription("sub_test123"),
	}

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

	handler.client = &mocks.StripeClient{
		MockEvent: webhookEvent("customer.subscription.updated",
			`{"id": "sub_test123", "status": "past_due"}`,
		),
		MockPlanFromPriceIDEnum: records.PlansPatreon,
		MockPlanFromPriceIDBool: true,
		MockSubscription:        activeSubscription("sub_test123"),
	}

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

	handler.client = &mocks.StripeClient{
		MockEvent: webhookEvent("customer.subscription.updated",
			`{"id": "sub_test123", "status": "active"}`,
		),
		MockPlanFromPriceIDEnum: 0,
		MockPlanFromPriceIDBool: false,
		MockSubscription:        activeSubscription("sub_test123"),
	}

	w := postWebhook(t, handler, "{}")
	must.Eq(t, http.StatusOK, w.Code)

	usage, err := store.Plans().Usage(t.Context(), uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, usage.Plan.ID)
}
