// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"fmt"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/payments"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/test/must"
)

func TestStripe_GetCustomer(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	uid := insertUser(t, ctx, db, "test")

	t.Run("not found", func(t *testing.T) {
		_, exists, err := stripe.GetCustomer(ctx, uid)
		must.NoError(t, err)
		must.False(t, exists)
	})

	t.Run("found", func(t *testing.T) {
		customerID := "cus_test123"
		serr := stripe.SetCustomer(ctx, uid, customerID)
		must.NoError(t, serr)

		got, exists, err := stripe.GetCustomer(ctx, uid)
		must.NoError(t, err)
		must.True(t, exists)
		must.Eq(t, customerID, got)
	})
}

func TestStripe_SetCustomer(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	uid := insertUser(t, ctx, db, "test")

	t.Run("create new", func(t *testing.T) {
		customerID := "cus_new123"
		serr := stripe.SetCustomer(ctx, uid, customerID)
		must.NoError(t, serr)

		got, exists, err := stripe.GetCustomer(ctx, uid)
		must.NoError(t, err)
		must.True(t, exists)
		must.Eq(t, customerID, got)
	})

	t.Run("replace existing", func(t *testing.T) {
		oldID := "cus_old456"
		newID := "cus_new789"
		must.NoError(t, stripe.SetCustomer(ctx, uid, oldID))
		must.NoError(t, stripe.SetCustomer(ctx, uid, newID))

		got, exists, err := stripe.GetCustomer(ctx, uid)
		must.NoError(t, err)
		must.True(t, exists)
		must.Eq(t, newID, got)
	})
}

func TestStripe_GetSubscription(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	uid := insertUser(t, ctx, db, "test")

	t.Run("not found", func(t *testing.T) {
		_, err := stripe.GetSubscription(ctx, uid)
		must.Error(t, err)
	})

	t.Run("found", func(t *testing.T) {
		sub := &payments.StripeSubscription{
			SubscriptionID:   "sub_test1234567890",
			PriceID:          "price_test",
			Status:           "active",
			CurrentPeriodEnd: 1735689600,
			CancelAt:         1738281600,
		}
		update := &payments.UpdateStripeSubscription{
			User: uid,
			Sub:  sub,
			Plan: records.PlansPatreon,
		}
		must.NoError(t, stripe.UpsertSubscription(ctx, update))

		got, err := stripe.GetSubscription(ctx, uid)
		must.NoError(t, err)
		must.NotNil(t, got)
		must.Eq(t, sub.SubscriptionID, got.SubscriptionID)
		must.Eq(t, sub.PriceID, got.PriceID)
		must.Eq(t, sub.Status, got.Status)
		must.Eq(t, sub.CurrentPeriodEnd, got.CurrentPeriodEnd)
		must.Eq(t, sub.CancelAt, got.CancelAt)
	})
}

func TestStripe_UpsertSubscription(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()
	u := db.Users()

	uid := insertUser(t, ctx, db, "test")

	sub := &payments.StripeSubscription{
		SubscriptionID:   "sub_test45678901234",
		PriceID:          "price_personal",
		Status:           "active",
		CurrentPeriodEnd: 1735689600,
	}
	update := &payments.UpdateStripeSubscription{
		User: uid,
		Sub:  sub,
		Plan: records.PlansPatreon,
	}
	must.NoError(t, stripe.UpsertSubscription(ctx, update))

	got, err := stripe.GetSubscription(ctx, uid)
	must.NoError(t, err)
	must.NotNil(t, got)
	must.Eq(t, sub.SubscriptionID, got.SubscriptionID)

	moderate, perr := u.Moderate(ctx, uid)
	must.NoError(t, perr)
	must.Eq(t, records.PlansPatreon, moderate.Plan.ID)
}

func TestStripe_GetUserBySubscription(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	uid := insertUser(t, ctx, db, "test")

	t.Run("not found", func(t *testing.T) {
		_, err := stripe.GetUserBySubscription(ctx, "nonexistent_sub")
		must.Error(t, err)
	})

	t.Run("found", func(t *testing.T) {
		sub := &payments.StripeSubscription{
			SubscriptionID:   "sub_findme_test_new",
			PriceID:          "price_test",
			Status:           "active",
			CurrentPeriodEnd: 1735689600,
		}
		update := &payments.UpdateStripeSubscription{
			User: uid,
			Sub:  sub,
			Plan: records.PlansPatreon,
		}
		must.NoError(t, stripe.UpsertSubscription(ctx, update))

		got, err := stripe.GetUserBySubscription(ctx, "sub_findme_test_new")
		must.NoError(t, err)
		must.Eq(t, uid, got)
	})
}

