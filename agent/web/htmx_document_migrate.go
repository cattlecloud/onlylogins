// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"errors"
	"net/http"
	"path"
	"strings"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/urlpath"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type migrateDocumentHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *migrateDocumentHTMX) Path() string {
	return "/internal/document/migrate/{folder}/{document}"
}

func (h *migrateDocumentHTMX) Methods() []string {
	return []string{http.MethodPost}
}

func (h *migrateDocumentHTMX) Consumer() consumer {
	return component
}

func (h *migrateDocumentHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodPost: h.post,
		},
	)
}

func (h *migrateDocumentHTMX) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	request := h.fields(r, session.Identity())

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if merr := h.store.Documents().Migrate(ctx, request); merr != nil {
		if errors.Is(merr, database.ErrDocumentExists) {
			return problem(w, database.ErrDocumentExists)
		}
		return abort(h.log, w, merr)
	}

	// the document URL changed; send the user to the new url
	target := path.Join("/folders", request.Target, request.Document)
	return htmx.SetRedirect(w, target)
}

func (h *migrateDocumentHTMX) fields(r *http.Request, user users.Identity) *sitecontent.MigrateDocumentRequest {
	var target string
	forms.MustParse(r, forms.Schema{
		"target": forms.String(&target),
	})
	target = strings.TrimSpace(target)
	folder, document := h.folderDocumentFrom(r)
	return &sitecontent.MigrateDocumentRequest{
		User:     user,
		Folder:   folder,
		Document: document,
		Target:   target,
	}
}

func (h *migrateDocumentHTMX) folderDocumentFrom(r *http.Request) (string, string) {
	var folder, document string
	urlpath.MustParse(r, urlpath.Schema{
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
	})
	return folder, document
}
