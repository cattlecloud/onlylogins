// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package payments

import (
	"errors"

	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/users"
)

type UpdateStripeSubscription struct {
	User users.Identity
	Sub  *StripeSubscription
	Plan records.PlansEnum
}

func (s *UpdateStripeSubscription) Validate() error {
	if s == nil {
		return errors.New("update subscription is nil")
	}
	if s.Sub == nil {
		return errors.New("subscription is nil")
	}
	if err := s.Sub.Validate(); err != nil {
		return err
	}
	if s.User <= 0 {
		return errors.New("user must be set")
	}
	if s.Plan != records.PlansFree && s.Plan != records.PlansPatreon {
		return errors.New("invalid plan")
	}
	return nil
}

type StripeSubscription struct {
	SubscriptionID   string
	PriceID          string
	Status           string
	CurrentPeriodEnd int64
	CancelAt         int64
}

func (s *StripeSubscription) Validate() error {
	if s == nil {
		return errors.New("subscription is nil")
	}
	if s.SubscriptionID == "" {
		return errors.New("subscription_id not set")
	}
	if s.PriceID == "" {
		return errors.New("price_id not set")
	}
	if s.Status == "" {
		return errors.New("status not set")
	}
	return nil
}
