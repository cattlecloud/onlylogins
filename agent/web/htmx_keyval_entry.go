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

type keyvalEntryHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *keyvalEntryHTMX) Path() string {
	return "/internal/keyval/{action}/{folder}/{document}/{keyval}"
}

func (h *keyvalEntryHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPut, http.MethodDelete}
}

func (h *keyvalEntryHTMX) Consumer() consumer {
	return component
}

func (h *keyvalEntryHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (h *keyvalEntryHTMX) get(_ scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document, keyvalID, action := h.from(r)

	if action != "edit" {
		h.log.W.Fmt("rejecting unknown keyval action %q", action)
		return abort(h.log, w, errors.New("no such keyval action"))
	}

	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	vault, verr := getVault(r)
	if verr != nil {
		return htmx.SetRedirect(w, "/login")
	}

	kv, kverr := h.store.Documents().GetKeyval(ctx, &sitecontent.GetKeyvalRequest{
		ID:       keyvalID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	})
	if kverr != nil {
		return abort(h.log, w, kverr)
	}
	if kv == nil || kv.ID != keyvalID {
		return abort(h.log, w, errors.New("kv not found"))
	}

	fields := struct {
		ID       records.KeyvalsAuto
		Key      string
		Value    string
		Folder   string
		Document string
	}{
		ID:       keyvalID,
		Key:      kv.K.Unveil(),
		Value:    kv.Unlock(vault),
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "keyval_edit_form.html",
		Fields:   fields,
	})
}

func (h *keyvalEntryHTMX) put(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document, keyvalID, action := h.from(r)

	if action != "update" {
		h.log.W.Fmt("rejecting unknown keyval action %q", action)
		return abort(h.log, w, errors.New("no such keyval action"))
	}

	vault, verr := getVault(r)
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
	ciphertext, cerr := vault.Encrypt(conceal.New(value))
	if cerr != nil {
		return abort(h.log, w, cerr)
	}

	request := &sitecontent.UpdateKeyvalRequest{
		ID:        keyvalID,
		User:      session.Identity(),
		Folder:    folder,
		Document:  document,
		Key:       key,
		Encrypted: ciphertext,
	}

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if uerr := h.store.Documents().UpdateKeyval(ctx, request); uerr != nil {
		return abort(h.log, w, uerr)
	}

	fields := struct {
		ID       records.KeyvalsAuto
		Key      string
		Value    string
		Folder   string
		Document string
	}{
		ID:       keyvalID,
		Key:      request.Key,
		Value:    value,
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "keyval_row.html",
		Fields:   fields,
	})
}

func (h *keyvalEntryHTMX) delete(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	folder, document, keyvalID, action := h.from(r)

	if action != "delete" {
		h.log.W.Fmt("rejecting unknown keyval action %q", action)
		return abort(h.log, w, errors.New("no such keyval action"))
	}

	request := &sitecontent.DeleteKeyvalRequest{
		ID:       keyvalID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	}

	if err := request.Validate(); err != nil {
		return abort(h.log, w, err)
	}

	if derr := h.store.Documents().DeleteKeyval(ctx, request); derr != nil {
		return abort(h.log, w, derr)
	}

	return nil
}

func (h *keyvalEntryHTMX) from(r *http.Request) (folder, document string, keyval records.KeyvalsAuto, action string) {
	urlpath.MustParse(r, urlpath.Schema{
		"action":   urlpath.String(&action),
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
		"keyval":   urlpath.Int(&keyval),
	})
	return folder, document, keyval, action
}

func (h *keyvalEntryHTMX) kvFromForm(r *http.Request) (string, string) {
	var key, value string
	forms.MustParse(r, forms.Schema{
		"key":   forms.String(&key),
		"value": forms.String(&value),
	})
	return key, value
}
