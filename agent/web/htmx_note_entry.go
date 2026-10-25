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

type noteEntryHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *noteEntryHTMX) Path() string {
	return "/internal/note/{action}/{folder}/{document}/{note}"
}

func (h *noteEntryHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPut, http.MethodDelete}
}

func (h *noteEntryHTMX) Consumer() consumer {
	return component
}

func (h *noteEntryHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (h *noteEntryHTMX) get(_ scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	folder, document, noteID, action := h.from(r)

	if action != "edit" {
		h.log.W.Fmt("rejecting unknown note action %q", action)
		return abort(h.log, w, errors.New("no such note action"))
	}

	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	vault, verr := getVault(r)
	if verr != nil {
		return abort(h.log, w, verr)
	}

	dbNote, gerr := h.store.Documents().GetNote(ctx, &sitecontent.GetNoteRequest{
		User: session.Identity(), Folder: folder, Document: document,
	})
	if gerr != nil {
		return abort(h.log, w, gerr)
	}
	if dbNote == nil {
		return abort(h.log, w, database.ErrDocumentNotFound)
	}

	if dbNote.ID != noteID {
		return abort(h.log, w, database.ErrDocumentNotFound)
	}

	current := dbNote.Unlock(vault)

	fields := struct {
		ID       records.NotesAuto
		Note     string
		Folder   string
		Document string
	}{
		ID:       noteID,
		Note:     current,
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "note_edit_form.html",
		Fields:   fields,
	})
}

func (h *noteEntryHTMX) put(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	folder, document, noteID, action := h.from(r)

	if action != "update" {
		h.log.W.Fmt("rejecting unknown note action %q", action)
		return abort(h.log, w, errors.New("no such note action"))
	}

	vault, verr := getVault(r)
	if verr != nil {
		return abort(h.log, w, verr)
	}

	raw := h.noteFromForm(r)

	if err := sitecontent.ValidateNote(raw); err != nil {
		return problem(w, err)
	}

	ciphertext, cerr := vault.Encrypt(conceal.New(raw))
	if cerr != nil {
		return abort(h.log, w, cerr)
	}

	request := &sitecontent.UpdateNoteRequest{
		ID:        noteID,
		User:      session.Identity(),
		Folder:    folder,
		Document:  document,
		Encrypted: ciphertext,
	}

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if uerr := h.store.Documents().UpdateNote(ctx, request); uerr != nil {
		if errors.Is(uerr, database.ErrDocumentNotFound) {
			return abort(h.log, w, uerr)
		}
		return problem(w, uerr)
	}

	fields := struct {
		ID       records.NotesAuto
		Note     string
		Folder   string
		Document string
	}{
		ID:       noteID,
		Note:     raw,
		Folder:   folder,
		Document: document,
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "note_panel.html",
		Fields:   fields,
	})
}

func (h *noteEntryHTMX) delete(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	folder, document, noteID, action := h.from(r)

	if action != "delete" {
		h.log.W.Fmt("rejecting unknown note action %q", action)
		return abort(h.log, w, errors.New("no such note action"))
	}

	request := &sitecontent.DeleteNoteRequest{
		ID:       noteID,
		User:     session.Identity(),
		Folder:   folder,
		Document: document,
	}

	if err := request.Validate(); err != nil {
		return abort(h.log, w, err)
	}

	if derr := h.store.Documents().DeleteNote(ctx, request); derr != nil {
		if errors.Is(derr, database.ErrDocumentNotFound) {
			return abort(h.log, w, derr)
		}
		return problem(w, derr)
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
		Filename: "note_add_btn_oob.html",
		Fields:   fields,
	})
}

func (h *noteEntryHTMX) from(r *http.Request) (folder, document string, note records.NotesAuto, action string) {
	urlpath.MustParse(r, urlpath.Schema{
		"action":   urlpath.String(&action),
		"folder":   urlpath.String(&folder),
		"document": urlpath.String(&document),
		"note":     urlpath.Int(&note),
	})
	return folder, document, note, action
}

func (h *noteEntryHTMX) noteFromForm(r *http.Request) string {
	var note string
	forms.MustParse(r, forms.Schema{
		"note": forms.String(&note),
	})
	return note
}
