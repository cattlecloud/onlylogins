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
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type toolsContent struct {
	Tools []sitecontent.Tool
}

type toolsHTTP struct {
	log         *ulog.Log
	environment runtime.Environment
	store       database.Store
	html        *template.Template
}

func (h *toolsHTTP) Path() string {
	return "/tools"
}

func (h *toolsHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *toolsHTTP) Metric() string {
	return "tools"
}

func (h *toolsHTTP) Consumer() consumer {
	return normal
}

func (h *toolsHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *toolsHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return perr
	}

	head := &pages.Head{
		Title:       format.PageTitle("Tools"),
		Styles:      []string{"tools.css"},
		Canonical:   h.environment.Canonical("%s", h.Path()),
		Description: h.description(),
		Profile:     profile,
	}

	middle := &toolsContent{
		Tools: sitecontent.Tools(),
	}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, middle).Execute(w, session)
}

func (h *toolsHTTP) description() string {
	const text = `
	One-off tools for working with the data in your account.`
	return format.PageDescription(text)
}
