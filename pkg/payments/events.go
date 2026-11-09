package payments

import "time"

// Stripe event inbox states.
const (
	EventStatusPending = "pending"
	EventStatusDone    = "done"
	EventStatusFailed  = "failed"
	EventStatusDead    = "dead"
)

// StripeEvent is a verified Stripe webhook event persisted before processing.
//
// The inbox exists so that an event which verifies but cannot be applied (for
// example because it arrives before the local subscription exists) is not lost
// when Stripe stops retrying deliveries.
type StripeEvent struct {
	ID        string
	Type      string
	Livemode  bool
	Created   int64
	Received  time.Time
	Payload   []byte
	Status    string
	Attempts  int
	LastError string
}
