// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package payments

import (
	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/database/records"
	"github.com/stripe/stripe-go/v87"
	bs "github.com/stripe/stripe-go/v87/billingportal/session"
	cs "github.com/stripe/stripe-go/v87/checkout/session"
	"github.com/stripe/stripe-go/v87/customer"
	"github.com/stripe/stripe-go/v87/subscription"
	"github.com/stripe/stripe-go/v87/webhook"
)

// Metadata keys used to associate Stripe objects with the onlylogins
// application. They are scoped so that other applications sharing the same
// Stripe account cannot collide with our user identifiers.
const (
	MetadataUserIDKey  = "onlylogins_user_id"
	MetadataPriceIDKey = "onlylogins_price_id"
)

// Client represents the interactions we make with the upstream Stripe API.
type Client interface {
	PatreonProductID() string
	PlanFromPriceID(string) (records.PlansEnum, bool)
	GetOrCreateCustomer(scope.C, string, string) (string, error)
	CreateCheckoutSession(scope.C, string, string, string, string, string) (*stripe.CheckoutSession, error)
	ConstructWebhookEvent([]byte, string) (stripe.Event, error)
	FetchSubscription(scope.C, string) (*stripe.Subscription, error)
	CreateBillingPortalSession(scope.C, string, string) (*stripe.BillingPortalSession, error)
}

type client struct {
	secretKey        string
	webhookSecret    string
	patreonProductID string
}

// New creates a real implementation of Client for interacting with Stripe.
func New(config *Config) Client {
	stripe.Key = config.SecretKey
	return &client{
		secretKey:        config.SecretKey,
		webhookSecret:    config.WebhookSecret,
		patreonProductID: config.PatreonProductID,
	}
}

func (c *client) PatreonProductID() string {
	return c.patreonProductID
}

func (c *client) PlanFromPriceID(priceID string) (records.PlansEnum, bool) {
	switch priceID {
	case c.patreonProductID:
		return records.PlansPatreon, true
	default:
		return 0, false
	}
}

func (c *client) GetOrCreateCustomer(_ scope.C, email, userID string) (string, error) {
	params := &stripe.CustomerSearchParams{
		SearchParams: stripe.SearchParams{
			Query: "metadata['" + MetadataUserIDKey + "']:'" + userID + "'",
		},
	}
	params.AddExpand("data")

	iter := customer.Search(params)
	if iter.Next() {
		return iter.Customer().ID, nil
	}

	newCustomerParams := &stripe.CustomerParams{
		Email: stripe.String(email),
		Metadata: map[string]string{
			MetadataUserIDKey: userID,
		},
	}
	newCustomer, err := customer.New(newCustomerParams)
	if err != nil {
		return "", err
	}
	return newCustomer.ID, nil
}

func (c *client) CreateCheckoutSession(_ scope.C, customerID, priceID, userID, successURL, cancelURL string) (*stripe.CheckoutSession, error) {
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Metadata: map[string]string{
			MetadataUserIDKey:  userID,
			MetadataPriceIDKey: priceID,
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				MetadataUserIDKey: userID,
			},
		},
	}
	params.AddExpand("line_items.data.price")
	return cs.New(params)
}

func (c *client) ConstructWebhookEvent(payload []byte, sig string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, sig, c.webhookSecret)
}

func (c *client) FetchSubscription(_ scope.C, subscriptionID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{}
	params.AddExpand("items")
	return subscription.Get(subscriptionID, params)
}

func (c *client) CreateBillingPortalSession(_ scope.C, customerID string, returnURL string) (*stripe.BillingPortalSession, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}
	return bs.New(params)
}
