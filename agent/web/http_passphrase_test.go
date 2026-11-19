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
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/ui"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

// newPassphraseHandler returns a wrapped passphrase handler plus the session
// cache backing it, for exercising the endpoint in tests.
func newPassphraseHandler(t *testing.T, store database.Store) (*mocks.Sessions, http.Handler) {
	t.Helper()

	scache := mocks.NewSessions()
	handler := &passphraseHTTP{
		log:     ulog.New("test"),
		store:   store,
		session: scache,
		html:    ui.Load("passphrase.html"),
	}

	return scache, session(handler, scache)
}

// newPassphraseUser creates a user with a known passphrase, returning the user
// id and the derived encryption key.
func newPassphraseUser(t *testing.T, store database.Store, username, passphrase string) (users.Identity, *conceal.Bytes) {
	t.Helper()

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	salt := encrypt.Salt()
	pass := conceal.New(passphrase)
	key := encrypt.DeriveKey(pass, salt)
	verifier := encrypt.MakeVerifier(key)

	id, err := store.Users().Create(ctx, &users.CreateUserRequest{
		Email:      username + "@example.com",
		Username:   username,
		Salt:       salt,
		Passphrase: pass,
	}, verifier)
	must.NoError(t, err)
	must.Positive(t, id)

	return id, key
}

// preparePassword creates a "github" document and one password encrypted with the
// given vault.
func preparePassword(t *testing.T, store database.Store, uid users.Identity, vault encrypt.Vault, plaintext string) {
	t.Helper()

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	_, err := store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User:   uid,
		Folder: "miscellaneous",
		Label:  "github",
		Slug:   "github",
	})
	must.NoError(t, err)

	ciphertext, cerr := vault.Encrypt(conceal.New(plaintext))
	must.NoError(t, cerr)

	_, err = store.Documents().CreatePassword(ctx, &sitecontent.CreatePasswordRequest{
		User:      uid,
		Folder:    "miscellaneous",
		Document:  "github",
		Encrypted: ciphertext,
	})
	must.NoError(t, err)
}

func TestPassphrase_GET_not_logged_in(t *testing.T) {
	t.Parallel()

	_, handler := newPassphraseHandler(t, mockdb.Open(t))

	request := requestGet(t, "/profile/passphrase", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)

	must.Eq(t, http.StatusTemporaryRedirect, w.Code)
	must.Eq(t, "/login", w.Header().Get("Location"))
}

func TestPassphrase_GET_logged_in(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	scache, handler := newPassphraseHandler(t, store)

	cookie := scache.Create(mocks.Nobody, time.Minute)
	request := requestGet(t, "/profile/passphrase", cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `hx-post="/profile/passphrase"`)
	must.StrContains(t, body, `name="previous"`)
	must.StrContains(t, body, `name="passphrase"`)
	must.StrContains(t, body, `name="confirm"`)
	must.StrContains(t, body, `<span id="error">`)
}

func TestPassphrase_POST_happy_path(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	scache, handler := newPassphraseHandler(t, store)

	const oldPass = "correct horse battery staple"
	const newPass = "brand new battery horse staple"

	uid, oldKey := newPassphraseUser(t, store, "alice", oldPass)
	preparePassword(t, store, uid, encrypt.New(oldKey), "hunter2hunter2hunter2")

	cookie := scache.Create(uid, time.Minute)
	request := requestPost(t, "/profile/passphrase", cookie, url.Values{
		"previous":   {oldPass},
		"passphrase": {newPass},
		"confirm":    {newPass},
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, "passphrase has been updated")
	must.StrContains(t, body, `href="/login"`)

	// the session and encryption cookies are cleared, forcing re-login
	clearedSession := false
	clearedEncryption := false
	for _, c := range w.Result().Cookies() {
		switch c.Name {
		case "onlylogins-session":
			clearedSession = c.Value == ""
		case "onlylogins-encryption":
			clearedEncryption = c.Value == ""
		}
	}
	must.True(t, clearedSession)
	must.True(t, clearedEncryption)

	ctx, cancel := scope.TTL(testwait)
	defer cancel()

	// the old passphrase no longer verifies
	_, _, err := store.Users().Check(ctx, &users.LoginRequest{
		Username:   "alice",
		Passphrase: conceal.New(oldPass),
	})
	must.ErrorIs(t, err, database.ErrIncorrectCredentials)

	// the new passphrase verifies and re-encrypted the stored content
	_, newKey, err := store.Users().Check(ctx, &users.LoginRequest{
		Username:   "alice",
		Passphrase: conceal.New(newPass),
	})
	must.NoError(t, err)

	doc, derr := store.Documents().Get(ctx, &sitecontent.GetDocumentRequest{
		User:     uid,
		Folder:   "miscellaneous",
		Document: "github",
	})
	must.NoError(t, derr)
	must.Eq(t, "hunter2hunter2hunter2", doc.Passwords[0].Unlock(encrypt.New(newKey)))
}

func TestPassphrase_POST_wrong_old_passphrase(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	scache, handler := newPassphraseHandler(t, store)

	const oldPass = "correct horse battery staple"
	const newPass = "brand new battery horse staple"

	uid, _ := newPassphraseUser(t, store, "alice", oldPass)

	cookie := scache.Create(uid, time.Minute)
	request := requestPost(t, "/profile/passphrase", cookie, url.Values{
		"previous":   {"wrong old passphrase here"},
		"passphrase": {newPass},
		"confirm":    {newPass},
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, w.Body.String(), database.ErrIncorrectCredentials.Error())
}

func TestPassphrase_POST_mismatched_confirm(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	scache, handler := newPassphraseHandler(t, store)

	const oldPass = "correct horse battery staple"
	const newPass = "brand new battery horse staple"

	uid, _ := newPassphraseUser(t, store, "alice", oldPass)

	cookie := scache.Create(uid, time.Minute)
	request := requestPost(t, "/profile/passphrase", cookie, url.Values{
		"previous":   {oldPass},
		"passphrase": {newPass},
		"confirm":    {"different battery horse staple"},
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, w.Body.String(), "passphrases do not match")
}

func TestPassphrase_POST_short_passphrase(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	scache, handler := newPassphraseHandler(t, store)

	const oldPass = "correct horse battery staple"

	uid, _ := newPassphraseUser(t, store, "alice", oldPass)

	cookie := scache.Create(uid, time.Minute)
	request := requestPost(t, "/profile/passphrase", cookie, url.Values{
		"previous":   {oldPass},
		"passphrase": {"too short"},
		"confirm":    {"too short"},
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, w.Body.String(), "passphrase must be at least 24 characters long")
}

func TestPassphrase_POST_not_logged_in(t *testing.T) {
	t.Parallel()

	_, handler := newPassphraseHandler(t, mockdb.Open(t))

	request := requestPost(t, "/profile/passphrase", nil, url.Values{
		"previous":   {"correct horse battery staple"},
		"passphrase": {"brand new battery horse staple"},
		"confirm":    {"brand new battery horse staple"},
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/login", w.Header().Get("HX-Redirect"))
}
