// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"strings"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/ui"
)

type emailHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *emailHTMX) Path() string {
	return "/internal/email/edit"
}

func (h *emailHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPut}
}

func (h *emailHTMX) Consumer() consumer {
	return component
}

func (h *emailHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
			http.MethodPut: h.put,
		},
	)
}

func (h *emailHTMX) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if not logged in, redirect to login page
	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return abort(h.log, w, perr)
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "email_edit_form.html",
		Fields:   profile.Email,
	})
}

func (h *emailHTMX) put(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if not logged in, redirect to login page
	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	email := h.emailFrom(r)

	if err := users.ValidateEmail(email); err != nil {
		return problem(w, err)
	}

	if err := h.store.Users().SetEmail(ctx, session.Identity(), email); err != nil {
		return abort(h.log, w, err)
	}

	// reload the profile page to show the updated email
	return htmx.SetRedirect(w, "/profile")
}

func (h *emailHTMX) emailFrom(r *http.Request) string {
	var email string
	forms.MustParse(r, forms.Schema{
		"email": forms.String(&email),
	})
	return strings.TrimSpace(email)
}
