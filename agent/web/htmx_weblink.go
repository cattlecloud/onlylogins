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
	"cattlecloud.net/onlylogins/pkg/encrypt"
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

func (h *weblinkHTMX) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
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
		h.log.W.Fmt("rejecting unknown weblink action %q", action)
		return abort(h.log, w, errors.New("no such weblink action"))
	}
}

// add renders the inline form used to create a new weblink.
func (h *weblinkHTMX) add(w http.ResponseWriter, folder, document string) error {
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

// list renders the decrypted weblinks for a document as a read-only list. The
// URLs are decrypted on demand using the cypher cookie, so the folder page never
// decrypts weblink values.
func (h *weblinkHTMX) list(ctx scope.C, w http.ResponseWriter, r *http.Request, user users.Identity, folder, document string) error {
	cypher, verr := getCypher(r)
	if verr != nil {
		return htmx.SetRedirect(w, "/login")
	}

	request := &sitecontent.ListWeblinksRequest{
		User:     user,
		Folder:   folder,
		Document: document,
	}
	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	weblinks, werr := h.store.Documents().ListWeblinks(ctx, request)
	if werr != nil {
		return abort(h.log, w, werr)
	}

	fields := struct {
		Weblinks []*sitecontent.Weblink
		Cypher   encrypt.Cypher
	}{
		Weblinks: weblinks,
		Cypher:   cypher,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "weblink_list.html",
		Fields:   fields,
	})
}

func (h *weblinkHTMX) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document := h.folderDocumentFrom(r)

	cypher, verr := getCypher(r)
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

	ciphertext, cerr := cypher.Encrypt(conceal.New(request.URL))
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

// from extracts the action, folder, and document path segments.
func (h *weblinkHTMX) from(r *http.Request) (folder, document, action string) {
	urlpath.MustParse(r, urlpath.Schema{
		"action":   urlpath.String(&action),
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
	})
	return folder, document, action
}

func (h *weblinkHTMX) urlFrom(r *http.Request) string {
	var raw string
	forms.MustParse(r, forms.Schema{
		"url": forms.String(&raw),
	})
	return strings.TrimSpace(raw)
}
