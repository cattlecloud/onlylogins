// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"errors"
	"html/template"
	"net/http"
	"path"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/urlpath"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/ui"
	"github.com/shoenig/go-conceal"
)

type passwordHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *passwordHTMX) Path() string {
	return "/internal/password/{action}/{folder}/{document}"
}

func (h *passwordHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPost}
}

func (h *passwordHTMX) Consumer() consumer {
	return component
}

func (h *passwordHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:  h.get,
			http.MethodPost: h.post,
		},
	)
}

func (h *passwordHTMX) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document, action := h.from(r)

	if err := (&sitecontent.GetDocumentRequest{
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	}).Validate(); err != nil {
		return abort(h.log, w, err)
	}

	switch action {
	case "add":
		return h.add(w, folder, document)
	case "list":
		return h.list(ctx, w, r, session.Identity(), folder, document)
	default:
		h.log.W.Fmt("rejecting unknown password action %q", action)
		return abort(h.log, w, errors.New("no such password action"))
	}
}

// add renders the inline form used to create a new password.
func (h *passwordHTMX) add(w http.ResponseWriter, folder, document string) error {
	fields := struct {
		Folder   string
		Document string
	}{
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "password_form.html",
		Fields:   fields,
	})
}

// list renders the passwords for a document as masked rows with copy and show
// actions. The values are decrypted on demand in the template, so the folder
// page never decrypts password values.
func (h *passwordHTMX) list(ctx scope.C, w http.ResponseWriter, r *http.Request, user users.Identity, folder, document string) error {
	cypher, verr := getCypher(r)
	if verr != nil {
		return htmx.SetRedirect(w, "/login")
	}

	request := &sitecontent.ListPasswordsRequest{
		User:     user,
		Folder:   folder,
		Document: document,
	}
	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	passwords, perr := h.store.Documents().ListPasswords(ctx, request)
	if perr != nil {
		return abort(h.log, w, perr)
	}

	fields := struct {
		Passwords []*sitecontent.Password
		Cypher    encrypt.Cypher
	}{
		Passwords: passwords,
		Cypher:    cypher,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "password_list.html",
		Fields:   fields,
		Funcs:    template.FuncMap{"Mask": format.Mask},
	})
}

func (h *passwordHTMX) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document := h.folderDocumentFrom(r)

	cypher, verr := getCypher(r)
	if verr != nil {
		return htmx.SetRedirect(w, "/login")
	}

	password := h.passwordFrom(r)

	if err := sitecontent.ValidatePassword(password); err != nil {
		return problem(w, err)
	}

	ciphertext, cerr := cypher.Encrypt(conceal.New(password))
	if cerr != nil {
		return abort(h.log, w, cerr)
	}

	request := &sitecontent.CreatePasswordRequest{
		User:      session.Identity(),
		Folder:    folder,
		Document:  document,
		Encrypted: ciphertext,
	}

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if _, ierr := h.store.Documents().CreatePassword(ctx, request); ierr != nil {
		return abort(h.log, w, ierr)
	}

	// send user back to the document page (just reload)
	target := path.Join("/folders", folder, document)
	return htmx.SetRedirect(w, target)
}

func (h *passwordHTMX) folderDocumentFrom(r *http.Request) (string, string) {
	var folder, document string
	urlpath.MustParse(r, urlpath.Schema{
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
	})
	return folder, document
}

// from extracts the action, folder, and document path segments.
func (h *passwordHTMX) from(r *http.Request) (folder, document, action string) {
	urlpath.MustParse(r, urlpath.Schema{
		"action":   urlpath.String(&action),
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
	})
	return folder, document, action
}

func (h *passwordHTMX) passwordFrom(r *http.Request) string {
	var password string
	forms.MustParse(r, forms.Schema{
		"password": forms.String(&password),
	})
	return password
}
