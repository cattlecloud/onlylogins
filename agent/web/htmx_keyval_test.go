// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

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

func makeKeyvalMux(t *testing.T, store database.Store) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()

	router.Handle((&keyvalHTMX{}).Path(), &keyvalHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	router.Handle((&keyvalEntryHTMX{}).Path(), &keyvalEntryHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	return router, scache
}

// setupGithubDocument creates the "miscellaneous/github" document
func setupGithubDocument(t *testing.T, store database.Store, user users.Identity) {
	t.Helper()

	ctx, cancel := scope.TTL(5 * time.Second)
	defer cancel()

	list, err := store.Folders().List(ctx, mocks.Nobody)
	must.NoError(t, err)
	must.SliceLen(t, 1, list)

	folder := list[0].Label
	_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User: user, Folder: folder, Label: "github",
	})
	must.NoError(t, err)
}

func insertKeyval(t *testing.T, store database.Store, user users.Identity, key, value string, crypt mocks.TestCrypt) records.KeyvalsAuto {
	t.Helper()

	ctx, cancel := scope.TTL(5 * time.Second)
	defer cancel()

	list, err := store.Folders().List(ctx, user)
	must.NoError(t, err)
	must.SliceLen(t, 1, list)

	folder := list[0].Label

	must.NoError(t, sitecontent.ValidateKeyvalValue(value))

	ciphertext, cerr := crypt.Vault().Encrypt(conceal.New(value))
	must.NoError(t, cerr)

	request := &sitecontent.CreateKeyvalRequest{
		User:      user,
		Folder:    folder,
		Document:  "github",
		Key:       key,
		Encrypted: ciphertext,
	}
	must.NoError(t, request.Validate())

	id, err := store.Documents().CreateKeyval(ctx, request)
	must.NoError(t, err)

	return id
}

func ensureBobKeyval(t *testing.T, store database.Store) users.Identity {
	t.Helper()

	ctx, cancel := scope.TTL(5 * time.Second)
	defer cancel()

	salt := encrypt.Salt()
	passphrase := conceal.New("pass+bob long enough")
	key := encrypt.DeriveKey(passphrase, salt)
	verifier := encrypt.MakeVerifier(key)

	id, err := store.Users().Create(ctx, &users.CreateUserRequest{
		Email:      "bob-keyval@example.com",
		Username:   "bob-keyval",
		Passphrase: passphrase,
		Salt:       salt,
	}, verifier)
	must.NoError(t, err)
	must.Positive(t, id)

	return id
}

func keyvalURL(action, folder, document string, keyvalID records.KeyvalsAuto) string {
	return path.Join("/internal/keyval", action, folder, document, strconv.Itoa(int(keyvalID)))
}

func postKeyvalForm(t *testing.T, method, urlPath string, values url.Values) *http.Request {
	t.Helper()

	r, err := http.NewRequestWithContext(t.Context(), method, urlPath, strings.NewReader(values.Encode()))
	must.NoError(t, err)

	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func fetchDocKeyvals(t *testing.T, store database.Store, user users.Identity, folder, document string) *sitecontent.Document {
	t.Helper()

	ctx, cancel := scope.TTL(5 * time.Second)
	defer cancel()

	doc, err := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User: user, Folder: folder, Document: document,
	})
	must.NoError(t, err)

	return doc
}

func TestKeyvalHTMX_Get_RendersForm(t *testing.T) {
	t.Parallel()

	router, scache := makeKeyvalMux(t, mockdb.Open(t))
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/keyval/add/miscellaneous/github", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)

	body := w.Body.String()
	must.StrContains(t, body, `hx-post="/internal/keyval/create/`)
	must.StrContains(t, body, `name="key"`)
	must.StrContains(t, body, `name="value"`)
}

