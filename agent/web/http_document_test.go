// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func newDocumentTestHarness(t *testing.T) (database.Store, *mux.Router, *http.Cookie) {
	t.Helper()

	store := mockdb.Open(t)
	handler := &documentHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("document.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}/{document}", ss).Methods(http.MethodGet, http.MethodDelete)

	cookie := scache.Create(mocks.Nobody, 1*time.Minute)
	return store, routes, cookie
}

func documentCount(t *testing.T, store database.Store) int {
	t.Helper()
	n, err := store.Documents().Count(t.Context())
	must.NoError(t, err)
	return n
}

func TestDocumentHTTP_Delete_HappyPath(t *testing.T) {
	t.Parallel()

	store, routes, cookie := newDocumentTestHarness(t)

	// seed a document for the user
	_, err := store.Documents().Create(t.Context(), &sitecontent.CreateDocumentRequest{
		User:   mocks.Nobody,
		Folder: "miscellaneous",
		Label:  "github",
	})
	must.NoError(t, err)
	must.One(t, documentCount(t, store))

	// DELETE the document; expect htmx redirect back to the folder page
	request := requestDelete(t, "/folders/miscellaneous/github", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/folders/miscellaneous", w.Header().Get("HX-Redirect"))
	must.Zero(t, documentCount(t, store))
}

func TestDocumentHTTP_Delete_NoSession(t *testing.T) {
	t.Parallel()

	_, routes, _ := newDocumentTestHarness(t)

	// no cookie -> forced to login
	request := requestDelete(t, "/folders/miscellaneous/github", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestDocumentHTTP_Delete_NotFound(t *testing.T) {
	t.Parallel()

	_, routes, cookie := newDocumentTestHarness(t)

	// no document with this label exists
	request := requestDelete(t, "/folders/miscellaneous/missing", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	// abort() translates to 500
	must.Eq(t, http.StatusInternalServerError, w.Code)
}
