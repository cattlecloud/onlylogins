// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/sessions/cookies"
	"cattlecloud.net/onlylogins/pkg/users"
)

type accountHTMX struct {
	log     *ulog.Log
	store   database.Store
	session middles.Sessions[users.Identity]
}

func (h *accountHTMX) Path() string {
	return "/internal/logout"
}

func (h *accountHTMX) Methods() []string {
	return []string{http.MethodPost, http.MethodDelete}
}

func (h *accountHTMX) Consumer() consumer {
	return component
}

func (h *accountHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodPost:   h.post,
			http.MethodDelete: h.delete,
		},
	)
}

func (h *accountHTMX) delete(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	sess := middles.GetSession[users.Identity](r)
	if !sess.Active() {
		return abort(h.log, w, errors.New("no user session to delete"))
	}

	// a user with a paid subscription must cancel it before the account
	// can be deleted; otherwise they would keep being billed for a
	// subscription they no longer have access to
	allowed, cerr := h.canDelete(ctx, sess.Identity())
	if cerr != nil {
		return abort(h.log, w, cerr)
	}
	if !allowed {
		h.log.W.Fmt("user %d cannot delete account while subscribed", sess.Identity())
		return conflict(h.log, w, errors.New("cancel your subscription before deleting your account"))
	}

	h.log.I.Fmt("delete user %d", sess.Identity())
	if err := h.store.Users().Delete(ctx, sess.Identity()); err != nil {
		h.log.E.Fmt("unable to delete user account: %v", err)
		return abort(h.log, w, err)
	}
	h.expire(w, sess)
	return htmx.SetRedirect(w, "/")
}

// canDelete reports whether the given user is allowed to delete their
// account.
//
// A user on a paid plan is still subscribed; they must cancel the
// subscription first. As a safeguard, a subscription which is still able to
// bill the customer also blocks the deletion, even if the local plan has
// already been downgraded.
func (h *accountHTMX) canDelete(ctx scope.C, uid users.Identity) (bool, error) {
	usage, uerr := h.store.Plans().Usage(ctx, uid)
	if uerr != nil {
		return false, fmt.Errorf("unable to get plan usage: %w", uerr)
	}

	if usage.Plan.ID == records.PlansPatreon {
		return false, nil
	}

	sub, serr := h.store.Stripe().GetSubscription(ctx, uid)
	if serr != nil {
		if errors.Is(serr, sql.ErrNoRows) {
			return true, nil
		}
		return false, fmt.Errorf("unable to look up subscription: %w", serr)
	}

	if subscriptionBlocksDeletion(sub.Status) {
		return false, nil
	}

	return true, nil
}

// subscriptionBlocksDeletion reports whether a subscription in the given
// status prevents the deletion of an account.
//
// Only terminal statuses, which cannot bill the customer any further, allow
// the deletion of an account.
func subscriptionBlocksDeletion(status string) bool {
	switch status {
	case "canceled", "unpaid", "incomplete_expired":
		return false
	default:
		return true
	}
}

func (h *accountHTMX) post(_ scope.C, w http.ResponseWriter, r *http.Request) error {
	// if the user is logged in, expire the cookie and send them to homepage
	if sess := middles.GetSession[users.Identity](r); sess.Active() {
		h.log.T.Fmt("user %d is logging out", sess.Identity())
		h.expire(w, sess)
		return htmx.SetRedirect(w, "/")
	}

	// otherwise send them to the login page (how did they even get here?)
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	return nil
}

func (h *accountHTMX) expire(w http.ResponseWriter, session identity.UserSession[users.Identity]) {
	// delete the session token server-side
	if err := h.session.Delete(session.Token()); err != nil {
		h.log.E.Fmt("unable to delete session: %v", err)
	}

	// delete the user encryption cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cookies.Encryption,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	// delete the user session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cookies.Session,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
}
