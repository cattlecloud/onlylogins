// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/agent/web/pages"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type exportHTTP struct {
	log         *ulog.Log
	environment runtime.Environment
	store       database.Store
	html        *template.Template
}

func (h *exportHTTP) Path() string {
	return "/export"
}

func (h *exportHTTP) Methods() []string {
	return []string{http.MethodGet, http.MethodPost}
}

func (h *exportHTTP) Metric() string {
	return "export"
}

func (h *exportHTTP) Consumer() consumer {
	return normal
}

func (h *exportHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:  h.get,
			http.MethodPost: h.post,
		},
	)
}

func (h *exportHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return perr
	}

	head := &pages.Head{
		Title:       format.PageTitle("Export"),
		Styles:      []string{"basics.css"},
		Canonical:   h.environment.Canonical("%s", h.Path()),
		Description: h.description(),
		Profile:     profile,
	}

	middle := struct{}{}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, middle, nil).Execute(w, session)
}

func (h *exportHTTP) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)
	if !session.Active() {
		return login(h.log, w, r)
	}

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return abort(h.log, w, perr)
	}

	vault, verr := getVault(r)
	if verr != nil {
		return login(h.log, w, r)
	}

	folders, ferr := h.store.Folders().List(ctx, session.Identity())
	if ferr != nil {
		return abort(h.log, w, ferr)
	}

	// inflate the object to create a json payload
	ee := &sitecontent.ExportEverything{
		Username: profile.Username,
		Folders:  make([]*sitecontent.ExportFolder, 0, len(folders)),
	}

	// iterate each folder
	for _, folder := range folders {
		ef := &sitecontent.ExportFolder{
			Name:      folder.Label,
			Documents: make([]*sitecontent.ExportDocument, 0),
		}

		doclist := &sitecontent.ListDocumentsRequest{
			User:   session.Identity(),
			Folder: folder.Label,
		}

		// iterate each document in the folder
		documents, dlerr := h.store.Documents().List(ctx, doclist)
		if dlerr != nil {
			return abort(h.log, w, dlerr)
		}

		for _, doc := range documents {
			docreq := &sitecontent.GetDocumentRequest{
				User:     session.Identity(),
				Folder:   folder.Label,
				Document: doc.Label,
			}

			// get the document and extract the data
			document, derr := h.store.Documents().Get(ctx, docreq)
			if derr != nil {
				return abort(h.log, w, derr)
			}

			ed := &sitecontent.ExportDocument{
				Name: document.Label,
			}

			for _, weblink := range document.Weblinks {
				ed.Weblinks = append(ed.Weblinks, weblink.Unlock(vault))
			}

			for _, username := range document.Usernames {
				ed.Usernames = append(ed.Usernames, username.Unlock(vault))
			}

			for _, password := range document.Passwords {
				ed.Passwords = append(ed.Passwords, password.Unlock(vault))
			}

			for _, keyval := range document.Keyvals {
				ed.Keyvals = append(ed.Keyvals, [2]string{keyval.K.Unveil(), keyval.Unlock(vault)})
			}

			if document.Note != nil {
				ed.Note = document.Note.Unlock(vault)
			}

			ef.Documents = append(ef.Documents, ed)
		}

		ee.Folders = append(ee.Folders, ef)
	}

	// create the filename with date
	today := time.Now().Format(time.DateOnly)
	filename := "onlylogins-" + today + ".json"

	// set content-type and content-disposition headers
	webtools.SetContentType(w, webtools.ContentTypeJSON)
	webtools.SetAttachment(w, filename)

	// json encode the response
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(ee)
}

func (h *exportHTTP) description() string {
	const text = `
	Download a complete backup of your vault, including every document's
	weblinks, usernames, passwords, key/value pairs, and notes, as one JSON file.`
	return format.PageDescription(text)
}
