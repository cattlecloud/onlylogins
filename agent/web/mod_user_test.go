// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http/httptest"
	"testing"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func TestModUser_ServeHTTP(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/moderate/user/{id}", &modUserHTTP{
		log:   ulog.New("test"),
		store: store,
		html:  ui.Mod("mod_user.html"),
	})

	request := requestGet(t, "/moderate/user/1", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<div class="modpage user">`)
	must.StrContains(t, body, `hx-get="/moderate/users/1/history"`)
	must.StrContains(t, body, `id="history-panel"`)
}
