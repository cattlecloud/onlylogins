// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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
	ss := sessions(handler, scache)

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
	ss := sessions(handler, scache)

	routes := mux.NewRouter()
	routes.Handle("/folders/{folder}", ss).Methods(http.MethodPost)

	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	// a label containing non-alphanumeric characters fails Validate()
	request := requestPost(t, "/folders/miscellaneous", cookie, url.Values{
		"label": {"bad-name!"},
	})
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	// problem() renders the OOB error span at StatusOK
	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, body, "document label must contain alphanumeric characters")

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
	ss := sessions(handler, scache)

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

	// second create with the same label fails with ErrDocumentLabelTaken
	r2 := requestPost(t, "/folders/miscellaneous", cookie, url.Values{
		"label": {"github"},
	})
	w2 := httptest.NewRecorder()
	routes.ServeHTTP(w2, r2)

	must.Eq(t, http.StatusOK, w2.Code)
	body := w2.Body.String()
	must.StrContains(t, body, `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, body, database.ErrDocumentLabelTaken.Error())

	count, err := store.Documents().Count(t.Context())
	must.NoError(t, err)
	must.One(t, count)
}
