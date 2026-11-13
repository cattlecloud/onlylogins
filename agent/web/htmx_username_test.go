// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/gorilla/mux"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func newUsernameRouter(t *testing.T, store database.Store) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle((&usernameHTMX{}).Path(), &usernameHTMX{
		log:   ulog.New("test"),
		store: store,
	})
	router.Handle((&usernameEntryHTMX{}).Path(), &usernameEntryHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	return router, scache
}

func insertUsernameHandlerHelper(t *testing.T, store database.Store, user users.Identity,
	doc, username string, crypt mocks.TestCrypt) records.UsernamesAuto {
	t.Helper()

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	list, err := store.Folders().List(ctx, user)
	must.NoError(t, err)
	must.SliceLen(t, 1, list)
	folder := list[0].Label

	_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User: user, Folder: folder, Label: doc, Slug: sitecontent.Slugify(doc),
	})
	must.NoError(t, err)

	ciphertext, cerr := crypt.Vault().Encrypt(conceal.New(username))
	must.NoError(t, cerr)

	id, err := store.Documents().CreateUsername(ctx, &sitecontent.CreateUsernameRequest{
		User:      user,
		Folder:    folder,
		Document:  doc,
		Username:  username,
		Encrypted: ciphertext,
	})
	must.NoError(t, err)
	return id
}

func ensureBobUsername(t *testing.T, store database.Store) users.Identity {
	t.Helper()
	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	salt := encrypt.Salt()
	passphrase := conceal.New("pass+bob long enough")
	key := encrypt.DeriveKey(passphrase, salt)
	verifier := encrypt.MakeVerifier(key)

	id, err := store.Users().Create(ctx, &users.CreateUserRequest{
		Email:      "bob@example.com",
		Username:   "bob",
		Passphrase: passphrase,
		Salt:       salt,
	}, verifier)
	must.NoError(t, err)
	must.Positive(t, id)
	return id
}

func usernameURL(action, folder, document string, usernameID records.UsernamesAuto) string {
	return path.Join("/internal/username", action, folder, document, strconv.Itoa(int(usernameID)))
}

func postUsernameForm(t *testing.T, method, urlPath string, values url.Values) *http.Request {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), method, urlPath, strings.NewReader(values.Encode()))
	must.NoError(t, err)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func fetchDocUsername(t *testing.T, store database.Store, user users.Identity,
	folder, document string) *sitecontent.Document {
	t.Helper()
	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	doc, err := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User: user, Folder: folder, Document: document,
	})
	must.NoError(t, err)
	return doc
}

