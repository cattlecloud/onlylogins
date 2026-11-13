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
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

var env = runtime.Setup(&runtime.Config{
	Domain:      "test.local",
	Environment: "local",
})

func TestFolder_CreateDocument(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &folderHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("folder.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodPost)

	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	// POST a new document label into the user's default "miscellaneous" folder
	request := requestPost(t, "/folders/miscellaneous", cookie, url.Values{
		"label": {"github"},
	})
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	// on success, the handler emits an htmx redirect to the new document page
	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/folders/miscellaneous/github", w.Header().Get("HX-Redirect"))

	count, err := store.Documents().Count(t.Context())
	must.NoError(t, err)
	must.One(t, count)
}

func TestFolder_CreateDocument_BadLabel(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &folderHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("folder.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodPost)

	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	// a label with no letters or numbers fails Validate()
	request := requestPost(t, "/folders/miscellaneous", cookie, url.Values{
		"label": {"!!!"},
	})
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	// problem() renders the OOB error span at StatusOK
	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, body, "document must contain at least one letter or number")

	count, err := store.Documents().Count(t.Context())
	must.NoError(t, err)
	must.Zero(t, count)
}

func TestFolder_CreateDocument_DuplicateLabel(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &folderHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("folder.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodPost)

	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	// first create succeeds
	r1 := requestPost(t, "/folders/miscellaneous", cookie, url.Values{
		"label": {"github"},
	})
	w1 := httptest.NewRecorder()
	routes.ServeHTTP(w1, r1)
	must.Eq(t, http.StatusFound, w1.Code)
	must.Eq(t, "/folders/miscellaneous/github", w1.Header().Get("HX-Redirect"))

	// second create with the same label fails (duplicate UNIQUE constraint);
	// the handler routes db errors to abort(), which produces a 500 page
	r2 := requestPost(t, "/folders/miscellaneous", cookie, url.Values{
		"label": {"github"},
	})
	w2 := httptest.NewRecorder()
	routes.ServeHTTP(w2, r2)

	must.Eq(t, http.StatusInternalServerError, w2.Code)

	count, err := store.Documents().Count(t.Context())
	must.NoError(t, err)
	must.One(t, count)
}

func TestFolder_Get_Empty(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &folderHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("folder.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodGet, http.MethodDelete)

	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	request := requestGet(t, "/folders/miscellaneous", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `<h2>Your documents in miscellaneous</h2>`)
	must.StrContains(t, body, "No documents yet")
	must.StrContains(t, body, `class="empty"`)
	must.StrContains(t, body, `<span id="error"></span>`)
	must.StrNotContains(t, body, `class="documents-list"`)

	// the default "miscellaneous" folder is the well-known system folder; the
	// Danger Zone UI must be hidden on this page
	must.StrNotContains(t, body, "Danger Zone")
	must.StrNotContains(t, body, `class="danger-zone"`)
}

func TestFolder_Get_WithDocuments(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &folderHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("folder.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodGet, http.MethodDelete)

	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	ctx, cancel := scope.TTL(timeout)
	defer cancel()

	_, err := store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User:   mocks.Nobody,
		Folder: "miscellaneous",
		Label:  "github",
		Slug:   "github",
	})
	must.NoError(t, err)

	_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User:   mocks.Nobody,
		Folder: "miscellaneous",
		Label:  "gitlab",
		Slug:   "gitlab",
	})
	must.NoError(t, err)

	request := requestGet(t, "/folders/miscellaneous", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// the rendered list contains both labels as anchor text, in URLs that
	// match the redirect target emitted by the POST handler
	must.StrContains(t, body, `href="/folders/miscellaneous/github"`)
	must.StrContains(t, body, `href="/folders/miscellaneous/gitlab"`)
	must.StrContains(t, body, `class="documents-list"`)

	// the empty-state copy is absent when there are documents
	must.StrContains(t, body, `<h2>Your documents in miscellaneous</h2>`)
	must.StrNotContains(t, body, "No documents yet")
	must.StrNotContains(t, body, `class="empty"`)

	// "miscellaneous" is the well-known system folder: Danger Zone stays hidden
	must.StrNotContains(t, body, "Danger Zone")
	must.StrNotContains(t, body, `class="danger-zone"`)
}

// TestFolder_Get_DangerZone_ForUserCreated confirms the Danger Zone is
// rendered for a user-created folder (anything other than the system
// "miscellaneous" folder).
func TestFolder_Get_DangerZone_ForUserCreated(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &folderHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("folder.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodGet, http.MethodDelete)

	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	// create a user-owned folder (anything other than "miscellaneous")
	ctx, cancel := scope.TTL(timeout)
	defer cancel()
	must.NoError(t, store.Folders().Create(ctx, &sitecontent.CreateFolderRequest{
		User:  mocks.Nobody,
		Label: "work",
		Slug:  "work",
		Icon:  sitecontent.Icon(0),
	}))

	request := requestGet(t, "/folders/work", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// user-created folders ARE deletable, so the Danger Zone is present
	must.StrContains(t, body, "Danger Zone")
	must.StrContains(t, body, `class="danger-zone"`)
	must.StrContains(t, body, `class="danger-zone-delete-btn"`)
	must.StrContains(t, body, "Delete folder")
	must.StrContains(t, body, `hx-delete="/folders/work"`)
	must.StrContains(t, body, `hx-confirm="Delete this folder work and ALL of its documents and secrets?"`)
}

func TestFolderHTTP_Delete_HappyPath(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &folderHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("folder.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodGet, http.MethodDelete)

	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	ctx, cancel := scope.TTL(timeout)
	defer cancel()

	// create a user-owned folder (anything other than the system folder)
	must.NoError(t, store.Folders().Create(ctx, &sitecontent.CreateFolderRequest{
		User:  mocks.Nobody,
		Label: "work",
		Slug:  "work",
		Icon:  sitecontent.Icon(0),
	}))

	// DELETE the folder; expect htmx redirect back to "/"
	request := requestDelete(t, "/folders/work", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/", w.Header().Get("HX-Redirect"))

	// folder count back to just the system default
	count, err := store.Folders().Count(ctx)
	must.NoError(t, err)
	must.Eq(t, 1, count)
}

func TestFolderHTTP_Delete_NoSession(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &folderHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("folder.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodGet, http.MethodDelete)

	// no cookie -> forced to login
	request := requestDelete(t, "/folders/work", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestFolderHTTP_Delete_ProtectedFolder(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &folderHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("folder.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodGet, http.MethodDelete)

	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	// attempting to delete the system "miscellaneous" folder triggers `abort`
	// (the SQL `protected = 0` filter refuses to delete protected rows)
	request := requestDelete(t, "/folders/miscellaneous", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusInternalServerError, w.Code)

	// folder is still intact
	ctx, cancel := scope.TTL(timeout)
	defer cancel()
	count, err := store.Folders().Count(ctx)
	must.NoError(t, err)
	must.Eq(t, 1, count)
}

func TestFolderHTTP_Delete_NotFound(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &folderHTTP{
		log:         ulog.New("test"),
		store:       store,
		environment: env,
		html:        ui.Load("folder.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodGet, http.MethodDelete)

	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	// no such folder exists
	request := requestDelete(t, "/folders/no-such-folder", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	// abort() -> 500
	must.Eq(t, http.StatusInternalServerError, w.Code)
}
