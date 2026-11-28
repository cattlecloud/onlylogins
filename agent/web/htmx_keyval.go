// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

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

type keyvalHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *keyvalHTMX) Path() string {
	return "/internal/keyval/{action}/{folder}/{document}"
}

func (h *keyvalHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPost}
}

func (h *keyvalHTMX) Consumer() consumer {
	return component
}

func (h *keyvalHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:  h.get,
			http.MethodPost: h.post,
		},
	)
}

func (h *keyvalHTMX) get(_ scope.C, w http.ResponseWriter, r *http.Request) error {
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
		Filename: "keyval_form.html",
		Fields:   fields,
	})
}

func (h *keyvalHTMX) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document := h.folderDocumentFrom(r)

	cypher, verr := getCypher(r)
	if verr != nil {
		return htmx.SetRedirect(w, "/login")
	}

	key, value := h.kvFromForm(r)

	// validate the plaintext value before encrypting so we never materialize
	// an invalid request struct
	if err := sitecontent.ValidateKeyvalValue(value); err != nil {
		return problem(w, err)
	}

	// encrypt the value plaintext into ciphertext before constructing the
	// request, so the request struct can be a single literal
	ciphertext, cerr := cypher.Encrypt(conceal.New(value))
	if cerr != nil {
		return abort(h.log, w, cerr)
	}

	request := &sitecontent.CreateKeyvalRequest{
		User:      session.Identity(),
		Folder:    folder,
		Document:  document,
		Key:       key,
		Encrypted: ciphertext,
	}

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if _, ierr := h.store.Documents().CreateKeyval(ctx, request); ierr != nil {
		return abort(h.log, w, ierr)
	}

	// send user back to the document page (full reload)
	target := path.Join("/folders", folder, document)
	return htmx.SetRedirect(w, target)
}

func (h *keyvalHTMX) folderDocumentFrom(r *http.Request) (string, string) {
	var folder, document string
	urlpath.MustParse(r, urlpath.Schema{
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
	})
	return folder, document
}

func (h *keyvalHTMX) kvFromForm(r *http.Request) (string, string) {
	var key, value string
	forms.MustParse(r, forms.Schema{
		"key":   forms.String(&key),
		"value": forms.String(&value),
	})
	return key, value
}
