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

func TestAbout_ServeHTTP(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/about", &aboutHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		html:        ui.Load("about.html"),
	})

	request := requestGet(t, "/about", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<div class="middle about">`)
	must.StrContains(t, body, `<section id="build" class="build" aria-label="Build information">`)
	must.StrContains(t, body, `System</dt>`)
	must.StrContains(t, body, `Architecture</dt>`)
	must.StrContains(t, body, `Environment</dt>`)
	must.StrContains(t, body, `local</dd>`)
}
