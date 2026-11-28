// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

// makeShareMux wires both the /tools/share and /open/{uid} endpoints into
// a fresh mux router with a session cache ready for creating test cookies.
func makeShareMux(t *testing.T, store database.Store, cypher mocks.TestCypher) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	scache := mocks.NewSessions()
	router := mux.NewRouter()

	router.Handle((&shareHTTP{}).Path(), &shareHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		cypher:      cypher.Cypher(),
		html:        ui.Load("share.html"),
	})

	router.Handle((&shareOpenHTTP{}).Path(), &shareOpenHTTP{
		log:    ulog.New("test"),
		store:  store,
		cypher: cypher.Cypher(),
		now:    time.Now,
		html:   ui.Load("share_open.html"),
	})

	return router, scache
}

func TestShare_ServeHTTP_GET(t *testing.T) {
	t.Parallel()

	router, scache := makeShareMux(t, mockdb.Open(t), mocks.NewTestCypher())
	ss := session(router, scache)

	r := requestGet(t, "/tools/share", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	body := w.Body.String()
	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, body, `<div class="middle share">`)
	// logged-out viewers should see the landing block, not the form
	must.StrContains(t, body, "Log in to start sharing")
}

func TestShare_ServeHTTP_GET_logged_in(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := makeShareMux(t, store, mocks.NewTestCypher())
	ss := session(router, scache)

	cookie := scache.Create(mocks.Nobody, 1*time.Minute)
	r := requestGet(t, "/tools/share", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	body := w.Body.String()
	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, body, `<div class="middle share">`)
	must.StrContains(t, body, "Create link to a secret message")
	must.StrContains(t, body, `hx-put="/tools/share"`)
}

func TestShare_ServeHTTP_PUT_not_logged_in(t *testing.T) {
	t.Parallel()

	router, scache := makeShareMux(t, mockdb.Open(t), mocks.NewTestCypher())
	ss := session(router, scache)

	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodPut,
		"/tools/share?content=secretvalue&ttl=900", nil)
	must.NoError(t, rerr)
	_ = r.ParseForm()

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestShare_ServeHTTP_PUT(t *testing.T) {
	t.Parallel()

	const userID = mocks.Nobody
	store := mockdb.Open(t)
	router, scache := makeShareMux(t, store, mocks.NewTestCypher())
	ss := session(router, scache)

	cookie := scache.Create(userID, 1*time.Minute)

	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodPut,
		"/tools/share?content=secretvalue&ttl=900", nil)
	must.NoError(t, rerr)
	r.AddCookie(cookie)
	_ = r.ParseForm()

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `share link`)
	must.StrContains(t, w.Body.String(), `/open/`)
}

func TestShare_ServeHTTP_PUT_empty_content(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := makeShareMux(t, store, mocks.NewTestCypher())
	ss := session(router, scache)

	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodPut,
		"/tools/share?ttl=900&content=", nil)
	must.NoError(t, rerr)
	r.AddCookie(cookie)
	_ = r.ParseForm()

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `content must be set`)
}

func TestShare_ServeHTTP_PUT_content_too_long(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := makeShareMux(t, store, mocks.NewTestCypher())
	ss := session(router, scache)

	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	longContent := strings.Repeat("x", 8001)

	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodPut,
		"/tools/share?content="+string(longContent)+"&ttl=900", nil)
	must.NoError(t, rerr)
	r.AddCookie(cookie)
	_ = r.ParseForm()

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `content is too long`)
}
