// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

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

// usernameEntryHTMX handles the per-row edit / update / delete actions for
// an individual username. It mirrors weblinkEntryHTMX with the same path
// layout and dispatch-by-action-segment approach.
type usernameEntryHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *usernameEntryHTMX) Path() string {
	return "/internal/username/{action}/{folder}/{document}/{username}"
}

func (h *usernameEntryHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPut, http.MethodDelete}
}

func (h *usernameEntryHTMX) Consumer() consumer {
	return component
}

func (h *usernameEntryHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

// get renders the inline edit form for a single username. The page already
// has the decrypted plaintext in scope (via the Vault cookie), so we pass
// it through a query parameter; the server does not re-decrypt.
func (h *usernameEntryHTMX) get(_ scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	folder, document, usernameID, action := h.from(r)

	if action != "edit" {
		h.log.W.Fmt("rejecting unknown username action %q", action)
		return abort(h.log, w, errors.New("no such username action"))
	}

	current := h.usernameFromQuery(r)

	fields := struct {
		ID       records.UsernamesAuto
		Username string
		Folder   string
		Document string
	}{
		ID:       usernameID,
		Username: current,
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "username_edit_form.html",
		Fields:   fields,
	})
}

// put updates a single username and returns the rendered display row markup,
// which htmx outerHTML-swaps into the .username-row slot.
func (h *usernameEntryHTMX) put(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	folder, document, usernameID, action := h.from(r)

	if action != "update" {
		h.log.W.Fmt("rejecting unknown username action %q", action)
		return abort(h.log, w, errors.New("no such username action"))
	}

	vault, verr := getVault(r)
	if verr != nil {
		return abort(h.log, w, verr)
	}

	request := &sitecontent.UpdateUsernameRequest{
		ID:       usernameID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
		Username: h.usernameFrom(r),
	}

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	ciphertext, cerr := vault.Encrypt(conceal.New(request.Username))
	if cerr != nil {
		return abort(h.log, w, cerr)
	}
	request.Encrypted = ciphertext

	if uerr := h.store.Documents().UpdateUsername(ctx, request); uerr != nil {
		if errors.Is(uerr, database.ErrDocumentNotFound) {
			return abort(h.log, w, uerr)
		}
		return problem(w, uerr)
	}

	fields := struct {
		ID       records.UsernamesAuto
		Username string
		Folder   string
		Document string
	}{
		ID:       usernameID,
		Username: request.Username,
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "username_row.html",
		Fields:   fields,
	})
}

// delete removes a single username row. Returns an empty body so htmx's
// hx-swap="delete" can drop the wrapping .username-row from the DOM.
func (h *usernameEntryHTMX) delete(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	folder, document, usernameID, action := h.from(r)

	if action != "delete" {
		h.log.W.Fmt("rejecting unknown username action %q", action)
		return abort(h.log, w, errors.New("no such username action"))
	}

	request := &sitecontent.DeleteUsernameRequest{
		ID:       usernameID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	}

	if err := request.Validate(); err != nil {
		return abort(h.log, w, err)
	}

	if derr := h.store.Documents().DeleteUsername(ctx, request); derr != nil {
		if errors.Is(derr, database.ErrDocumentNotFound) {
			return abort(h.log, w, derr)
		}
		return problem(w, derr)
	}

	return nil
}

// from extracts the four trailing path segments. The action segment
// distinguishes edit / update / delete and is dispatched manually rather
// than by HTTP method alone, so the URL is human-readable.
func (h *usernameEntryHTMX) from(r *http.Request) (folder, document string, username records.UsernamesAuto, action string) {
	urlpath.MustParse(r, urlpath.Schema{
		"action":   urlpath.String(&action),
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
		"username": urlpath.Int(&username),
	})
	return folder, document, username, action
}

// usernameFrom reads the "username" form value from a PUT body.
func (h *usernameEntryHTMX) usernameFrom(r *http.Request) string {
	var raw string
	forms.MustParse(r, forms.Schema{
		"username": forms.String(&raw),
	})
	return raw
}

// usernameFromQuery reads a "username" query parameter from a GET.
func (h *usernameEntryHTMX) usernameFromQuery(r *http.Request) string {
	return r.URL.Query().Get("username")
}
