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

func newRenameDocumentRouter(t *testing.T, store database.Store) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle((&renameDocumentHTMX{}).Path(), &renameDocumentHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	return router, scache
}

func createTestDocument(t *testing.T, store database.Store, label string) {
	t.Helper()
	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	_, err := store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User:   mocks.Nobody,
		Folder: "miscellaneous",
		Label:  label,
		Slug:   sitecontent.Slugify(label),
	})
	must.NoError(t, err)
}

func TestRenameDocumentHTMX_HappyPath(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestDocument(t, store, "github")

	router, scache := newRenameDocumentRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/document/rename/miscellaneous/github", cookie, url.Values{
		"label": {"gitlab"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/folders/miscellaneous/gitlab", w.Header().Get("HX-Redirect"))

	// the document now lives under its new slug
	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	doc, err := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User: mocks.Nobody, Folder: "miscellaneous", Document: "gitlab",
	})
	must.NoError(t, err)
	must.Eq(t, "gitlab", doc.Label)
}

func TestRenameDocumentHTMX_NoSession(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newRenameDocumentRouter(t, store)
	ss := session(router, scache)

	r := requestPost(t, "/internal/document/rename/miscellaneous/github", nil, url.Values{
		"label": {"gitlab"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestRenameDocumentHTMX_BadLabel(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestDocument(t, store, "github")

	router, scache := newRenameDocumentRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/document/rename/miscellaneous/github", cookie, url.Values{
		"label": {"!!!"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, body, "document must contain at least one letter or number")

	// the document is untouched
	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	_, err := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
	})
	must.NoError(t, err)
}

func TestRenameDocumentHTMX_DuplicateName(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestDocument(t, store, "github")
	createTestDocument(t, store, "gitlab")

	router, scache := newRenameDocumentRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/document/rename/miscellaneous/github", cookie, url.Values{
		"label": {"gitlab"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), "a document with that name already exists")

	// both documents remain intact
	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	list, err := store.Documents().List(ctx, &sitecontent.ListDocumentsRequest{
		User: mocks.Nobody, Folder: "miscellaneous",
	})
	must.NoError(t, err)
	must.SliceLen(t, 2, list)
}

func TestRenameDocumentHTMX_NotFound(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newRenameDocumentRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/document/rename/miscellaneous/no-such-document", cookie, url.Values{
		"label": {"gitlab"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}
