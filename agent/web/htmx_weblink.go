// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
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

type weblinkHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *weblinkHTMX) Path() string {
	return "/internal/weblink/{action}/{folder}/{document}"
}

func (h *weblinkHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPost}
}

func (h *weblinkHTMX) Consumer() consumer {
	return component
}

func (h *weblinkHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:  h.get,
			http.MethodPost: h.post,
		},
	)
}

func (h *weblinkHTMX) get(_ scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
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
		Filename: "weblink_form.html",
		Fields:   fields,
	})
}

func (h *weblinkHTMX) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document := h.folderDocumentFrom(r)

	vault, verr := getVault(r)
	if verr != nil {
		return htmx.SetRedirect(w, "/login")
	}

	rawURL := h.urlFrom(r)

	request := &sitecontent.CreateWeblinkRequest{
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
		URL:      rawURL,
	}

	// validate will normalize the URL if needed
	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	ciphertext, cerr := vault.Encrypt(conceal.New(request.URL))
	if cerr != nil {
		return abort(h.log, w, cerr)
	}
	request.Encrypted = ciphertext

	if _, ierr := h.store.Documents().CreateWeblink(ctx, request); ierr != nil {
		return abort(h.log, w, ierr)
	}

	// send user back to the document page (just reload)
	target := path.Join("/folders", folder, document)
	return htmx.SetRedirect(w, target)
}

func (h *weblinkHTMX) folderDocumentFrom(r *http.Request) (string, string) {
	var folder, document string
	urlpath.MustParse(r, urlpath.Schema{
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
	})
	return folder, document
}

func (h *weblinkHTMX) urlFrom(r *http.Request) string {
	var raw string
	forms.MustParse(r, forms.Schema{
		"url": forms.String(&raw),
	})
	return strings.TrimSpace(raw)
}
