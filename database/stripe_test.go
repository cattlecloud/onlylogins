package database

import (
	"fmt"
	"testing"

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
