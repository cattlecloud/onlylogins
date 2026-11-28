// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/ui"
	"github.com/BurntSushi/toml"
	"github.com/gorilla/mux"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func TestExport_ServeHTTP_LoggedOut(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/tools/export", &exportHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		html:        ui.Load("export.html"),
	})

	request := requestGet(t, "/tools/export", nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<div class="middle export">`)
	must.StrContains(t, body, "You must be logged in to export your data.")
	must.StrContains(t, body, `<form method="get" action="/login">`)
	must.StrContains(t, body, `Log in to export your data`)
	must.StrNotContains(t, body, `<form method="post" action="/tools/export">`)
}

func TestExport_ServeHTTP_LoggedIn(t *testing.T) {
	t.Parallel()

	_, routes, scache, _ := newExportHarness(t)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	request := requestGet(t, "/tools/export", cookie)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<div class="middle export">`)
	must.StrContains(t, body, `<form method="post" action="/tools/export">`)
	must.StrContains(t, body, `Get backup file`)
	must.StrNotContains(t, body, `hx-post="/tools/export"`)
	must.StrNotContains(t, body, "You must be logged in")
}

func newExportHarness(t *testing.T) (database.Store, *mux.Router, *mocks.Sessions, mocks.TestCypher) {
	t.Helper()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()
	scache := mocks.NewSessions()

	handler := &exportHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		html:        ui.Load("export.html"),
	}

	routes := mux.NewRouter()
	routes.Handle("/tools/export", session(handler, scache))

	return store, routes, scache, cypher
}

func TestExport_Post(t *testing.T) {
	t.Parallel()

	store, routes, scache, cypher := newExportHarness(t)

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	// seed a document and one of each kind of encrypted value
	_, err := store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User:   mocks.Nobody,
		Folder: "miscellaneous",
		Label:  "github",
		Slug:   "github",
	})
	must.NoError(t, err)

	encrypted := func(value string) *conceal.Bytes {
		ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New(value))
		must.NoError(t, cerr)
		return ciphertext
	}

	_, err = store.Documents().CreateWeblink(ctx, &sitecontent.CreateWeblinkRequest{
		User:      mocks.Nobody,
		Folder:    "miscellaneous",
		Document:  "github",
		URL:       "https://github.com",
		Encrypted: encrypted("https://github.com"),
	})
	must.NoError(t, err)

	_, err = store.Documents().CreateUsername(ctx, &sitecontent.CreateUsernameRequest{
		User:      mocks.Nobody,
		Folder:    "miscellaneous",
		Document:  "github",
		Username:  "shoenig",
		Encrypted: encrypted("shoenig"),
	})
	must.NoError(t, err)

	_, err = store.Documents().CreatePassword(ctx, &sitecontent.CreatePasswordRequest{
		User:      mocks.Nobody,
		Folder:    "miscellaneous",
		Document:  "github",
		Encrypted: encrypted("s3cret!"),
	})
	must.NoError(t, err)

	_, err = store.Documents().CreateKeyval(ctx, &sitecontent.CreateKeyvalRequest{
		User:      mocks.Nobody,
		Folder:    "miscellaneous",
		Document:  "github",
		Key:       "recovery",
		Encrypted: encrypted("backup-code"),
	})
	must.NoError(t, err)

	_, err = store.Documents().CreateNote(ctx, &sitecontent.CreateNoteRequest{
		User:      mocks.Nobody,
		Folder:    "miscellaneous",
		Document:  "github",
		Encrypted: encrypted("remember the milk"),
	})
	must.NoError(t, err)

	cookie := scache.Create(mocks.Nobody, 1*time.Minute)
	request := requestPost(t, "/tools/export", cookie, nil)
	request.AddCookie(cypher.Cookie())
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	must.Eq(t, "application/toml; charset=utf8", w.Header().Get("Content-Type"))
	must.StrContains(t, w.Header().Get("Content-Disposition"), "attachment")

	var ee sitecontent.ExportEverything
	must.NoError(t, toml.Unmarshal(w.Body.Bytes(), &ee))

	must.Eq(t, "_nobody", ee.Username)
	must.SliceLen(t, 1, ee.Folders)

	folder := ee.Folders[0]
	must.Eq(t, "miscellaneous", folder.Name)
	must.SliceLen(t, 1, folder.Documents)

	document := folder.Documents[0]
	must.Eq(t, "github", document.Name)
	must.SliceEqOp(t, []string{"https://github.com"}, document.Weblinks)
	must.SliceEqOp(t, []string{"shoenig"}, document.Usernames)
	must.SliceEqOp(t, []string{"s3cret!"}, document.Passwords)
	must.SliceEqOp(t, [][2]string{{"recovery", "backup-code"}}, document.Keyvals)
	must.Eq(t, "remember the milk", document.Note)
}

func TestExport_Post_Empty(t *testing.T) {
	t.Parallel()

	_, routes, scache, cypher := newExportHarness(t)

	cookie := scache.Create(mocks.Nobody, 1*time.Minute)
	request := requestPost(t, "/tools/export", cookie, nil)
	request.AddCookie(cypher.Cookie())
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)

	var ee sitecontent.ExportEverything
	must.NoError(t, toml.Unmarshal(w.Body.Bytes(), &ee))
	must.SliceLen(t, 1, ee.Folders)

	// the default folder exists but has no documents; it must serialize as []
	must.Eq(t, "miscellaneous", ee.Folders[0].Name)
	must.SliceEmpty(t, ee.Folders[0].Documents)
	must.StrContains(t, w.Body.String(), `documents = []`)
}

func TestExport_Post_NoSession(t *testing.T) {
	t.Parallel()

	_, routes, _, _ := newExportHarness(t) //nolint:dogsled

	request := requestPost(t, "/tools/export", nil, nil)
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	must.Eq(t, http.StatusTemporaryRedirect, w.Code)
	must.Eq(t, "/login", w.Header().Get("Location"))
}
