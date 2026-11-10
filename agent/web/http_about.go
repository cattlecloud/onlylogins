// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"html/template"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/agent/web/pages"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/users"
)

type aboutHTTP struct {
	log         *ulog.Log
	environment runtime.Environment
	store       database.Store
	html        *template.Template
}

func (h *aboutHTTP) Path() string {
	return "/about"
}

func (h *aboutHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *aboutHTTP) Metric() string {
	return "about"
}

func (h *aboutHTTP) Consumer() consumer {
	return normal
}

func (h *aboutHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *aboutHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return perr
	}

	head := &pages.Head{
		Title:       format.PageTitle("About"),
		Styles:      []string{"basics.css"},
		Canonical:   h.environment.Canonical("%s", h.Path()),
		Description: h.description(),
		Profile:     profile,
	}

	middle := struct{}{}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, middle, nil).Execute(w, session)
}

func (h *aboutHTTP) description() string {
	const text = `
	Store login usernames, passwords, 2FA backup codes, securely.`
	return format.PageDescription(text)
}
