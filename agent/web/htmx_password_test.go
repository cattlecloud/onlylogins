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

func newPasswordRouter(t *testing.T, store database.Store) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()

	router.Handle((&passwordHTMX{}).Path(), &passwordHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	router.Handle((&passwordEntryHTMX{}).Path(), &passwordEntryHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	return router, scache
}

func insertPasswordHandlerHelper(t *testing.T, store database.Store, user users.Identity,
	doc, password string, crypt mocks.TestCrypt) records.PasswordsAuto {
	t.Helper()

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	list, err := store.Folders().List(ctx, user)
	must.NoError(t, err)
	must.SliceLen(t, 1, list)
	folder := list[0].Label

	_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User: user, Folder: folder, Label: doc,
	})
	must.NoError(t, err)

	ciphertext, cerr := crypt.Vault().Encrypt(conceal.New(password))
	must.NoError(t, cerr)

	id, err := store.Documents().CreatePassword(ctx, &sitecontent.CreatePasswordRequest{
		User:      user,
		Folder:    folder,
		Document:  doc,
		Encrypted: ciphertext,
	})
	must.NoError(t, err)
	return id
}

func ensureBobPassword(t *testing.T, store database.Store) users.Identity {
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

func passwordURL(action, folder, document string, passwordID records.PasswordsAuto) string {
	return path.Join("/internal/password", action, folder, document, strconv.Itoa(int(passwordID)))
}

func postPasswordForm(t *testing.T, method, urlPath string, values url.Values) *http.Request {
	t.Helper()

	r, err := http.NewRequestWithContext(t.Context(), method, urlPath, strings.NewReader(values.Encode()))
	must.NoError(t, err)

	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func fetchDocPassword(t *testing.T, store database.Store, user users.Identity, folder, document string) *sitecontent.Document {
	t.Helper()

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	doc, err := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User: user, Folder: folder, Document: document,
	})
	must.NoError(t, err)

	return doc
}

func TestPasswordHTMX_Get_RendersForm(t *testing.T) {
	t.Parallel()

	router, scache := newPasswordRouter(t, mockdb.Open(t))
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/password/add/miscellaneous/github", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)

	body := w.Body.String()
	must.StrContains(t, body, `hx-post="/internal/password/create/`)
	must.StrContains(t, body, `name="password"`)
	must.StrContains(t, body, `type="password"`)
}

