// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

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

func TestTOS_ServeHTTP(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/terms", &tosHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		html:        ui.Load("tos.html"),
	})

	request := requestGet(t, "/terms", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<div class="middle tos">`)
}
