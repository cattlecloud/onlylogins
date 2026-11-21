// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/ui"
	"github.com/shoenig/lang"
)

// historySize is the limit of historical login rows
const historySize = 20

type historyHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *historyHTMX) Path() string {
	return "/internal/profile/history"
}

func (h *historyHTMX) Methods() []string {
	return []string{http.MethodGet}
}

func (h *historyHTMX) Consumer() consumer {
	return component
}

func (h *historyHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

// history is a single login attempt prepared for display
type history struct {
	IP      string
	When    string
	Outcome string
	Class   string
	Detail  string
}

func (h *historyHTMX) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if not logged in, redirect to login page
	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	entries, aerr := h.store.Audit().History(ctx, session.Identity(), historySize)
	if aerr != nil {
		return abort(h.log, w, aerr)
	}

	rows := make([]history, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, history{
			IP:      e.IP,
			When:    format.Ago(e.Creation),
			Detail:  format.UserAgent(e.UserAgent),
			Outcome: lang.Maybe(e.Success, "success", "failure"),
			Class:   lang.Maybe(e.Success, "success", "failure"),
		})
	}

	return htmx.Write(w, &htmx.Template{
		FS:       htmx.FS(ui.HTMX, "parts"),
		Filename: "history.html",
		Fields:   struct{ Rows []history }{Rows: rows},
	})
}
