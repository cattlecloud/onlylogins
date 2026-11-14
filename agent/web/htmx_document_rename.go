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

type renameDocumentHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *renameDocumentHTMX) Path() string {
	return "/internal/document/rename/{folder}/{document}"
}

func (h *renameDocumentHTMX) Methods() []string {
	return []string{http.MethodPost}
}

func (h *renameDocumentHTMX) Consumer() consumer {
	return component
}

func (h *renameDocumentHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodPost: h.post,
		},
	)
}

func (h *renameDocumentHTMX) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	request := h.fields(r, session.Identity())

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if rerr := h.store.Documents().Rename(ctx, request); rerr != nil {
		if errors.Is(rerr, database.ErrDocumentExists) {
			return problem(w, database.ErrDocumentExists)
		}
		return abort(h.log, w, rerr)
	}

	// the document URL changed; send the user to the new url
	target := path.Join("/folders", request.Folder, request.NewSlug)
	return htmx.SetRedirect(w, target)
}

func (h *renameDocumentHTMX) fields(r *http.Request, user users.Identity) *sitecontent.RenameDocumentRequest {
	var label string
	forms.MustParse(r, forms.Schema{
		"label": forms.String(&label),
	})
	label = strings.TrimSpace(label)
	folder, document := h.folderDocumentFrom(r)
	return &sitecontent.RenameDocumentRequest{
		User:     user,
		Folder:   folder,
		Document: document,
		Label:    label,
		NewSlug:  sitecontent.Slugify(label),
	}
}

func (h *renameDocumentHTMX) folderDocumentFrom(r *http.Request) (folder, document string) {
	urlpath.MustParse(r, urlpath.Schema{
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
	})
	return folder, document
}
