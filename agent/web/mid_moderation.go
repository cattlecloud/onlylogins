// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/lang"
)

// moderationGate allows access to pages only for administrators and maybe
// moderators.
type moderationGate struct {
	next    http.Handler
	council users.Council
	log     *ulog.Log
}

func (mg *moderationGate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	abort := func() {
		origins := webtools.Origins(r)
		ip := lang.Maybe(origins.Forward != "", origins.Forward, origins.Host)
		mg.log.W.Fmt("reject non-admin access to moderation page (%s) %s", ip, r.URL.Path)
		http.Error(w, "no such page", http.StatusNotFound)
	}

	session := middles.GetSession[users.Identity](r)

	// ensure a user is logged in; otherwise abort
	if !session.Active() {
		abort()
		return
	}

	// ensure the user is an administrator (or moderator); otherwise abort
	if !mg.council.IsAdmin(session.Identity()) {
		abort()
		return
	}

	// user is an administrator; allow to go through
	mg.next.ServeHTTP(w, r)
}
