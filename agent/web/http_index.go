// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"html/template"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/agent/web/pages"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type indexContent struct {
	Profile *users.Profile
	Folders []*sitecontent.Folder
}

type indexHTTP struct {
	log   *ulog.Log
	store database.Store
	html  *template.Template
}

func (h *indexHTTP) Path() string {
	return "/"
}

func (h *indexHTTP) Methods() []string {
	return []string{http.MethodGet, http.MethodHead}
}

func (h *indexHTTP) Metric() string {
	return "index"
}

func (h *indexHTTP) Consumer() consumer {
	return normal
}

func (h *indexHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodHead: h.head,
			http.MethodGet:  h.get,
		},
	)
}

func (h *indexHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return abort(h.log, w, perr)
	}

	head := &pages.Head{
		Title:       format.PageTitle("Open Source Password Manager"),
		Styles:      []string{"index.css"},
		Description: h.description(),
		Profile:     profile,
	}

	middle := &indexContent{
		Profile: profile,
	}

	if session.Active() {
		middle.Folders, _ = h.store.Folders().List(ctx, session.Identity())
	}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, middle, nil).Execute(w, session)
}

func (h *indexHTTP) head(_ scope.C, w http.ResponseWriter, _ *http.Request) error {
	// our HEAD response simply returns HTTP StatusOK, so that we satisfy
	// the UptimeRobot crawler contract (https://uptimerobot.com)
	w.WriteHeader(http.StatusOK)
	return nil
}

func (h *indexHTTP) description() string {
	const text = `
	Store login information like usernames, passwords, backup codes, etc.`
	return format.PageDescription(text)
}
