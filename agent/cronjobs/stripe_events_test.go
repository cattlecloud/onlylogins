// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package cronjobs

import (
	"fmt"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/crons"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
	"github.com/stripe/stripe-go/v85"
)

// stripeTestTimeout is long enough for any reasonable testcase to complete;
// the creation of users burns a lot of CPU due to argon.
const stripeTestTimeout = 1 * time.Minute

func TestRetryStripeEvents_RetriesFailedEvent(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	ctx, cancel := scope.TTL(stripeTestTimeout)
	defer cancel()

	uid := insertStripeUser(t, ctx, store, "stripe-retry")

	payload := []byte(`{
		"id": "evt_test_retry",
		"type": "invoice.paid",
		"data": {
			"object": {
				"billing_reason": "subscription_cycle",
				"subscription": "sub_retry"
			}
		}
	}`)

	record(t, ctx, store, &payments.StripeEvent{
		ID:      "evt_test_retry",
		Type:    "invoice.paid",
		Payload: payload,
	})
	must.NoError(t, store.Stripe().MarkEventFailed(ctx, "evt_test_retry", "stripe is down"))

	sub := &stripe.Subscription{
		ID:     "sub_retry",
		Status: stripe.SubscriptionStatusActive,
		Metadata: map[string]string{
			payments.MetadataUserIDKey: fmt.Sprintf("%d", uid),
		},
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour).Unix(),
					Price:            &stripe.Price{ID: "price_patreon"},
				},
			},
		},
	}

	job := NewRetryStripeEvents(
		&crons.Config{Expression: "* * * * *"},
		store,
		&mocks.StripeClient{
			MockPlanFromPriceIDEnum: records.PlansPatreon,
			MockPlanFromPriceIDBool: true,
			MockSubscription:        sub,
		},
	)

	result := job.Start()
	must.NotNil(t, result)
	must.Eq(t, crons.ExitSuccess, result.Code)

	event, err := store.Stripe().GetEvent(ctx, "evt_test_retry")
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusDone, event.Status)
	must.Eq(t, "", event.LastError)

	usage, err := store.Plans().Usage(ctx, uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, usage.Plan.ID)
}

func TestRetryStripeEvents_ExpiresOldEvent(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	ctx, cancel := scope.TTL(stripeTestTimeout)
	defer cancel()

	record(t, ctx, store, &payments.StripeEvent{
		ID:       "evt_test_expire",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: time.Now().Add(-4 * 24 * time.Hour),
	})
	must.NoError(t, store.Stripe().MarkEventFailed(ctx, "evt_test_expire", "stripe is down"))

	job := NewRetryStripeEvents(
		&crons.Config{Expression: "* * * * *"},
		store,
		&mocks.StripeClient{},
	)

	result := job.Start()
	must.NotNil(t, result)
	must.Eq(t, crons.ExitSuccess, result.Code)

	event, err := store.Stripe().GetEvent(ctx, "evt_test_expire")
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusDead, event.Status)
	must.Eq(t, "stripe is down", event.LastError)
}

func TestRetryStripeEvents_InvalidPayloadIsDead(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	ctx, cancel := scope.TTL(stripeTestTimeout)
	defer cancel()

	record(t, ctx, store, &payments.StripeEvent{
		ID:      "evt_test_invalid",
		Type:    "invoice.paid",
		Payload: []byte(`not json`),
	})
	must.NoError(t, store.Stripe().MarkEventFailed(ctx, "evt_test_invalid", "stripe is down"))

	job := NewRetryStripeEvents(
		&crons.Config{Expression: "* * * * *"},
		store,
		&mocks.StripeClient{},
	)

	result := job.Start()
	must.NotNil(t, result)
	must.Eq(t, crons.ExitSuccess, result.Code)

	event, err := store.Stripe().GetEvent(ctx, "evt_test_invalid")
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusDead, event.Status)
	must.StrContains(t, event.LastError, "invalid event payload")
}

func TestRetryStripeEvents_EventWithoutDataIsDead(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	ctx, cancel := scope.TTL(stripeTestTimeout)
	defer cancel()

	record(t, ctx, store, &payments.StripeEvent{
		ID:      "evt_test_nodata",
		Type:    "invoice.paid",
		Payload: []byte(`{"id":"evt_test_nodata"}`),
	})
	must.NoError(t, store.Stripe().MarkEventFailed(ctx, "evt_test_nodata", "stripe is down"))

	job := NewRetryStripeEvents(
		&crons.Config{Expression: "* * * * *"},
		store,
		&mocks.StripeClient{},
	)

	result := job.Start()
	must.NotNil(t, result)
	must.Eq(t, crons.ExitSuccess, result.Code)

	event, err := store.Stripe().GetEvent(ctx, "evt_test_nodata")
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusDead, event.Status)
	must.StrContains(t, event.LastError, "missing data")
}

func TestRetryStripeEvents_NoEvents(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	job := NewRetryStripeEvents(
		&crons.Config{Expression: "* * * * *"},
		store,
		&mocks.StripeClient{},
	)

	result := job.Start()
	must.NotNil(t, result)
	must.Eq(t, crons.ExitSuccess, result.Code)
}

// record persists a webhook event in the inbox for later processing.
func record(t *testing.T, ctx scope.C, store database.Store, event *payments.StripeEvent) {
	t.Helper()

	created, err := store.Stripe().RecordEvent(ctx, event)
	must.NoError(t, err)
	must.True(t, created)
}

// insertStripeUser creates a user for the stripe event tests.
func insertStripeUser(t *testing.T, ctx scope.C, store database.Store, username string) users.Identity {
	t.Helper()

	salt := encrypt.Salt()

	request := &users.CreateUserRequest{
		Email:         username + "@example.com",
		Username:      username,
		Salt:          salt,
		Passphrase:    conceal.New("pass+" + username),
		Notifications: true,
		Spam:          true,
	}

	key := encrypt.DeriveKey(request.Passphrase, request.Salt)
	verifier := encrypt.MakeVerifier(key)

	id, err := store.Users().Create(ctx, request, verifier)
	must.NoError(t, err)
	must.Positive(t, id)

	return id
}
