// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"errors"
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/database"
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
	if sess := middles.GetSession[users.Identity](r); sess.Active() {
		h.log.I.Fmt("delete user %d", sess.Identity())
		if err := h.store.Users().Delete(ctx, sess.Identity()); err != nil {
			h.log.E.Fmt("unable to delete user account: %v", err)
			return abort(h.log, w, err)
		}
		h.expire(w, sess)
		return htmx.SetRedirect(w, "/")
	}

	return abort(h.log, w, errors.New("no user session to delete"))
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
