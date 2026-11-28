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

// newWeblinkRouter mounts both weblink HTMX handlers in a mux router with
// the sessions middleware so requests can carry user identity. Returns the
// router and the session store so callers can mint cookies.
func newWeblinkRouter(t *testing.T, store database.Store) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()
	router.Handle((&weblinkHTMX{}).Path(), &weblinkHTMX{
		log:   ulog.New("test"),
		store: store,
	})
	router.Handle((&weblinkEntryHTMX{}).Path(), &weblinkEntryHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	return router, scache
}

// insertWeblink stands up a document under the default folder and creates a
// weblink through the store using the supplied cypher.
func insertWeblink(t *testing.T, store database.Store, user users.Identity,
	doc, url string, cypher mocks.TestCypher) records.WeblinksAuto {
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

	ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New(url))
	must.NoError(t, cerr)

	id, err := store.Documents().CreateWeblink(ctx, &sitecontent.CreateWeblinkRequest{
		User:      user,
		Folder:    folder,
		Document:  doc,
		URL:       url,
		Encrypted: ciphertext,
	})
	must.NoError(t, err)
	return id
}

// ensureBob provisions a second user (id=2) on the given store so wrong-user
// tests have a real distinct identity.
func ensureBob(t *testing.T, store database.Store) users.Identity {
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

// weblinkURL joins the path segments after /internal/weblink/... for the
// given action keyword and folder/document/weblink-id triple.
func weblinkURL(action, folder, document string, weblinkID records.WeblinksAuto) string {
	return path.Join("/internal/weblink", action, folder, document, strconv.Itoa(int(weblinkID)))
}

// postForm builds a POST/PUT request with an x-www-form-urlencoded body
// suitable for forms.MustParse on the handler side.
func postForm(t *testing.T, method, urlPath string, values url.Values) *http.Request {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), method, urlPath, strings.NewReader(values.Encode()))
	must.NoError(t, err)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

