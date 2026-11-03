package mocks

import (
	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/database/records"
	"github.com/stripe/stripe-go/v85"
)

type StripeClient struct {
	MockPatreonProductID     string
	MockPlanFromPriceIDEnum  records.PlansEnum
	MockPlanFromPriceIDBool  bool
	MockCustomerID           string
	MockCustomerErr          error
	MockCheckoutSession      *stripe.CheckoutSession
	MockCheckoutErr          error
	MockEvent                stripe.Event
	MockEventErr             error
	MockSubscription         *stripe.Subscription
	MockSubscriptionErr      error
	MockBillingPortalSession *stripe.BillingPortalSession
	MockBillingPortalErr     error
}

func (sc *StripeClient) PatreonProductID() string {
	return sc.MockPatreonProductID
}

func (sc *StripeClient) PlanFromPriceID(string) (records.PlansEnum, bool) {
	return sc.MockPlanFromPriceIDEnum, sc.MockPlanFromPriceIDBool
}

func (sc *StripeClient) GetOrCreateCustomer(scope.C, string, string) (string, error) {
	return sc.MockCustomerID, sc.MockCustomerErr
}

func (sc *StripeClient) CreateCheckoutSession(scope.C, string, string, string, string, string) (*stripe.CheckoutSession, error) {
	return sc.MockCheckoutSession, sc.MockCheckoutErr
}

func (sc *StripeClient) ConstructWebhookEvent([]byte, string) (stripe.Event, error) {
	return sc.MockEvent, sc.MockEventErr
}

func (sc *StripeClient) FetchSubscription(scope.C, string) (*stripe.Subscription, error) {
	return sc.MockSubscription, sc.MockSubscriptionErr
}

func (sc *StripeClient) CreateBillingPortalSession(scope.C, string, string) (*stripe.BillingPortalSession, error) {
	return sc.MockBillingPortalSession, sc.MockBillingPortalErr
}
