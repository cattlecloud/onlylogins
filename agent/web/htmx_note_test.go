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

func newNoteRouter(t *testing.T, store database.Store) (*mux.Router, *mocks.Sessions) {
	t.Helper()

	scache := mocks.NewSessions()
	router := mux.NewRouter()

	router.Handle((&noteHTMX{}).Path(), &noteHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	router.Handle((&noteEntryHTMX{}).Path(), &noteEntryHTMX{
		log:   ulog.New("test"),
		store: store,
	})

	return router, scache
}

func insertNoteHandlerHelper(t *testing.T, store database.Store, user users.Identity,
	doc, note string, crypt mocks.TestCrypt) records.NotesAuto {
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

	ciphertext, cerr := crypt.Vault().Encrypt(conceal.New(note))
	must.NoError(t, cerr)

	id, err := store.Documents().CreateNote(ctx, &sitecontent.CreateNoteRequest{
		User:      user,
		Folder:    folder,
		Document:  doc,
		Encrypted: ciphertext,
	})
	must.NoError(t, err)
	return id
}

func ensureBobNote(t *testing.T, store database.Store) users.Identity {
	t.Helper()
	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	salt := encrypt.Salt()
	passphrase := conceal.New("pass+bob long enough")
	key := encrypt.DeriveKey(passphrase, salt)
	verifier := encrypt.MakeVerifier(key)

	id, err := store.Users().Create(ctx, &users.CreateUserRequest{
		Email:      "bob-note@example.com",
		Username:   "bob-note",
		Passphrase: passphrase,
		Salt:       salt,
	}, verifier)
	must.NoError(t, err)
	must.Positive(t, id)
	return id
}

func noteURL(action, folder, document string, noteID records.NotesAuto) string {
	return path.Join("/internal/note", action, folder, document, strconv.Itoa(int(noteID)))
}

func postNoteForm(t *testing.T, method, urlPath string, values url.Values) *http.Request {
	t.Helper()

	r, err := http.NewRequestWithContext(t.Context(), method, urlPath, strings.NewReader(values.Encode()))
	must.NoError(t, err)

	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func fetchDocNote(t *testing.T, store database.Store, user users.Identity, folder, document string) *sitecontent.Document {
	t.Helper()

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	doc, err := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User: user, Folder: folder, Document: document,
	})
	must.NoError(t, err)

	return doc
}

func TestNoteHTMX_Get_RendersForm(t *testing.T) {
	t.Parallel()

	router, scache := newNoteRouter(t, mockdb.Open(t))
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/note/add/miscellaneous/github", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)

	body := w.Body.String()
	must.StrContains(t, body, `hx-post="/internal/note/create/`)
	must.StrContains(t, body, `name="note"`)
	must.StrContains(t, body, `<textarea`)
}