func TestKeyvalHTMX_Get_NoSession_Aborts(t *testing.T) {
	t.Parallel()

	router, scache := makeKeyvalMux(t, mockdb.Open(t))
	ss := sessions(router, scache)

	r := requestGet(t, "/internal/keyval/add/miscellaneous/github", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestKeyvalHTMX_Post_HappyPath_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	setupGithubDocument(t, store, mocks.Nobody)

	r := postKeyvalForm(t, http.MethodPost,
		"/internal/keyval/create/miscellaneous/github",
		url.Values{"key": {"api_endpoint"}, "value": {"https://api.example.com"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/folders/miscellaneous/github", w.Header().Get("HX-Redirect"))

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Keyvals)
	must.Eq(t, "api_endpoint", doc.Keyvals[0].K.Unveil())
	must.Eq(t, "https://api.example.com", doc.Keyvals[0].Unlock(crypt.Vault()))
}

func TestKeyvalHTMX_Post_EmptyKey_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	setupGithubDocument(t, store, mocks.Nobody)

	r := postKeyvalForm(t, http.MethodPost,
		"/internal/keyval/create/miscellaneous/github",
		url.Values{"key": {"   "}, "value": {"v"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceEmpty(t, doc.Keyvals)
}

func TestKeyvalHTMX_Post_EmptyValue_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	setupGithubDocument(t, store, mocks.Nobody)

	r := postKeyvalForm(t, http.MethodPost,
		"/internal/keyval/create/miscellaneous/github",
		url.Values{"key": {"k"}, "value": {""}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceEmpty(t, doc.Keyvals)
}

func TestKeyvalHTMX_Post_TooLongKey_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	setupGithubDocument(t, store, mocks.Nobody)

	long := strings.Repeat("a", 257)
	r := postKeyvalForm(t, http.MethodPost,
		"/internal/keyval/create/miscellaneous/github",
		url.Values{"key": {long}, "value": {"v"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceEmpty(t, doc.Keyvals)
}

func TestKeyvalHTMX_Post_DuplicateKey_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	setupGithubDocument(t, store, mocks.Nobody)
	insertKeyval(t, store, mocks.Nobody, "shared", "first", crypt)

	r := postKeyvalForm(t, http.MethodPost,
		"/internal/keyval/create/miscellaneous/github",
		url.Values{"key": {"shared"}, "value": {"second"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Keyvals)
	must.Eq(t, "shared", doc.Keyvals[0].K.Unveil())
	must.Eq(t, "first", doc.Keyvals[0].Unlock(crypt.Vault()))
}

func TestKeyvalHTMX_Post_TrimsKeyWhitespace(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	setupGithubDocument(t, store, mocks.Nobody)

	r := postKeyvalForm(t, http.MethodPost,
		"/internal/keyval/create/miscellaneous/github",
		url.Values{"key": {"  pretty_key  "}, "value": {"v"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Keyvals)
	must.Eq(t, "pretty_key", doc.Keyvals[0].K.Unveil())
}

func TestKeyvalHTMX_Post_PreservesValueWhitespace(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	setupGithubDocument(t, store, mocks.Nobody)

	r := postKeyvalForm(t, http.MethodPost,
		"/internal/keyval/create/miscellaneous/github",
		url.Values{"key": {"k"}, "value": {"  spaced  "}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Keyvals)
	must.Eq(t, "  spaced  ", doc.Keyvals[0].Unlock(crypt.Vault()))
}

func TestKeyvalHTMX_Post_MissingVault_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	setupGithubDocument(t, store, mocks.Nobody)

	r := postKeyvalForm(t, http.MethodPost,
		"/internal/keyval/create/miscellaneous/github",
		url.Values{"key": {"k"}, "value": {"v"}},
	)
	r.AddCookie(cookie)
	// intentionally no vault cookie

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestKeyvalHTMX_Post_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobKeyval(t, store)

	// nobody owns the document
	setupGithubDocument(t, store, mocks.Nobody)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	r := postKeyvalForm(t, http.MethodPost,
		"/internal/keyval/create/miscellaneous/github",
		url.Values{"key": {"k"}, "value": {"v"}},
	)
	r.AddCookie(bobCookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestKeyvalEntryHTMX_Get_RendersEditForm(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	setupGithubDocument(t, store, mocks.Nobody)
	id := insertKeyval(t, store, mocks.Nobody, "redis_url", "redis://localhost:6379", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := keyvalURL("edit", "miscellaneous", "github", id)
	r := requestGet(t, u, cookie)
	r.AddCookie(crypt.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)

	body := w.Body.String()
	must.StrContains(t, body, `hx-put="/internal/keyval/update/`)
	must.StrContains(t, body, `value="redis_url"`)
	must.StrContains(t, body, `value="redis://localhost:6379"`)
}

func TestKeyvalEntryHTMX_Get_MissingVault_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	setupGithubDocument(t, store, mocks.Nobody)
	id := insertKeyval(t, store, mocks.Nobody, "k", "v", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := keyvalURL("edit", "miscellaneous", "github", id)
	r := requestGet(t, u, cookie)
	// intentionally no vault cookie

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestKeyvalEntryHTMX_Get_WrongKeyvalID_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	setupGithubDocument(t, store, mocks.Nobody)
	insertKeyval(t, store, mocks.Nobody, "k", "v", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := keyvalURL("edit", "miscellaneous", "github", records.KeyvalsAuto(9999))
	r := requestGet(t, u, cookie)
	r.AddCookie(crypt.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestKeyvalEntryHTMX_Get_UnknownAction_Aborts(t *testing.T) {
	t.Parallel()

	router, scache := makeKeyvalMux(t, mockdb.Open(t))
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/keyval/whatever/miscellaneous/github/1", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestKeyvalEntryHTMX_Put_HappyPath_ReturnsRow(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	setupGithubDocument(t, store, mocks.Nobody)
	id := insertKeyval(t, store, mocks.Nobody, "oldkey", "oldvalue", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := keyvalURL("update", "miscellaneous", "github", id)
	r := postKeyvalForm(t, http.MethodPut, u,
		url.Values{"key": {"newkey"}, "value": {"newvalue"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)

	body := w.Body.String()
	must.StrContains(t, body, `class="keyval-row"`)
	must.StrContains(t, body, `data-value="newvalue"`)
	must.StrContains(t, body, `title="newkey"`)
	must.StrContains(t, body, `>newkey<`)

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Keyvals)
	must.Eq(t, "newkey", doc.Keyvals[0].K.Unveil())
	must.Eq(t, "newvalue", doc.Keyvals[0].Unlock(crypt.Vault()))
}

func TestKeyvalEntryHTMX_Put_DuplicateKey_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	setupGithubDocument(t, store, mocks.Nobody)
	insertKeyval(t, store, mocks.Nobody, "first", "value1", crypt)
	id := insertKeyval(t, store, mocks.Nobody, "second", "value2", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := keyvalURL("update", "miscellaneous", "github", id)
	r := postKeyvalForm(t, http.MethodPut, u,
		url.Values{"key": {"first"}, "value": {"hijacked"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 2, doc.Keyvals)
}

func TestKeyvalEntryHTMX_Put_EmptyValue_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	setupGithubDocument(t, store, mocks.Nobody)
	id := insertKeyval(t, store, mocks.Nobody, "k", "realvalue", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := keyvalURL("update", "miscellaneous", "github", id)
	r := postKeyvalForm(t, http.MethodPut, u,
		url.Values{"key": {"k"}, "value": {""}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Keyvals)
	must.Eq(t, "realvalue", doc.Keyvals[0].Unlock(crypt.Vault()))
}

func TestKeyvalEntryHTMX_Put_UnknownAction_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	setupGithubDocument(t, store, mocks.Nobody)
	id := insertKeyval(t, store, mocks.Nobody, "k", "v", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := keyvalURL("wat", "miscellaneous", "github", id)
	r := postKeyvalForm(t, http.MethodPut, u,
		url.Values{"key": {"k"}, "value": {"v"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestKeyvalEntryHTMX_Put_MissingVault_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	setupGithubDocument(t, store, mocks.Nobody)
	id := insertKeyval(t, store, mocks.Nobody, "k", "v", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := keyvalURL("update", "miscellaneous", "github", id)
	r := postKeyvalForm(t, http.MethodPut, u,
		url.Values{"key": {"k"}, "value": {"v"}},
	)
	r.AddCookie(cookie)
	// intentionally no vault cookie

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestKeyvalEntryHTMX_Put_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobKeyval(t, store)
	setupGithubDocument(t, store, mocks.Nobody)
	id := insertKeyval(t, store, mocks.Nobody, "k", "realvalue", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	u := keyvalURL("update", "miscellaneous", "github", id)
	r := postKeyvalForm(t, http.MethodPut, u,
		url.Values{"key": {"k"}, "value": {"hacked"}},
	)
	r.AddCookie(bobCookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestKeyvalEntryHTMX_Delete_HappyPath_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	setupGithubDocument(t, store, mocks.Nobody)
	id := insertKeyval(t, store, mocks.Nobody, "k", "v", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := keyvalURL("delete", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(cookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.Eq(t, "", w.Body.String())

	doc := fetchDocKeyvals(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceEmpty(t, doc.Keyvals)
}

func TestKeyvalEntryHTMX_Delete_UnknownAction_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	setupGithubDocument(t, store, mocks.Nobody)
	id := insertKeyval(t, store, mocks.Nobody, "k", "v", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := keyvalURL("wat", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(cookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestKeyvalEntryHTMX_Delete_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobKeyval(t, store)
	setupGithubDocument(t, store, mocks.Nobody)
	id := insertKeyval(t, store, mocks.Nobody, "k", "v", crypt)

	router, scache := makeKeyvalMux(t, store)
	ss := sessions(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	u := keyvalURL("delete", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(bobCookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}