func TestPasswordHTMX_Get_NoSession_Redirects(t *testing.T) {
	t.Parallel()

	router, scache := newPasswordRouter(t, mockdb.Open(t))
	ss := session(router, scache)

	r := requestGet(t, "/internal/password/add/miscellaneous/github", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestPasswordHTMX_Post_HappyPath_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	// create a document under the default folder
	{
		ctx, cancel := scope.TTL(testwait)
		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github",
		})
		must.NoError(t, err)
		cancel()
	}

	r := postPasswordForm(t, http.MethodPost,
		"/internal/password/create/miscellaneous/github",
		url.Values{"password": {"s3cret!"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/folders/miscellaneous/github", w.Header().Get("HX-Redirect"))

	doc := fetchDocPassword(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Passwords)
	must.Eq(t, "s3cret!", doc.Passwords[0].Unlock(crypt.Vault()))
}

func TestPasswordHTMX_Post_EmptyPassword_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	// create document
	{
		ctx, cancel := scope.TTL(testwait)
		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github",
		})
		must.NoError(t, err)
		cancel()
	}

	r := postPasswordForm(t, http.MethodPost,
		"/internal/password/create/miscellaneous/github",
		url.Values{"password": {""}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocPassword(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceEmpty(t, doc.Passwords)
}

func TestPasswordHTMX_Post_TooLongPassword_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	// create document
	{
		ctx, cancel := scope.TTL(testwait)
		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github",
		})
		must.NoError(t, err)
		cancel()
	}

	longPass := strings.Repeat("a", 257)
	r := postPasswordForm(t, http.MethodPost,
		"/internal/password/create/miscellaneous/github",
		url.Values{"password": {longPass}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocPassword(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceEmpty(t, doc.Passwords)
}

func TestPasswordHTMX_Post_PreservesWhitespace(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	// create document
	{
		ctx, cancel := scope.TTL(testwait)
		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github",
		})
		must.NoError(t, err)
		cancel()
	}

	r := postPasswordForm(t, http.MethodPost,
		"/internal/password/create/miscellaneous/github",
		url.Values{"password": {"  spaces  "}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)

	doc := fetchDocPassword(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Passwords)
	must.Eq(t, "  spaces  ", doc.Passwords[0].Unlock(crypt.Vault()))
}

func TestPasswordHTMX_Post_MissingVault_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := postPasswordForm(t, http.MethodPost,
		"/internal/password/create/miscellaneous/github",
		url.Values{"password": {"hunter2"}},
	)
	r.AddCookie(cookie)
	// intentionally no vault cookie

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestPasswordHTMX_Post_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobPassword(t, store)

	// nobody owns the document
	{
		ctx, cancel := scope.TTL(testwait)
		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github",
		})
		must.NoError(t, err)
		cancel()
	}

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	r := postPasswordForm(t, http.MethodPost,
		"/internal/password/create/miscellaneous/github",
		url.Values{"password": {"hacked!"}},
	)
	r.AddCookie(bobCookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestPasswordEntryHTMX_Get_RendersEditForm(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertPasswordHandlerHelper(t, store, mocks.Nobody, "github", "s3cret!", crypt)

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := passwordURL("edit", "miscellaneous", "github", id)
	r := requestGet(t, u, cookie)
	r.AddCookie(crypt.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)

	body := w.Body.String()
	must.StrContains(t, body, `hx-put="/internal/password/update/`)
	must.StrContains(t, body, `value="s3cret!"`)
	must.StrContains(t, body, `<span id="error">`)
}

func TestPasswordEntryHTMX_Put_HappyPath_ReturnsNewRow(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertPasswordHandlerHelper(t, store, mocks.Nobody, "github", "oldpass", crypt)

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := passwordURL("update", "miscellaneous", "github", id)
	r := postPasswordForm(t, http.MethodPut, u, url.Values{"password": {"newpass1"}})
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()

	must.StrContains(t, body, "newpass1")
	must.StrContains(t, body, `class="password-row"`)

	doc := fetchDocPassword(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Passwords)
	must.Eq(t, "newpass1", doc.Passwords[0].Unlock(crypt.Vault()))
}

func TestPasswordEntryHTMX_Put_EmptyPassword_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertPasswordHandlerHelper(t, store, mocks.Nobody, "github", "realpass", crypt)

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := passwordURL("update", "miscellaneous", "github", id)
	r := postPasswordForm(t, http.MethodPut, u, url.Values{"password": {""}})
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	// original row remains untouched
	doc := fetchDocPassword(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Passwords)
	must.Eq(t, "realpass", doc.Passwords[0].Unlock(crypt.Vault()))
}

func TestPasswordEntryHTMX_Put_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobPassword(t, store)
	id := insertPasswordHandlerHelper(t, store, mocks.Nobody, "github", "realpass", crypt)

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	u := passwordURL("update", "miscellaneous", "github", id)
	r := postPasswordForm(t, http.MethodPut, u, url.Values{"password": {"hacked!"}})
	r.AddCookie(bobCookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestPasswordEntryHTMX_Delete_HappyPath_EmptyBody(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertPasswordHandlerHelper(t, store, mocks.Nobody, "github", "secret", crypt)

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := passwordURL("delete", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(cookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.Eq(t, "", w.Body.String())

	doc := fetchDocPassword(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceEmpty(t, doc.Passwords)
}

func TestPasswordEntryHTMX_Delete_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobPassword(t, store)
	id := insertPasswordHandlerHelper(t, store, mocks.Nobody, "github", "realpass", crypt)

	router, scache := newPasswordRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	u := passwordURL("delete", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(bobCookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}
