// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"html/template"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/urlpath"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type modUserContent struct {
	User *sitecontent.UserModeration
}

type modUserHTTP struct {
	log   *ulog.Log
	store database.Store
	html  *template.Template
}

func (h *modUserHTTP) Path() string {
	return "/moderate/users/{id}"
}

func (h *modUserHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *modUserHTTP) Consumer() consumer {
	return moderate
}

func (h *modUserHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *modUserHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	id := h.idFrom(r)

	user, uerr := h.store.Users().Moderate(ctx, id)
	if uerr != nil {
		return abort(h.log, w, uerr)
	}

	content := &modUserContent{
		User: user,
	}

	return h.html.ExecuteTemplate(w, "layout", content)
}

func (h *modUserHTTP) idFrom(r *http.Request) users.Identity {
	var id users.Identity
	urlpath.MustParse(r, urlpath.Schema{
		"id": urlpath.Int(&id),
	})
	return id
}
