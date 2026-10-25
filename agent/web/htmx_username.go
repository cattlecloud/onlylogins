// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

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
	"cattlecloud.net/onlylogins/ui"
	"github.com/shoenig/go-conceal"
)

type usernameHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *usernameHTMX) Path() string {
	return "/internal/username/{action}/{folder}/{document}"
}

func (h *usernameHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPost}
}

func (h *usernameHTMX) Consumer() consumer {
	return component
}

func (h *usernameHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:  h.get,
			http.MethodPost: h.post,
		},
	)
}

func (h *usernameHTMX) get(_ scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	folder, document := h.folderDocumentFrom(r)

	if err := (&sitecontent.GetDocumentRequest{
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	}).Validate(); err != nil {
		return abort(h.log, w, err)
	}

	fields := struct {
		Folder   string
		Document string
	}{
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "username_form.html",
		Fields:   fields,
	})
}

func (h *usernameHTMX) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	folder, document := h.folderDocumentFrom(r)

	vault, verr := getVault(r)
	if verr != nil {
		return abort(h.log, w, verr)
	}

	rawUsername := h.usernameFrom(r)

	request := &sitecontent.CreateUsernameRequest{
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
		Username: rawUsername,
	}

	// validate will trim whitespace and apply the length cap
	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	ciphertext, cerr := vault.Encrypt(conceal.New(request.Username))
	if cerr != nil {
		return abort(h.log, w, cerr)
	}
	request.Encrypted = ciphertext

	if _, ierr := h.store.Documents().CreateUsername(ctx, request); ierr != nil {
		if errors.Is(ierr, database.ErrDocumentNotFound) {
			return abort(h.log, w, ierr)
		}
		return problem(w, ierr)
	}

	// send user back to the document page (just reload)
	target := path.Join("/folders", folder, document)
	return htmx.SetRedirect(w, target)
}

func (h *usernameHTMX) folderDocumentFrom(r *http.Request) (string, string) {
	var folder, document string
	urlpath.MustParse(r, urlpath.Schema{
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
	})
	return folder, document
}

func (h *usernameHTMX) usernameFrom(r *http.Request) string {
	var raw string
	forms.MustParse(r, forms.Schema{
		"username": forms.String(&raw),
	})
	return strings.TrimSpace(raw)
}
