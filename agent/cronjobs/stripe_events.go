// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package cronjobs

import (
	"encoding/json"
	"fmt"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/crons"
	"cattlecloud.net/onlylogins/pkg/payments"
	"github.com/hashicorp/cronexpr"
	"github.com/stripe/stripe-go/v85"
)

const (
	// stripeRetryGrace is how long an event is retried before it is
	// considered dead; this mirrors the delivery window of stripe.
	stripeRetryGrace = 3 * 24 * time.Hour

	// stripeRetryBatch is the maximum number of events retried in a single
	// run of the job.
	stripeRetryBatch = 100

	// stripeTimeout is the amount of time allowed for a single run.
	stripeTimeout = 10 * time.Minute
)

func NewRetryStripeEvents(config *crons.Config, store database.Store, client payments.Client) *RetryStripeEvents {
	log := ulog.New("cron/stripe-events")
	return &RetryStripeEvents{
		expression: cronexpr.MustParse(config.Expression),
		store:      store,
		processor:  payments.NewProcessor(store.Stripe(), client),
		log:        log,
		now:        time.Now,
	}
}

// RetryStripeEvents retries failed webhook events.
//
// These get pulled from records in the stripe_events table marked failure.
//
// Each event is retried on every run of the cronjob (which runs every 15
// minutes). We no longer retry after the 3 day grace period.
type RetryStripeEvents struct {
	store      database.Store
	processor  *payments.Processor
	expression *cronexpr.Expression
	log        *ulog.Log
	now        func() time.Time
}

func (c *RetryStripeEvents) Name() string {
	return "stripe-events"
}

func (c *RetryStripeEvents) Schedule() *cronexpr.Expression {
	return c.expression
}

func (c *RetryStripeEvents) Start() *crons.RunMetric {
	start := time.Now()

	ctx, cancel := scope.TTL(stripeTimeout)
	defer cancel()

	cutoff := c.now().Add(-stripeRetryGrace)

	// events which have been given the full opportunity to be applied are no
	// longer retried
	expired, xerr := c.store.Stripe().ExpireEvents(ctx, cutoff)
	if xerr != nil {
		c.log.E.Fmt("failed to expire stripe events: %v", xerr)
		return &crons.RunMetric{
			Code:  crons.ExitFailure,
			Start: start,
			Stop:  time.Now(),
		}
	}
	if expired > 0 {
		c.log.W.Fmt("marked %d stripe events dead after %v", expired, stripeRetryGrace)
	}

	events, rerr := c.store.Stripe().RetryableEvents(ctx, cutoff, stripeRetryBatch)
	if rerr != nil {
		c.log.E.Fmt("failed to list stripe events for retry: %v", rerr)
		return &crons.RunMetric{
			Code:  crons.ExitFailure,
			Start: start,
			Stop:  time.Now(),
		}
	}

	var succeeded, failed int

	for _, record := range events {
		var event stripe.Event
		if err := json.Unmarshal(record.Payload, &event); err != nil {
			c.log.E.Fmt("unable to decode stripe event %s: %v", record.ID, err)
			c.markDead(ctx, record.ID, fmt.Sprintf("invalid event payload: %v", err))
			continue
		}

		if event.Data == nil {
			c.log.E.Fmt("stripe event %s is missing data", record.ID)
			c.markDead(ctx, record.ID, "event is missing data")
			continue
		}

		if err := c.processor.Dispatch(ctx, event); err != nil {
			c.log.D.Fmt("unable to process stripe event %s (attempt %d): %v", record.ID, record.Attempts+1, err)
			c.markFailed(ctx, record.ID, err.Error())
			failed++
			continue
		}

		if err := c.store.Stripe().MarkEventDone(ctx, record.ID); err != nil {
			c.log.E.Fmt("unable to mark stripe event %s processed: %v", record.ID, err)
			continue
		}
		succeeded++
	}

	c.log.I.Fmt("processed %d stripe events: %d succeeded, %d failed", len(events), succeeded, failed)

	return &crons.RunMetric{
		Code:  crons.ExitSuccess,
		Start: start,
		Stop:  time.Now(),
	}
}

func (c *RetryStripeEvents) markFailed(ctx scope.C, eventID, reason string) {
	if err := c.store.Stripe().MarkEventFailed(ctx, eventID, reason); err != nil {
		c.log.E.Fmt("unable to mark stripe event %s failed: %v", eventID, err)
	}
}

func (c *RetryStripeEvents) markDead(ctx scope.C, eventID, reason string) {
	if err := c.store.Stripe().MarkEventDead(ctx, eventID, reason); err != nil {
		c.log.E.Fmt("unable to mark stripe event %s dead: %v", eventID, err)
	}
}
