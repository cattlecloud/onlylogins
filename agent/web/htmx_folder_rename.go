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
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type renameFolderHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *renameFolderHTMX) Path() string {
	return "/internal/folder/rename/{folder}"
}

func (h *renameFolderHTMX) Methods() []string {
	return []string{http.MethodPost}
}

func (h *renameFolderHTMX) Consumer() consumer {
	return component
}

func (h *renameFolderHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodPost: h.post,
		},
	)
}

func (h *renameFolderHTMX) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	request := h.fields(r, session.Identity())

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if rerr := h.store.Folders().Rename(ctx, request); rerr != nil {
		if errors.Is(rerr, database.ErrFolderExists) {
			return problem(w, database.ErrFolderExists)
		}
		return abort(h.log, w, rerr)
	}

	// the folder URL changed; send the user to the new slug
	target := path.Join("/folders", request.NewSlug)
	return htmx.SetRedirect(w, target)
}

func (h *renameFolderHTMX) fields(r *http.Request, user users.Identity) *sitecontent.RenameFolderRequest {
	var label string
	forms.MustParse(r, forms.Schema{
		"label": forms.String(&label),
	})
	label = strings.TrimSpace(label)
	return &sitecontent.RenameFolderRequest{
		User:    user,
		Slug:    h.folder(r),
		Label:   label,
		NewSlug: sitecontent.Slugify(label),
	}
}

func (h *renameFolderHTMX) folder(r *http.Request) string {
	var folder string
	urlpath.MustParse(r, urlpath.Schema{
		"folder": urlpath.String(&folder),
	})
	return folder
}
