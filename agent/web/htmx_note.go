// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"net/http"
	"path"

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

type noteHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *noteHTMX) Path() string {
	return "/internal/note/{action}/{folder}/{document}"
}

func (h *noteHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPost}
}

func (h *noteHTMX) Consumer() consumer {
	return component
}

func (h *noteHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:  h.get,
			http.MethodPost: h.post,
		},
	)
}

func (h *noteHTMX) get(_ scope.C, w http.ResponseWriter, r *http.Request) error {
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
		Filename: "note_form.html",
		Fields:   fields,
	})
}

func (h *noteHTMX) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	folder, document := h.folderDocumentFrom(r)

	vault, verr := getVault(r)
	if verr != nil {
		return abort(h.log, w, verr)
	}

	raw := h.noteFrom(r)

	if err := sitecontent.ValidateNote(raw); err != nil {
		return problem(w, err)
	}

	ciphertext, cerr := vault.Encrypt(conceal.New(raw))
	if cerr != nil {
		return abort(h.log, w, cerr)
	}

	request := &sitecontent.CreateNoteRequest{
		User:      session.Identity(),
		Folder:    folder,
		Document:  document,
		Encrypted: ciphertext,
	}

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if _, ierr := h.store.Documents().CreateNote(ctx, request); ierr != nil {
		return abort(h.log, w, ierr)
	}

	// send user back to the document page (just reload)
	target := path.Join("/folders", folder, document)
	return htmx.SetRedirect(w, target)
}

func (h *noteHTMX) folderDocumentFrom(r *http.Request) (string, string) {
	var folder, document string
	urlpath.MustParse(r, urlpath.Schema{
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
	})
	return folder, document
}

func (h *noteHTMX) noteFrom(r *http.Request) string {
	var raw string
	forms.MustParse(r, forms.Schema{
		"note": forms.String(&raw),
	})
	return raw
}
