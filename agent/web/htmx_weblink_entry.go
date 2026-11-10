// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"errors"
	"net/http"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/urlpath"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/ui"
	"github.com/shoenig/go-conceal"
)

type weblinkEntryHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *weblinkEntryHTMX) Path() string {
	return "/internal/weblink/{action}/{folder}/{document}/{weblink}"
}

func (h *weblinkEntryHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPut, http.MethodDelete}
}

func (h *weblinkEntryHTMX) Consumer() consumer {
	return component
}

func (h *weblinkEntryHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:    h.get,
			http.MethodPut:    h.put,
			http.MethodDelete: h.delete,
		},
	)
}

// get renders the inline edit form for a single weblink. The server decrypts
// the URL using the vault cookie from the request, then renders the plaintext
// directly into the form HTML so the secret never travels through the URL.
func (h *weblinkEntryHTMX) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document, weblinkID, action := h.from(r)

	if action != "edit" {
		h.log.W.Fmt("rejecting unknown weblink action %q", action)
		return abort(h.log, w, errors.New("no such weblink action"))
	}

	vault, verr := getVault(r)
	if verr != nil {
		return htmx.SetRedirect(w, "/login")
	}

	dbWeblink, gerr := h.store.Documents().GetWeblink(ctx, &sitecontent.GetWeblinkRequest{
		ID:       weblinkID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	})
	if gerr != nil {
		return abort(h.log, w, gerr)
	}
	if dbWeblink == nil || dbWeblink.ID != weblinkID {
		return abort(h.log, w, errors.New("record not found"))
	}

	fields := struct {
		ID       records.WeblinksAuto
		URL      string
		Folder   string
		Document string
	}{
		ID:       weblinkID,
		URL:      dbWeblink.Unlock(vault),
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "weblink_edit_form.html",
		Fields:   fields,
	})
}

// put updates the URL of a single weblink and returns the rendered display
// row markup, which htmx outerHTML-swaps into the .weblink-row slot.
func (h *weblinkEntryHTMX) put(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document, weblinkID, action := h.from(r)

	if action != "update" {
		h.log.W.Fmt("rejecting unknown weblink action %q", action)
		return abort(h.log, w, errors.New("no such weblink action"))
	}

	vault, verr := getVault(r)
	if verr != nil {
		return htmx.SetRedirect(w, "/login")
	}

	request := &sitecontent.UpdateWeblinkRequest{
		ID:       weblinkID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
		URL:      h.urlFrom(r),
	}

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	ciphertext, cerr := vault.Encrypt(conceal.New(request.URL))
	if cerr != nil {
		return abort(h.log, w, cerr)
	}
	request.Encrypted = ciphertext

	if uerr := h.store.Documents().UpdateWeblink(ctx, request); uerr != nil {
		return abort(h.log, w, uerr)
	}

	fields := struct {
		ID       records.WeblinksAuto
		URL      string
		Folder   string
		Document string
	}{
		ID:       weblinkID,
		URL:      request.URL,
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "weblink_row.html",
		Fields:   fields,
	})
}

// delete removes a single weblink row. Returns an empty body so htmx's
// hx-swap="delete" can drop the wrapping .weblink-row from the DOM.
func (h *weblinkEntryHTMX) delete(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document, weblinkID, action := h.from(r)

	if action != "delete" {
		h.log.W.Fmt("rejecting unknown weblink action %q", action)
		return abort(h.log, w, errors.New("no such weblink action"))
	}

	request := &sitecontent.DeleteWeblinkRequest{
		ID:       weblinkID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	}

	if err := request.Validate(); err != nil {
		return abort(h.log, w, err)
	}

	if derr := h.store.Documents().DeleteWeblink(ctx, request); derr != nil {
		return abort(h.log, w, derr)
	}

	return nil
}

// from extracts the four trailing path segments. The action segment
// distinguishes edit / update / delete and is dispatched manually rather
// than by HTTP method alone, so the URL is human-readable.
func (h *weblinkEntryHTMX) from(r *http.Request) (folder, document string, weblink records.WeblinksAuto, action string) {
	urlpath.MustParse(r, urlpath.Schema{
		"action":   urlpath.String(&action),
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
		"weblink":  urlpath.Int(&weblink),
	})
	return folder, document, weblink, action
}

// urlFrom reads the "url" form value from a PUT body.
func (h *weblinkEntryHTMX) urlFrom(r *http.Request) string {
	var raw string
	forms.MustParse(r, forms.Schema{
		"url": forms.String(&raw),
	})
	return raw
}
