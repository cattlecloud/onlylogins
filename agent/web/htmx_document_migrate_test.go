// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func newMigrateDocumentRouter(t *testing.T, store database.Store) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle((&migrateDocumentHTMX{}).Path(), &migrateDocumentHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	return router, scache
}

func TestMigrateDocumentHTMX_HappyPath(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestDocument(t, store, "github")
	createTestFolder(t, store, "work", "work")

	router, scache := newMigrateDocumentRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/document/migrate/miscellaneous/github", cookie, url.Values{
		"target": {"work"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/folders/work/github", w.Header().Get("HX-Redirect"))

	// the document now lives under the target folder
	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	doc, err := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User: mocks.Nobody, Folder: "work", Document: "github",
	})
	must.NoError(t, err)
	must.Eq(t, "github", doc.Label)
}

func TestMigrateDocumentHTMX_NoSession(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newMigrateDocumentRouter(t, store)
	ss := session(router, scache)

	r := requestPost(t, "/internal/document/migrate/miscellaneous/github", nil, url.Values{
		"target": {"work"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestMigrateDocumentHTMX_BadTarget(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestDocument(t, store, "github")

	router, scache := newMigrateDocumentRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/document/migrate/miscellaneous/github", cookie, url.Values{
		"target": {""},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, body, "target folder must be set")

	// the document is untouched
	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	_, err := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
	})
	must.NoError(t, err)
}

func TestMigrateDocumentHTMX_SameFolder(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestDocument(t, store, "github")

	router, scache := newMigrateDocumentRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/document/migrate/miscellaneous/github", cookie, url.Values{
		"target": {"miscellaneous"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), "document is already in the target folder")
}

func TestMigrateDocumentHTMX_DuplicateName(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestDocument(t, store, "github")
	createTestFolder(t, store, "work", "work")

	// a second document with the same label already exists in the target
	ctx, cancel := scope.TTL(testwait)
	_, err := store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User:   mocks.Nobody,
		Folder: "work",
		Label:  "github",
		Slug:   "github",
	})
	cancel()
	must.NoError(t, err)

	router, scache := newMigrateDocumentRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/document/migrate/miscellaneous/github", cookie, url.Values{
		"target": {"work"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), "a document with that name already exists")
}

func TestMigrateDocumentHTMX_NotFound(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestFolder(t, store, "work", "work")

	router, scache := newMigrateDocumentRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/document/migrate/miscellaneous/no-such-document", cookie, url.Values{
		"target": {"work"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}
