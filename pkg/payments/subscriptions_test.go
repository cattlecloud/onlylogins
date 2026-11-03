package payments

import (
	"testing"

	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/test/must"
)

func TestStripeSubscription_Validate(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		s := (*StripeSubscription)(nil)
		err := s.Validate()
		must.ErrorContains(t, err, "subscription is nil")
	})

	t.Run("subscription_id not set", func(t *testing.T) {
		s := &StripeSubscription{
			SubscriptionID:   "",
			PriceID:          "price_test",
			Status:           "active",
			CurrentPeriodEnd: 1735689600,
		}
		err := s.Validate()
		must.ErrorContains(t, err, "subscription_id not set")
	})

	t.Run("price_id not set", func(t *testing.T) {
		s := &StripeSubscription{
			SubscriptionID:   "sub_test1234567890",
			PriceID:          "",
			Status:           "active",
			CurrentPeriodEnd: 1735689600,
		}
		err := s.Validate()
		must.ErrorContains(t, err, "price_id not set")
	})

	t.Run("status not set", func(t *testing.T) {
		s := &StripeSubscription{
			SubscriptionID:   "sub_test1234567890",
			PriceID:          "price_test",
			Status:           "",
			CurrentPeriodEnd: 1735689600,
		}
		err := s.Validate()
		must.ErrorContains(t, err, "status not set")
	})

	t.Run("valid", func(t *testing.T) {
		s := &StripeSubscription{
			SubscriptionID:   "sub_test1234567890",
			PriceID:          "price_test",
			Status:           "active",
			CurrentPeriodEnd: 1735689600,
		}
		err := s.Validate()
		must.NoError(t, err)
	})
}

func TestUpdateStripeSubscription_Validate(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		s := (*UpdateStripeSubscription)(nil)
		err := s.Validate()
		must.ErrorContains(t, err, "update subscription is nil")
	})

	t.Run("subscription is nil", func(t *testing.T) {
		s := &UpdateStripeSubscription{
			User: users.Identity(1),
			Sub:  nil,
			Plan: records.PlansPatreon,
		}
		err := s.Validate()
		must.ErrorContains(t, err, "subscription is nil")
	})

	t.Run("user not set", func(t *testing.T) {
		s := &UpdateStripeSubscription{
			User: 0,
			Sub: &StripeSubscription{
				SubscriptionID:   "sub_test1234567890",
				PriceID:          "price_test",
				Status:           "active",
				CurrentPeriodEnd: 1735689600,
			},
			Plan: records.PlansPatreon,
		}
		err := s.Validate()
		must.ErrorContains(t, err, "user must be set")
	})

	t.Run("invalid plan", func(t *testing.T) {
		s := &UpdateStripeSubscription{
			User: users.Identity(1),
			Sub: &StripeSubscription{
				SubscriptionID:   "sub_test1234567890",
				PriceID:          "price_test",
				Status:           "active",
				CurrentPeriodEnd: 1735689600,
			},
			Plan: records.PlansEnum(99),
		}
		err := s.Validate()
		must.ErrorContains(t, err, "invalid plan")
	})

	t.Run("nested subscription invalid", func(t *testing.T) {
		s := &UpdateStripeSubscription{
			User: users.Identity(1),
			Sub: &StripeSubscription{
				SubscriptionID:   "",
				PriceID:          "price_test",
				Status:           "active",
				CurrentPeriodEnd: 1735689600,
			},
			Plan: records.PlansPatreon,
		}
		err := s.Validate()
		must.ErrorContains(t, err, "subscription_id not set")
	})

	t.Run("valid", func(t *testing.T) {
		s := &UpdateStripeSubscription{
			User: users.Identity(1),
			Sub: &StripeSubscription{
				SubscriptionID:   "sub_test1234567890",
				PriceID:          "price_test",
				Status:           "active",
				CurrentPeriodEnd: 1735689600,
			},
			Plan: records.PlansPatreon,
		}
		err := s.Validate()
		must.NoError(t, err)
	})
}
