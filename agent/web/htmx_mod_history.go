// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/urlpath"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/ui"
	"github.com/shoenig/lang"
)

// modHistoryHTMX serves the login history of a single user to the moderation
// page. It is protected by the moderation gate, and unlike the profile history
// endpoint, reads the target user identity from the request path.
type modHistoryHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *modHistoryHTMX) Path() string {
	return "/moderate/users/{id}/history"
}

func (h *modHistoryHTMX) Methods() []string {
	return []string{http.MethodGet}
}

func (h *modHistoryHTMX) Consumer() consumer {
	return moderate
}

func (h *modHistoryHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *modHistoryHTMX) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	var id users.Identity
	urlpath.MustParse(r, urlpath.Schema{
		"id": urlpath.Int(&id),
	})

	entries, aerr := h.store.Audit().History(ctx, id, historySize)
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
