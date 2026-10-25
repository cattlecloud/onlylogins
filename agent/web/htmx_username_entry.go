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

func (h *usernameEntryHTMX) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	folder, document, usernameID, action := h.from(r)

	if action != "edit" {
		h.log.W.Fmt("rejecting unknown username action %q", action)
		return abort(h.log, w, errors.New("no such username action"))
	}

	vault, verr := getVault(r)
	if verr != nil {
		return abort(h.log, w, verr)
	}

	dbUsername, gerr := h.store.Documents().GetUsername(ctx, &sitecontent.GetUsernameRequest{
		ID:       usernameID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	})
	if gerr != nil {
		if errors.Is(gerr, database.ErrDocumentNotFound) {
			return abort(h.log, w, gerr)
		}
		return abort(h.log, w, gerr)
	}
	if dbUsername == nil || dbUsername.ID != usernameID {
		return abort(h.log, w, database.ErrDocumentNotFound)
	}

	fields := struct {
		ID       records.UsernamesAuto
		Username string
		Folder   string
		Document string
	}{
		ID:       usernameID,
		Username: dbUsername.Unlock(vault),
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "username_edit_form.html",
		Fields:   fields,
	})
}

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

func (h *usernameEntryHTMX) from(r *http.Request) (folder, document string, username records.UsernamesAuto, action string) {
	urlpath.MustParse(r, urlpath.Schema{
		"action":   urlpath.String(&action),
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
		"username": urlpath.Int(&username),
	})
	return folder, document, username, action
}

func (h *usernameEntryHTMX) usernameFrom(r *http.Request) string {
	var raw string
	forms.MustParse(r, forms.Schema{
		"username": forms.String(&raw),
	})
	return raw
}