func TestUsernameHTMX_Get_RendersForm(t *testing.T) {
	t.Parallel()

	router, scache := newUsernameRouter(t, mockdb.Open(t))
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/username/add/miscellaneous/github", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `hx-post="/internal/username/create/`)
	must.StrContains(t, body, `name="username"`)
	must.StrContains(t, body, `placeholder="username"`)
}

func TestUsernameHTMX_Get_NoSession_Redirects(t *testing.T) {
	t.Parallel()

	router, scache := newUsernameRouter(t, mockdb.Open(t))
	ss := session(router, scache)

	r := requestGet(t, "/internal/username/add/miscellaneous/github", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestUsernameHTMX_Post_HappyPath_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	// create a document under the default folder
	{
		ctx, cancel := scope.TTL(testwait)
		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github", Slug: "github",
		})
		must.NoError(t, err)
		cancel()
	}

	r := postUsernameForm(t, http.MethodPost,
		"/internal/username/create/miscellaneous/github",
		url.Values{"username": {"alice"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/folders/miscellaneous/github", w.Header().Get("HX-Redirect"))

	doc := fetchDocUsername(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Usernames)
	must.Eq(t, "alice", doc.Usernames[0].Unlock(crypt.Vault()))
}

func TestUsernameHTMX_Post_EmptyUsername_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	// create document
	{
		ctx, cancel := scope.TTL(testwait)
		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github", Slug: "github",
		})
		must.NoError(t, err)
		cancel()
	}

	r := postUsernameForm(t, http.MethodPost,
		"/internal/username/create/miscellaneous/github",
		url.Values{"username": {""}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocUsername(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 0, doc.Usernames)
}

func TestUsernameHTMX_Post_WhitespaceOnlyUsername_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	// create document
	{
		ctx, cancel := scope.TTL(testwait)
		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github", Slug: "github",
		})
		must.NoError(t, err)
		cancel()
	}

	r := postUsernameForm(t, http.MethodPost,
		"/internal/username/create/miscellaneous/github",
		url.Values{"username": {"   "}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocUsername(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 0, doc.Usernames)
}

func TestUsernameHTMX_Post_TooLongUsername_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	// create document
	{
		ctx, cancel := scope.TTL(testwait)
		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github", Slug: "github",
		})
		must.NoError(t, err)
		cancel()
	}

	longName := strings.Repeat("a", 257)
	r := postUsernameForm(t, http.MethodPost,
		"/internal/username/create/miscellaneous/github",
		url.Values{"username": {longName}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocUsername(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 0, doc.Usernames)
}

func TestUsernameHTMX_Post_MissingVault_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := postUsernameForm(t, http.MethodPost,
		"/internal/username/create/miscellaneous/github",
		url.Values{"username": {"alice"}},
	)
	r.AddCookie(cookie)
	// intentionally no vault cookie

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestUsernameHTMX_Post_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobUsername(t, store)

	// nobody owns the document
	{
		ctx, cancel := scope.TTL(testwait)
		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github", Slug: "github",
		})
		must.NoError(t, err)
		cancel()
	}

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	r := postUsernameForm(t, http.MethodPost,
		"/internal/username/create/miscellaneous/github",
		url.Values{"username": {"mallory"}},
	)
	r.AddCookie(bobCookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestUsernameEntryHTMX_Get_RendersEditForm(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertUsernameHandlerHelper(t, store, mocks.Nobody, "github", "alice", crypt)

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := usernameURL("edit", "miscellaneous", "github", id)
	r := requestGet(t, u, cookie)
	r.AddCookie(crypt.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `hx-put="/internal/username/update/`)
	must.StrContains(t, body, `value="alice"`)
	must.StrContains(t, body, `<span id="error">`)
}

func TestUsernameEntryHTMX_Put_HappyPath_ReturnsNewRow(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertUsernameHandlerHelper(t, store, mocks.Nobody, "github", "alice", crypt)

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := usernameURL("update", "miscellaneous", "github", id)
	r := postUsernameForm(t, http.MethodPut, u, url.Values{"username": {"alice2"}})
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `alice2`)
	must.StrContains(t, body, `class="username-row"`)

	doc := fetchDocUsername(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Usernames)
	must.Eq(t, "alice2", doc.Usernames[0].Unlock(crypt.Vault()))
}

func TestUsernameEntryHTMX_Put_EmptyUsername_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertUsernameHandlerHelper(t, store, mocks.Nobody, "github", "alice", crypt)

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := usernameURL("update", "miscellaneous", "github", id)
	r := postUsernameForm(t, http.MethodPut, u, url.Values{"username": {""}})
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	// original row remains untouched
	doc := fetchDocUsername(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Usernames)
	must.Eq(t, "alice", doc.Usernames[0].Unlock(crypt.Vault()))
}

func TestUsernameEntryHTMX_Put_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobUsername(t, store)
	id := insertUsernameHandlerHelper(t, store, mocks.Nobody, "github", "alice", crypt)

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	u := usernameURL("update", "miscellaneous", "github", id)
	r := postUsernameForm(t, http.MethodPut, u, url.Values{"username": {"mallory"}})
	r.AddCookie(bobCookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestUsernameEntryHTMX_Delete_HappyPath_EmptyBody(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertUsernameHandlerHelper(t, store, mocks.Nobody, "github", "alice", crypt)

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := usernameURL("delete", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(cookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.Eq(t, "", w.Body.String())

	doc := fetchDocUsername(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 0, doc.Usernames)
}

func TestUsernameEntryHTMX_Delete_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobUsername(t, store)
	id := insertUsernameHandlerHelper(t, store, mocks.Nobody, "github", "alice", crypt)

	router, scache := newUsernameRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	u := usernameURL("delete", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(bobCookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}
