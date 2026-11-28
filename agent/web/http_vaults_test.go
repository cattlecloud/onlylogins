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

func TestVaults_ServeHTTP(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/vaults", &vaultsHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		html:        ui.Load("vaults.html"),
	})

	request := requestGet(t, "/vaults", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<div class="middle vaults">`)
	must.StrContains(t, body, `<h1>Vaults</h1>`)
	must.StrContains(t, body, `We are building Vaults.`)
}

func TestVaults_ServeHTTP_NavLinks(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/vaults", &vaultsHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		html:        ui.Load("vaults.html"),
	})

	request := requestGet(t, "/vaults", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<a href="/vaults">Vaults</a>`)
	must.StrContains(t, body, `<a href="/tools">Tools</a>`)
	must.StrNotContains(t, body, `<a href="/share">Share</a>`)
}
