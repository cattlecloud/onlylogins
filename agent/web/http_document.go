// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"errors"
	"html/template"
	"net/http"
	"path"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/urlpath"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/agent/web/pages"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type documentContent struct {
	Document *sitecontent.Document
	Vault    encrypt.Vault
}

type documentHTTP struct {
	log         *ulog.Log
	store       database.Store
	environment runtime.Environment
	html        *template.Template
}

func (h *documentHTTP) Path() string {
	return "/folders/{folder}/{document}"
}

func (h *documentHTTP) Methods() []string {
	return []string{http.MethodGet, http.MethodDelete}
}

func (h *documentHTTP) Metric() string {
	return "document"
}

func (h *documentHTTP) Consumer() consumer {
	return normal
}

func (h *documentHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:    h.get,
			http.MethodDelete: h.delete,
		},
	)
}

func (h *documentHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if user is not logged in, redirect to login page
	if !session.Active() {
		return login(h.log, w, r)
	}

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return abort(h.log, w, perr)
	}

	request := h.document(r, session.Identity())

	if err := request.Validate(); err != nil {
		return abort(h.log, w, err)
	}

	document, derr := h.store.Documents().Get(ctx, request)
	if derr != nil {
		return abort(h.log, w, derr)
	}

	vault, verr := getVault(r)
	if verr != nil {
		return login(h.log, w, r)
	}

	head := &pages.Head{
		Title:       format.PageTitle(document.Label),
		Styles:      []string{"document.css"},
		Description: h.description(),
		Profile:     profile,
	}

	middle := &documentContent{
		Document: document,
		Vault:    vault,
	}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, middle).Execute(w, session)
}

func (h *documentHTTP) delete(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	user := session.Identity()
	folder, document := h.fromPath(r)

	// turn the folder and document labels into a document id
	get := &sitecontent.GetDocumentRequest{
		User:     user,
		Folder:   folder,
		Document: document,
	}
	id, gerr := h.store.Documents().Resolve(ctx, get)
	if gerr != nil {
		return abort(h.log, w, errors.New("document not found"))
	}

	request := &sitecontent.DeleteDocumentRequest{
		ID:   id,
		User: user,
	}
	if verr := request.Validate(); verr != nil {
		return problem(w, verr)
	}

	if derr := h.store.Documents().Delete(ctx, request); derr != nil {
		return abort(h.log, w, derr)
	}

	target := path.Join("/folders", folder)
	return htmx.SetRedirect(w, target)
}

func (h *documentHTTP) document(r *http.Request, user users.Identity) *sitecontent.GetDocumentRequest {
	folder, document := h.fromPath(r)
	return &sitecontent.GetDocumentRequest{
		User:     user,
		Folder:   folder,
		Document: document,
	}
}

func (h *documentHTTP) fromPath(r *http.Request) (folder, document string) {
	urlpath.MustParse(r, urlpath.Schema{
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
	})
	return folder, document
}

func (h *documentHTTP) description() string {
	const text = `
	Mange contents of a document`
	return format.PageDescription(text)
}
