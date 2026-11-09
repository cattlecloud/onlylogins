// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/payments"
)

type stripeWebhookAPI struct {
	log       *ulog.Log
	store     database.Store
	client    payments.Client
	processor *payments.Processor
}

func (h *stripeWebhookAPI) Path() string {
	return "/api/stripe/webhooks"
}

func (h *stripeWebhookAPI) Methods() []string {
	return []string{http.MethodPost}
}

func (h *stripeWebhookAPI) Consumer() consumer {
	return robotic
}

func (h *stripeWebhookAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const long = 1 * time.Minute // give stripe extra time
	ctx, cancel := scope.WithTTL(r.Context(), long)
	defer cancel()

	serve(ctx, h.log, w, r, methods{
		http.MethodPost: h.post,
	})
}

func (h *stripeWebhookAPI) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	// unpack the Stripe webhook event issued to us
	payload, perr := io.ReadAll(r.Body)
	if perr != nil {
		return abort(h.log, w, fmt.Errorf("failed to read webhook body: %w", perr))
	}

	sig := r.Header.Get("Stripe-Signature")

	event, werr := h.client.ConstructWebhookEvent(payload, sig)
	if werr != nil {
		return abort(h.log, w, fmt.Errorf("unable to construct webhook event: %w", werr))
	}

	if event.ID == "" {
		return abort(h.log, w, errors.New("webhook event is missing an id"))
	}

	// record the event in the database before doing anything with it
	// if our processing fails, we can retry to process the event in the
	// supplementary strip-events cronjob that runs every 10 minutes; J.I.C.
	created, rerr := h.store.Stripe().RecordEvent(ctx, &payments.StripeEvent{
		ID:       event.ID,
		Type:     string(event.Type),
		Livemode: event.Livemode,
		Created:  event.Created,
		Payload:  payload,
	})
	if rerr != nil {
		return abort(h.log, w, fmt.Errorf("unable to record webhook event %s: %w", event.ID, rerr))
	}

	// protect against Stripe sending the same event more than once
	if !created {
		prior, gerr := h.store.Stripe().GetEvent(ctx, event.ID)
		if gerr != nil {
			return abort(h.log, w, fmt.Errorf("unable to read webhook event %s: %w", event.ID, gerr))
		}
		if prior.Status == payments.EventStatusDone {
			h.log.D.Fmt("skipping already processed webhook: type=%s id=%s", event.Type, event.ID)
			w.WriteHeader(http.StatusOK)
			return nil
		}
	}

	// actually process the event Stripe has issued to us
	if derr := h.processor.Dispatch(ctx, event); derr != nil {
		if merr := h.store.Stripe().MarkEventFailed(ctx, event.ID, derr.Error()); merr != nil {
			h.log.E.Fmt("unable to mark webhook event %s failed: %v", event.ID, merr)
		}
		return abort(h.log, w, derr)
	}

	if merr := h.store.Stripe().MarkEventDone(ctx, event.ID); merr != nil {
		// the state change succeeded but we could not record it; a retry is
		// safe because the processor is idempotent
		return abort(h.log, w, fmt.Errorf("unable to mark webhook event %s processed: %w", event.ID, merr))
	}

	// whew! we processed the event
	w.WriteHeader(http.StatusOK)
	return nil
}