func TestStripe_DowngradeExpired(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()
	u := db.Users()

	paid := insertUser(t, ctx, db, "paid")
	free := insertUser(t, ctx, db, "free")

	upgradeUser(ctx, t, stripe, paid, records.PlansPatreon)
	upgradeUser(ctx, t, stripe, free, records.PlansFree)

	t.Run("no expired", func(t *testing.T) {
		count, err := stripe.DowngradeExpired(ctx)
		must.NoError(t, err)
		must.Eq(t, 0, count)

		moderate, perr := u.Moderate(ctx, paid)
		must.NoError(t, perr)
		must.Eq(t, records.PlansPatreon, moderate.Plan.ID)
	})

	t.Run("one expired and downgraded", func(t *testing.T) {
		sub := &payments.StripeSubscription{
			SubscriptionID:   "sub_expired_test",
			PriceID:          "price_personal",
			Status:           "canceled",
			CurrentPeriodEnd: 1735689600,
		}
		update := &payments.UpdateStripeSubscription{
			User: paid,
			Sub:  sub,
			Plan: records.PlansPatreon,
		}
		must.NoError(t, stripe.UpsertSubscription(ctx, update))

		count, err := stripe.DowngradeExpired(ctx)
		must.NoError(t, err)
		must.Eq(t, 1, count)

		moderate, perr := u.Moderate(ctx, paid)
		must.NoError(t, perr)
		must.Eq(t, records.PlansFree, moderate.Plan.ID)
	})
}

func TestStripe_GetUserByCustomer(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	uid := insertUser(t, ctx, db, "test")

	t.Run("not found", func(t *testing.T) {
		_, err := stripe.GetUserByCustomer(ctx, "cus_missing")
		must.Error(t, err)
	})

	t.Run("found", func(t *testing.T) {
		must.NoError(t, stripe.SetCustomer(ctx, uid, "cus_find_me"))

		got, err := stripe.GetUserByCustomer(ctx, "cus_find_me")
		must.NoError(t, err)
		must.Eq(t, uid, got)
	})
}

func TestStripe_EventInbox(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	event := &payments.StripeEvent{
		ID:       "evt_test_inbox",
		Type:     "invoice.paid",
		Livemode: true,
		Created:  1735689600,
		Payload:  []byte(`{"id":"evt_test_inbox"}`),
	}

	created, err := stripe.RecordEvent(ctx, event)
	must.NoError(t, err)
	must.True(t, created)

	// Recording the same event again must not create a second row.
	created, err = stripe.RecordEvent(ctx, event)
	must.NoError(t, err)
	must.False(t, created)

	got, err := stripe.GetEvent(ctx, event.ID)
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusPending, got.Status)
	must.Eq(t, 0, got.Attempts)
	must.Eq(t, "", got.LastError)
	must.Eq(t, true, got.Livemode)
	must.Eq(t, event.Created, got.Created)
	must.Eq(t, event.Payload, got.Payload)
	must.True(t, time.Since(got.Received) < time.Minute)

	must.NoError(t, stripe.MarkEventFailed(ctx, event.ID, "stripe is down"))
	got, err = stripe.GetEvent(ctx, event.ID)
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusFailed, got.Status)
	must.Eq(t, 1, got.Attempts)
	must.Eq(t, "stripe is down", got.LastError)

	must.NoError(t, stripe.MarkEventDone(ctx, event.ID))
	got, err = stripe.GetEvent(ctx, event.ID)
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusDone, got.Status)
	must.Eq(t, 2, got.Attempts)
	must.Eq(t, "", got.LastError)
}

func TestStripe_MarkEventDead(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	event := &payments.StripeEvent{
		ID:      "evt_test_dead",
		Type:    "invoice.paid",
		Payload: []byte(`{"id":"evt_test_dead"}`),
	}

	_, err := stripe.RecordEvent(ctx, event)
	must.NoError(t, err)

	must.NoError(t, stripe.MarkEventDead(ctx, event.ID, "invalid event payload"))

	got, err := stripe.GetEvent(ctx, event.ID)
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusDead, got.Status)
	must.Eq(t, "invalid event payload", got.LastError)
}

func TestStripe_RetryableEvents(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	now := time.Now()

	// an event which has not been applied yet
	pending := &payments.StripeEvent{
		ID:       "evt_test_pending",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now,
	}
	_, err := stripe.RecordEvent(ctx, pending)
	must.NoError(t, err)

	// an event which failed to be applied
	failed := &payments.StripeEvent{
		ID:       "evt_test_failed",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now,
	}
	_, err = stripe.RecordEvent(ctx, failed)
	must.NoError(t, err)
	must.NoError(t, stripe.MarkEventFailed(ctx, failed.ID, "stripe is down"))

	// an event which was applied successfully
	done := &payments.StripeEvent{
		ID:       "evt_test_done",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now,
	}
	_, err = stripe.RecordEvent(ctx, done)
	must.NoError(t, err)
	must.NoError(t, stripe.MarkEventDone(ctx, done.ID))

	// an event which is too old to be retried
	old := &payments.StripeEvent{
		ID:       "evt_test_old",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now.Add(-4 * 24 * time.Hour),
	}
	_, err = stripe.RecordEvent(ctx, old)
	must.NoError(t, err)
	must.NoError(t, stripe.MarkEventFailed(ctx, old.ID, "stripe is down"))

	events, err := stripe.RetryableEvents(ctx, now.Add(-24*time.Hour), 10)
	must.NoError(t, err)
	must.Eq(t, 2, len(events))

	ids := []string{events[0].ID, events[1].ID}
	must.SliceContains(t, ids, pending.ID)
	must.SliceContains(t, ids, failed.ID)
}

