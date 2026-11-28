// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http/httptest"
	"testing"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func TestTools_ServeHTTP(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/tools", &toolsHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		html:        ui.Load("tools.html"),
	})

	request := requestGet(t, "/tools", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<div class="middle tools">`)
}

func TestTools_ServeHTTP_ListsExportTool(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/tools", &toolsHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		html:        ui.Load("tools.html"),
	})

	request := requestGet(t, "/tools", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `href="/tools/export"`)
	must.StrContains(t, body, `Export a backup`)
	must.StrContains(t, body, `href="/tools/share"`)
	must.StrContains(t, body, `Share a secret`)
}

func TestTools_ServeHTTP_NavLinks(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/tools", &toolsHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		html:        ui.Load("tools.html"),
	})

	request := requestGet(t, "/tools", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<a href="/tools">Tools</a>`)
	must.StrContains(t, body, `<a href="/vaults">Vaults</a>`)
	must.StrNotContains(t, body, `<a href="/export">Export</a>`)
	must.StrNotContains(t, body, `<a href="/share">Share</a>`)
}
