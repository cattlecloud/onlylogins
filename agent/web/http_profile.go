// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"fmt"
	"html/template"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/agent/web/pages"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type profileContent struct {
	Profile *sitecontent.Profile
}

type profileHTTP struct {
	log   *ulog.Log
	store database.Store
	html  *template.Template
}

func (h *profileHTTP) Path() string {
	return "/profile"
}

func (h *profileHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *profileHTTP) Metric() string {
	return "profile"
}

func (h *profileHTTP) Consumer() consumer {
	return normal
}

func (h *profileHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *profileHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if user is not logged in, redirect to login page
	if !session.Active() {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil
	}

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return perr
	}

	content, cerr := h.store.Users().Profile(ctx, session.Identity())
	if cerr != nil {
		return fmt.Errorf("unable to get profile: %w", cerr)
	}

	head := &pages.Head{
		Title:       format.PageTitle("Profile"),
		Styles:      []string{"profile.css"},
		Description: h.description(),
		Profile:     profile,
	}

	middle := &profileContent{
		Profile: content,
	}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, middle, nil).Execute(w, session)
}

func (h *profileHTTP) description() string {
	const text = `
	Your username and how many documents you have stored.`
	return format.PageDescription(text)
}