func TestStripe_ExpireEvents(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	now := time.Now()

	old := &payments.StripeEvent{
		ID:       "evt_test_expire",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now.Add(-4 * 24 * time.Hour),
	}
	_, err := stripe.RecordEvent(ctx, old)
	must.NoError(t, err)
	must.NoError(t, stripe.MarkEventFailed(ctx, old.ID, "stripe is down"))

	recent := &payments.StripeEvent{
		ID:       "evt_test_recent",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now,
	}
	_, err = stripe.RecordEvent(ctx, recent)
	must.NoError(t, err)
	must.NoError(t, stripe.MarkEventFailed(ctx, recent.ID, "stripe is down"))

	count, err := stripe.ExpireEvents(ctx, now.Add(-72*time.Hour))
	must.NoError(t, err)
	must.Eq(t, 1, count)

	got, err := stripe.GetEvent(ctx, old.ID)
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusDead, got.Status)
	must.Eq(t, "stripe is down", got.LastError)

	got, err = stripe.GetEvent(ctx, recent.ID)
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusFailed, got.Status)
}

func TestStripe_ProblemEvents(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	now := time.Now()

	dead := &payments.StripeEvent{
		ID:       "evt_test_problem_dead",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now,
	}
	_, err := stripe.RecordEvent(ctx, dead)
	must.NoError(t, err)
	must.NoError(t, stripe.MarkEventDead(ctx, dead.ID, "invalid event payload"))

	stuck := &payments.StripeEvent{
		ID:       "evt_test_problem_stuck",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now.Add(-2 * time.Hour),
	}
	_, err = stripe.RecordEvent(ctx, stuck)
	must.NoError(t, err)
	must.NoError(t, stripe.MarkEventFailed(ctx, stuck.ID, "stripe is down"))

	fresh := &payments.StripeEvent{
		ID:       "evt_test_problem_fresh",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now,
	}
	_, err = stripe.RecordEvent(ctx, fresh)
	must.NoError(t, err)
	must.NoError(t, stripe.MarkEventFailed(ctx, fresh.ID, "stripe is down"))

	problems, err := stripe.ProblemEvents(ctx, now.Add(-time.Hour), 10)
	must.NoError(t, err)
	must.Eq(t, 2, len(problems))

	ids := []string{problems[0].ID, problems[1].ID}
	must.SliceContains(t, ids, dead.ID)
	must.SliceContains(t, ids, stuck.ID)
}

func TestStripe_PruneEvents(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	now := time.Now()

	applied := &payments.StripeEvent{
		ID:       "evt_test_prune_applied",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now.Add(-40 * 24 * time.Hour),
	}
	_, err := stripe.RecordEvent(ctx, applied)
	must.NoError(t, err)
	must.NoError(t, stripe.MarkEventDone(ctx, applied.ID))

	failed := &payments.StripeEvent{
		ID:       "evt_test_prune_failed",
		Type:     "invoice.paid",
		Payload:  []byte(`{}`),
		Received: now.Add(-40 * 24 * time.Hour),
	}
	_, err = stripe.RecordEvent(ctx, failed)
	must.NoError(t, err)
	must.NoError(t, stripe.MarkEventFailed(ctx, failed.ID, "stripe is down"))

	count, err := stripe.PruneEvents(ctx, 30*24*time.Hour)
	must.NoError(t, err)
	must.Eq(t, 1, count)

	_, err = stripe.GetEvent(ctx, applied.ID)
	must.Error(t, err)

	// events which have not been applied are kept
	got, err := stripe.GetEvent(ctx, failed.ID)
	must.NoError(t, err)
	must.Eq(t, payments.EventStatusFailed, got.Status)
}

func TestStripe_CurrentPlan(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	stripe := db.Stripe()

	uid := insertUser(t, ctx, db, "test")

	plan, err := stripe.CurrentPlan(ctx, uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansFree, plan)

	upgradeUser(ctx, t, stripe, uid, records.PlansPatreon)

	plan, err = stripe.CurrentPlan(ctx, uid)
	must.NoError(t, err)
	must.Eq(t, records.PlansPatreon, plan)
}

func upgradeUser(ctx scope.C, t *testing.T, s Stripe, uid users.Identity, plan records.PlansEnum) {
	sub := &payments.StripeSubscription{
		SubscriptionID:   fmt.Sprintf("sub_test_%d", uid),
		PriceID:          "price_test",
		Status:           "active",
		CurrentPeriodEnd: 1735689600,
	}
	update := &payments.UpdateStripeSubscription{
		User: uid,
		Sub:  sub,
		Plan: plan,
	}
	err := s.UpsertSubscription(ctx, update)
	must.NoError(t, err)
}
