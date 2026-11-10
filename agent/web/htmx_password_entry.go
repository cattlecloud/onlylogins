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

type passwordEntryHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *passwordEntryHTMX) Path() string {
	return "/internal/password/{action}/{folder}/{document}/{password}"
}

func (h *passwordEntryHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPut, http.MethodDelete}
}

func (h *passwordEntryHTMX) Consumer() consumer {
	return component
}

func (h *passwordEntryHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (h *passwordEntryHTMX) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document, passwordID, action := h.from(r)

	if action != "edit" {
		h.log.W.Fmt("rejecting unknown password action %q", action)
		return abort(h.log, w, errors.New("no such password action"))
	}

	vault, verr := getVault(r)
	if verr != nil {
		return htmx.SetRedirect(w, "/login")
	}

	dbPassword, gerr := h.store.Documents().GetPassword(ctx, &sitecontent.GetPasswordRequest{
		ID:       passwordID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	})
	if gerr != nil {
		return abort(h.log, w, gerr)
	}
	if dbPassword == nil || dbPassword.ID != passwordID {
		return abort(h.log, w, errors.New("record not found"))
	}

	fields := struct {
		ID       records.PasswordsAuto
		Password string
		Folder   string
		Document string
	}{
		ID:       passwordID,
		Password: dbPassword.Unlock(vault),
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "password_edit_form.html",
		Fields:   fields,
	})
}

func (h *passwordEntryHTMX) put(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document, passwordID, action := h.from(r)

	if action != "update" {
		h.log.W.Fmt("rejecting unknown password action %q", action)
		return abort(h.log, w, errors.New("no such password action"))
	}

	vault, verr := getVault(r)
	if verr != nil {
		return htmx.SetRedirect(w, "/login")
	}

	password := h.passwordFromForm(r)

	if err := sitecontent.ValidatePassword(password); err != nil {
		return problem(w, err)
	}

	ciphertext, cerr := vault.Encrypt(conceal.New(password))
	if cerr != nil {
		return abort(h.log, w, cerr)
	}

	request := &sitecontent.UpdatePasswordRequest{
		ID:        passwordID,
		User:      session.Identity(),
		Folder:    folder,
		Document:  document,
		Encrypted: ciphertext,
	}

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if uerr := h.store.Documents().UpdatePassword(ctx, request); uerr != nil {
		return abort(h.log, w, uerr)
	}

	fields := struct {
		ID       records.PasswordsAuto
		Password string
		Folder   string
		Document string
	}{
		ID:       passwordID,
		Password: password,
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "password_row.html",
		Fields:   fields,
	})
}

func (h *passwordEntryHTMX) delete(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document, passwordID, action := h.from(r)

	if action != "delete" {
		h.log.W.Fmt("rejecting unknown password action %q", action)
		return abort(h.log, w, errors.New("no such password action"))
	}

	request := &sitecontent.DeletePasswordRequest{
		ID:       passwordID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	}

	if err := request.Validate(); err != nil {
		return abort(h.log, w, err)
	}

	if derr := h.store.Documents().DeletePassword(ctx, request); derr != nil {
		return abort(h.log, w, derr)
	}

	return nil
}

func (h *passwordEntryHTMX) from(r *http.Request) (folder, document string, password records.PasswordsAuto, action string) {
	urlpath.MustParse(r, urlpath.Schema{
		"action":   urlpath.String(&action),
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
		"password": urlpath.Int(&password),
	})
	return folder, document, password, action
}

func (h *passwordEntryHTMX) passwordFromForm(r *http.Request) string {
	var raw string
	forms.MustParse(r, forms.Schema{
		"password": forms.String(&raw),
	})
	return raw
}
