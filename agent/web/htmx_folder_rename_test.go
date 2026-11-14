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

func newRenameFolderRouter(t *testing.T, store database.Store) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle((&renameFolderHTMX{}).Path(), &renameFolderHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	return router, scache
}

func createTestFolder(t *testing.T, store database.Store, label, slug string) {
	t.Helper()
	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	must.NoError(t, store.Folders().Create(ctx, &sitecontent.CreateFolderRequest{
		User:  mocks.Nobody,
		Label: label,
		Slug:  slug,
		Icon:  sitecontent.Icon(0),
	}))
}

func TestRenameFolderHTMX_HappyPath(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestFolder(t, store, "work", "work")

	router, scache := newRenameFolderRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/folder/rename/work", cookie, url.Values{
		"label": {"home"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/folders/home", w.Header().Get("HX-Redirect"))

	// the folder now lives under its new slug
	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	folder, err := store.Folders().Get(ctx, mocks.Nobody, "home")
	must.NoError(t, err)
	must.Eq(t, "home", folder.Label)
}

func TestRenameFolderHTMX_NoSession(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newRenameFolderRouter(t, store)
	ss := session(router, scache)

	r := requestPost(t, "/internal/folder/rename/work", nil, url.Values{
		"label": {"home"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestRenameFolderHTMX_BadLabel(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestFolder(t, store, "work", "work")

	router, scache := newRenameFolderRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/folder/rename/work", cookie, url.Values{
		"label": {"!!!"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, body, "label must contain only letters, numbers, and dashes")

	// the folder is untouched
	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	_, err := store.Folders().Get(ctx, mocks.Nobody, "work")
	must.NoError(t, err)
}

func TestRenameFolderHTMX_DuplicateName(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	createTestFolder(t, store, "home", "home")
	createTestFolder(t, store, "work", "work")

	router, scache := newRenameFolderRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/folder/rename/work", cookie, url.Values{
		"label": {"home"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), "a folder with that name already exists")

	// both folders remain intact
	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	list, err := store.Folders().List(ctx, mocks.Nobody)
	must.NoError(t, err)
	must.SliceLen(t, 3, list) // miscellaneous + home + work
}

func TestRenameFolderHTMX_ProtectedFolder(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newRenameFolderRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/folder/rename/miscellaneous", cookie, url.Values{
		"label": {"other"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestRenameFolderHTMX_NotFound(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newRenameFolderRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestPost(t, "/internal/folder/rename/no-such-folder", cookie, url.Values{
		"label": {"home"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}
