// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"html/template"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
)

type moderationContent struct {
	UsersCount int
}

type moderationHTTP struct {
	log   *ulog.Log
	store database.Store
	html  *template.Template
}

func (h *moderationHTTP) Path() string {
	return "/moderate"
}

func (h *moderationHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *moderationHTTP) Consumer() consumer {
	return moderate
}

func (h *moderationHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *moderationHTTP) get(ctx scope.C, w http.ResponseWriter, _ *http.Request) error {
	usersCount, ucerr := h.store.Users().Count(ctx)
	if ucerr != nil {
		return abort(h.log, w, ucerr)
	}

	content := &moderationContent{
		UsersCount: usersCount,
	}

	if err := h.html.ExecuteTemplate(w, "layout", content); err != nil {
		h.log.E.Fmt("unable to render moderation page: %v", err)
	}

	return nil
}
