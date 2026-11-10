// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/urlpath"
	"cattlecloud.net/go/webtools"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/agent/web/pages"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type shareOpenContent struct {
	UID    records.SharesUnique
	Paper  *sitecontent.Share
	IP     string
	Opened time.Time
}

type shareOpenHTTP struct {
	log   *ulog.Log
	store database.Store
	vault encrypt.Vault
	now   func() time.Time
	html  *template.Template
}

func (h *shareOpenHTTP) Path() string {
	return "/share/open/{uid}"
}

func (h *shareOpenHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *shareOpenHTTP) Metric() string {
	return "share_open"
}

func (h *shareOpenHTTP) Consumer() consumer {
	return normal
}

func (h *shareOpenHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

// get handles a recipient visiting a share link
//
// being logged in is not a requirement
func (h *shareOpenHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return abort(h.log, w, perr)
	}

	now := h.now()
	uid := h.getUID(r)
	origins := webtools.Origins(r)
	ip := origins.From()

	request := &sitecontent.ConsumeShareRequest{
		UUID: uid,
		IP:   ip,
	}

	// share will be nil if it does not exist
	share, serr := h.store.Shares(h.vault).Consume(ctx, request)
	if serr != nil && !errors.Is(serr, sql.ErrNoRows) {
		return abort(h.log, w, serr)
	}

	// ensure the share is not expired; clear it if so
	if share.Expired(now) {
		share = nil
	}

	head := &pages.Head{
		Title:       format.PageTitle("Open Share"),
		Styles:      []string{"share_open.css"},
		NoIndex:     true,
		Description: h.description(),
		Profile:     profile,
	}

	middle := &shareOpenContent{
		UID:    uid,
		Paper:  share,
		IP:     ip,
		Opened: now,
	}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, middle, nil).Execute(w, session)
}

func (h *shareOpenHTTP) getUID(r *http.Request) records.SharesUnique {
	var uid records.SharesUnique
	urlpath.MustParse(r, urlpath.Schema{
		"uid": urlpath.String(&uid),
	})
	return uid
}

func (h *shareOpenHTTP) description() string {
	const text = `
	Open a secure, encrypted Share message on OnlyLogins. Read a one-time
	secret before it self-destructs. No account needed to view.`
	return format.PageDescription(text)
}