func TestNoteHTMX_Get_NoSession_Redirects(t *testing.T) {
	t.Parallel()

	router, scache := newNoteRouter(t, mockdb.Open(t))
	ss := session(router, scache)

	r := requestGet(t, "/internal/note/add/miscellaneous/github", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestNoteHTMX_Post_HappyPath_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newNoteRouter(t, store)
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

	note := "secret sauce is mayo"
	r := postNoteForm(t, http.MethodPost,
		"/internal/note/create/miscellaneous/github",
		url.Values{"note": {note}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/folders/miscellaneous/github", w.Header().Get("HX-Redirect"))

	doc := fetchDocNote(t, store, mocks.Nobody, "miscellaneous", "github")
	must.NotNil(t, doc.Note)
	must.Eq(t, note, doc.Note.Unlock(crypt.Vault()))
}

func TestNoteHTMX_Post_EmptyNote_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newNoteRouter(t, store)
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

	r := postNoteForm(t, http.MethodPost,
		"/internal/note/create/miscellaneous/github",
		url.Values{"note": {""}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocNote(t, store, mocks.Nobody, "miscellaneous", "github")
	must.Nil(t, doc.Note)
}

func TestNoteHTMX_Post_TooLongNote_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newNoteRouter(t, store)
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

	long := strings.Repeat("a", 6001)
	r := postNoteForm(t, http.MethodPost,
		"/internal/note/create/miscellaneous/github",
		url.Values{"note": {long}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocNote(t, store, mocks.Nobody, "miscellaneous", "github")
	must.Nil(t, doc.Note)
}

func TestNoteHTMX_Post_PreservesWhitespace(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newNoteRouter(t, store)
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

	r := postNoteForm(t, http.MethodPost,
		"/internal/note/create/miscellaneous/github",
		url.Values{"note": {"  spaced  "}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)

	doc := fetchDocNote(t, store, mocks.Nobody, "miscellaneous", "github")
	must.NotNil(t, doc.Note)
	must.Eq(t, "  spaced  ", doc.Note.Unlock(crypt.Vault()))
}

func TestNoteHTMX_Post_PreservesNewlines(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newNoteRouter(t, store)
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

	multi := "hello\nworld\n\tindented"
	r := postNoteForm(t, http.MethodPost,
		"/internal/note/create/miscellaneous/github",
		url.Values{"note": {multi}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)

	doc := fetchDocNote(t, store, mocks.Nobody, "miscellaneous", "github")
	must.NotNil(t, doc.Note)
	must.Eq(t, multi, doc.Note.Unlock(crypt.Vault()))
}

func TestNoteHTMX_Post_AlreadyExists_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "first", crypt)

	r := postNoteForm(t, http.MethodPost,
		"/internal/note/create/miscellaneous/github",
		url.Values{"note": {"second"}},
	)
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	// duplicate-note insert fails the UNIQUE constraint; the handler
	// routes DB errors to abort() with a 500 response
	must.Eq(t, http.StatusInternalServerError, w.Code)

	// original is preserved
	doc := fetchDocNote(t, store, mocks.Nobody, "miscellaneous", "github")
	must.NotNil(t, doc.Note)
	must.Eq(t, "first", doc.Note.Unlock(crypt.Vault()))
}

func TestNoteHTMX_Post_MissingVault_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := postNoteForm(t, http.MethodPost,
		"/internal/note/create/miscellaneous/github",
		url.Values{"note": {"hi"}},
	)
	r.AddCookie(cookie)
	// intentionally no vault cookie

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestNoteHTMX_Post_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobNote(t, store)

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

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	r := postNoteForm(t, http.MethodPost,
		"/internal/note/create/miscellaneous/github",
		url.Values{"note": {"hacked"}},
	)
	r.AddCookie(bobCookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestNoteEntryHTMX_Get_RendersEditForm(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "something clever", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	// server decrypts the note itself using the vault cookie, so no ?note= query param
	u := noteURL("edit", "miscellaneous", "github", id)
	r := requestGet(t, u, cookie)
	r.AddCookie(crypt.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)

	body := w.Body.String()
	must.StrContains(t, body, `hx-put="/internal/note/update/`)
	must.StrContains(t, body, `>something clever<`)
	must.StrContains(t, body, `<span id="error">`)
}

func TestNoteEntryHTMX_Get_PreservesNewlinesInEditForm(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "first\nsecond", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	doc, err := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
	})
	must.NoError(t, err)
	must.NotNil(t, doc.Note)

	u := noteURL("edit", "miscellaneous", "github", doc.Note.ID)
	r := requestGet(t, u, cookie)
	r.AddCookie(crypt.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)

	body := w.Body.String()
	must.StrContains(t, body, "first\nsecond")
}

func TestNoteEntryHTMX_Get_MissingVault_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "anything", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	ctx, cancel := scope.TTL(testwait)
	defer cancel()
	doc, err := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User: mocks.Nobody, Folder: "miscellaneous", Document: "github",
	})
	must.NoError(t, err)
	must.NotNil(t, doc.Note)

	u := noteURL("edit", "miscellaneous", "github", doc.Note.ID)
	r := requestGet(t, u, cookie)
	// intentionally no vault cookie

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestNoteEntryHTMX_Get_WrongNoteID_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "realtext", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := noteURL("edit", "miscellaneous", "github", records.NotesAuto(9999))
	r := requestGet(t, u, cookie)
	r.AddCookie(crypt.Cookie())
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestNoteEntryHTMX_Get_UnknownAction_Aborts(t *testing.T) {
	t.Parallel()

	router, scache := newNoteRouter(t, mockdb.Open(t))
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	r := requestGet(t, "/internal/note/whatever/miscellaneous/github/1", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestNoteEntryHTMX_Put_HappyPath_ReturnsPanel(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "oldtext", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := noteURL("update", "miscellaneous", "github", id)
	r := postNoteForm(t, http.MethodPut, u, url.Values{"note": {"newtext"}})
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `class="note-area"`)
	must.StrContains(t, body, `class="note-panel"`)
	must.StrContains(t, body, `data-note="newtext"`)
	must.StrContains(t, body, `note-reveal-btn`)

	doc := fetchDocNote(t, store, mocks.Nobody, "miscellaneous", "github")
	must.NotNil(t, doc.Note)
	must.Eq(t, "newtext", doc.Note.Unlock(crypt.Vault()))
}

func TestNoteEntryHTMX_Put_PreservesNewlines(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "oldtext", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := noteURL("update", "miscellaneous", "github", id)
	multi := "line1\nline2\n\tindented"
	r := postNoteForm(t, http.MethodPut, u, url.Values{"note": {multi}})
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)

	doc := fetchDocNote(t, store, mocks.Nobody, "miscellaneous", "github")
	must.NotNil(t, doc.Note)
	must.Eq(t, multi, doc.Note.Unlock(crypt.Vault()))
}

func TestNoteEntryHTMX_Put_EmptyNote_ReturnsProblem(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "realtext", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := noteURL("update", "miscellaneous", "github", id)
	r := postNoteForm(t, http.MethodPut, u, url.Values{"note": {""}})
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)

	doc := fetchDocNote(t, store, mocks.Nobody, "miscellaneous", "github")
	must.NotNil(t, doc.Note)
	must.Eq(t, "realtext", doc.Note.Unlock(crypt.Vault()))
}

func TestNoteEntryHTMX_Put_UnknownAction_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "realtext", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := noteURL("what", "miscellaneous", "github", id)
	r := postNoteForm(t, http.MethodPut, u, url.Values{"note": {"hi"}})
	r.AddCookie(cookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestNoteEntryHTMX_Put_MissingVault_Redirects(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "realtext", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := noteURL("update", "miscellaneous", "github", id)
	r := postNoteForm(t, http.MethodPut, u, url.Values{"note": {"hi"}})
	r.AddCookie(cookie)
	// intentionally no vault cookie

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}

func TestNoteEntryHTMX_Put_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobNote(t, store)
	id := insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "realtext", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	u := noteURL("update", "miscellaneous", "github", id)
	r := postNoteForm(t, http.MethodPut, u, url.Values{"note": {"hacked"}})
	r.AddCookie(bobCookie)
	r.AddCookie(crypt.Cookie())

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestNoteEntryHTMX_Delete_HappyPath_RestoresAddButton(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "secret", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := noteURL("delete", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(cookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	// response carries an out-of-band swap that re-shows the "+ Note" button
	must.StrContains(t, body, `id="note-add-btn"`)
	must.StrContains(t, body, `hx-swap-oob="true"`)
	must.StrContains(t, body, `hx-get="/internal/note/add/miscellaneous/github"`)

	doc := fetchDocNote(t, store, mocks.Nobody, "miscellaneous", "github")
	must.Nil(t, doc.Note)
}

func TestNoteEntryHTMX_Delete_UnknownAction_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	id := insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "realtext", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	cookie := scache.Create(mocks.Nobody, 1*time.Minute)

	u := noteURL("wat", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(cookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}

func TestNoteEntryHTMX_Delete_WrongUser_Aborts(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	crypt := mocks.NewTestCrypt()
	bob := ensureBobNote(t, store)
	id := insertNoteHandlerHelper(t, store, mocks.Nobody, "github", "realtext", crypt)

	router, scache := newNoteRouter(t, store)
	ss := session(router, scache)
	bobCookie := scache.Create(bob, 1*time.Minute)

	u := noteURL("delete", "miscellaneous", "github", id)
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, u, nil)
	must.NoError(t, err)
	r.AddCookie(bobCookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)

	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "unable to process request", w.Body.String())
}