// fetchDoc reads back a single document for assertion.
func fetchDoc(t *testing.T, store database.Store, user users.Identity,
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

// --- weblinkHTMX (add flow) ---

func TestWeblinkHTMX_Get_RendersForm(t *testing.T) {
	t.Parallel()

	router, scache := newWeblinkRouter(t, mockdb.Open(t))
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/weblink/add/miscellaneous/github", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `hx-post="/internal/weblink/create/`)
	must.StrContains(t, body, `name="url"`)
	must.StrContains(t, body, `placeholder="https://example.com"`)
}

func TestWeblinkHTMX_Get_NoSession_Redirects(t *testing.T) {
	t.Parallel()

	router, scache := newWeblinkRouter(t, mockdb.Open(t))
	ss := session(router, scache)

	// No cookie => session middleware sets an inactive session; the handler
	// emits an HX-Redirect to /login so the page can recover via a fresh login.
	r := requestGet(t, "/internal/weblink/add/miscellaneous/github", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestWeblinkHTMX_Get_List_RendersLinks(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()
	insertWeblink(t, store, mocks.Nobody, "github", "https://example.com", cypher)

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/weblink/list/miscellaneous/github", cookie)
	r.AddCookie(cypher.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `class="weblink-menu-list"`)
	must.StrContains(t, body, `href="https://example.com"`)
}

func TestWeblinkHTMX_Get_List_MultipleLinksOrdered(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()

	// create a document and two weblinks directly (insertWeblink only adds one)
	{
		ctx, cancel := scope.TTL(testwait)
		defer cancel()

		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github", Slug: "github",
		})
		must.NoError(t, err)

		for _, u := range []string{"https://first.example.com", "https://second.example.com"} {
			ciphertext, cerr := cypher.Cypher().Encrypt(conceal.New(u))
			must.NoError(t, cerr)
			_, err = store.Documents().CreateWeblink(ctx, &sitecontent.CreateWeblinkRequest{
				User: mocks.Nobody, Folder: folder, Document: "github", URL: u, Encrypted: ciphertext,
			})
			must.NoError(t, err)
		}
	}

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/weblink/list/miscellaneous/github", cookie)
	r.AddCookie(cypher.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	first := strings.Index(body, "https://first.example.com")
	second := strings.Index(body, "https://second.example.com")
	must.True(t, first >= 0)
	must.True(t, second >= 0)
	must.True(t, first < second)
}

func TestWeblinkHTMX_Get_List_NoWeblinks(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()

	// a document with no weblinks
	{
		ctx, cancel := scope.TTL(testwait)
		defer cancel()

		list, err := store.Folders().List(ctx, mocks.Nobody)
		must.NoError(t, err)
		folder := list[0].Label
		_, err = store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User: mocks.Nobody, Folder: folder, Label: "github", Slug: "github",
		})
		must.NoError(t, err)
	}

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/weblink/list/miscellaneous/github", cookie)
	r.AddCookie(cypher.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `class="weblink-menu-list"`)
	must.StrNotContains(t, w.Body.String(), "https://")
}

func TestWeblinkHTMX_Get_List_MissingCypher_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()
	insertWeblink(t, store, mocks.Nobody, "github", "https://example.com", cypher)

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/weblink/list/miscellaneous/github", cookie)
	// intentionally no cypher cookie
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestWeblinkHTMX_Post_HappyPath_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()

	router, scache := newWeblinkRouter(t, store)
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

	r := postForm(t, http.MethodPost,
		"/internal/weblink/create/miscellaneous/github",
		url.Values{"url": {"https://example.com"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(cypher.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/folders/miscellaneous/github", w.Header().Get("HX-Redirect"))

	doc := fetchDoc(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Weblinks)
	must.Eq(t, "https://example.com", doc.Weblinks[0].Unlock(cypher.Cypher()))
}

func TestWeblinkHTMX_Post_EmptyURL_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()

	router, scache := newWeblinkRouter(t, store)
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

	r := postForm(t, http.MethodPost,
		"/internal/weblink/create/miscellaneous/github",
		url.Values{"url": {""}},
	)
	r.AddCookie(cookie)
	r.AddCookie(cypher.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDoc(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 0, doc.Weblinks)
}

func TestWeblinkHTMX_Post_MissingCypher_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := postForm(t, http.MethodPost,
		"/internal/weblink/create/miscellaneous/github",
		url.Values{"url": {"https://example.com"}},
	)
	r.AddCookie(cookie)
	// intentionally no cypher cookie

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestWeblinkHTMX_Post_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()
	bob := ensureBob(t, store)

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

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	r := postForm(t, http.MethodPost,
		"/internal/weblink/create/miscellaneous/github",
		url.Values{"url": {"https://attacker.example.com"}},
	)
	r.AddCookie(bobCookie)
	r.AddCookie(cypher.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

// --- weblinkEntryHTMX (edit/delete flow) ---

func TestWeblinkEntryHTMX_Get_RendersEditForm(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()
	id := insertWeblink(t, store, mocks.Nobody, "github", "https://example.com", cypher)

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := weblinkURL("edit", "miscellaneous", "github", id)
	r := requestGet(t, u, cookie)
	r.AddCookie(cypher.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `hx-put="/internal/weblink/update/`)
	must.StrContains(t, body, `value="https://example.com"`)
	must.StrContains(t, body, `<span id="error">`)
}

func TestWeblinkEntryHTMX_Put_HappyPath_ReturnsNewRow(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()
	id := insertWeblink(t, store, mocks.Nobody, "github", "https://original.example.com", cypher)

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := weblinkURL("update", "miscellaneous", "github", id)
	r := postForm(t, http.MethodPut, u, url.Values{"url": {"https://updated.example.com"}})
	r.AddCookie(cookie)
	r.AddCookie(cypher.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `href="https://updated.example.com"`)
	must.StrContains(t, body, `class="weblink-row"`)

	doc := fetchDoc(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Weblinks)
	must.Eq(t, "https://updated.example.com", doc.Weblinks[0].Unlock(cypher.Cypher()))
}

func TestWeblinkEntryHTMX_Put_InvalidURL_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()
	id := insertWeblink(t, store, mocks.Nobody, "github", "https://original.example.com", cypher)

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := weblinkURL("update", "miscellaneous", "github", id)
	r := postForm(t, http.MethodPut, u, url.Values{"url": {""}})
	r.AddCookie(cookie)
	r.AddCookie(cypher.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	// original row remains untouched
	doc := fetchDoc(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 1, doc.Weblinks)
	must.Eq(t, "https://original.example.com", doc.Weblinks[0].Unlock(cypher.Cypher()))
}

func TestWeblinkEntryHTMX_Put_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()
	bob := ensureBob(t, store)
	id := insertWeblink(t, store, mocks.Nobody, "github", "https://example.com", cypher)

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	u := weblinkURL("update", "miscellaneous", "github", id)
	r := postForm(t, http.MethodPut, u, url.Values{"url": {"https://attacker.example.com"}})
	r.AddCookie(bobCookie)
	r.AddCookie(cypher.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestWeblinkEntryHTMX_Delete_HappyPath_EmptyBody(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()
	id := insertWeblink(t, store, mocks.Nobody, "github", "https://example.com", cypher)

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := weblinkURL("delete", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(cookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.Eq(t, "", w.Body.String())

	doc := fetchDoc(t, store, mocks.Nobody, "miscellaneous", "github")
	must.SliceLen(t, 0, doc.Weblinks)
}

func TestWeblinkEntryHTMX_Delete_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	cypher := mocks.NewTestCypher()
	bob := ensureBob(t, store)
	id := insertWeblink(t, store, mocks.Nobody, "github", "https://example.com", cypher)

	router, scache := newWeblinkRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	u := weblinkURL("delete", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(bobCookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}
