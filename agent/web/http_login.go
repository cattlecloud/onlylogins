// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

type loginHTTP struct {
	log         *ulog.Log
	store       database.Store
	sessions    middles.Sessions[users.Identity]
	environment runtime.Environment
	html        *template.Template
}

func (h *loginHTTP) Path() string {
	return "/login"
}

func (h *loginHTTP) Methods() []string {
	return []string{http.MethodGet, http.MethodPost, http.MethodPut}
}

func (h *loginHTTP) Metric() string {
	return "login"
}

func (h *loginHTTP) Consumer() consumer {
	return normal
}

func (h *loginHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:  h.get,
			http.MethodPost: h.post,
		},
	)
}

func (h *loginHTTP) get(_ scope.C, w http.ResponseWriter, _ *http.Request) error {
	type page struct {
		Canonical template.URL
	}

	content := &page{
		Canonical: h.environment.Canonical("%s", h.Path()),
	}

	// render the login page
	if err := h.html.Execute(w, content); err != nil {
		return fmt.Errorf("unable to render login page: %w", err)
	}

	return nil
}

// put handles new signups
func (h *loginHTTP) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	switch r.URL.Query().Get("do") {
	case "login":
		return h.login(ctx, w, r)
	case "create":
		return h.signup(ctx, w, r)
	}

	return errors.New("no such login action")
}

func (*loginHTTP) signupFields(r *http.Request) *users.CreateUserRequest {
	var email, username string
	var passphrase, confirm *conceal.Text
	var notifications, spam bool
	forms.MustParse(r, forms.Schema{
		"signup-email":      forms.String(&email),
		"signup-username":   forms.String(&username),
		"signup-passphrase": forms.Secret(&passphrase),
		"signup-confirm":    forms.Secret(&confirm),
		"notifications":     forms.BoolOr(&notifications, false),
		"marketing":         forms.BoolOr(&spam, false),
	})

	return &users.CreateUserRequest{
		Email:         email,
		Username:      username,
		Passphrase:    passphrase,
		Confirm:       confirm,
		Notifications: notifications,
		Spam:          spam,
	}
}

func (*loginHTTP) loginFields(r *http.Request) *users.LoginRequest {
	var username string
	var passphrase *conceal.Text
	forms.MustParse(r, forms.Schema{
		"username":   forms.String(&username),
		"passphrase": forms.Secret(&passphrase),
	})
	o := webtools.Origins(r)
	return &users.LoginRequest{
		Username:   username,
		Passphrase: passphrase,
		UserAgent:  o.UserAgent.String,
		IP:         o.From(),
	}
}

func (h *loginHTTP) login(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	request := h.loginFields(r)

	if err := request.Validate(); err != nil {
		go func() { h.audit(request, false) }()
		return problem(w, err)
	}

	// verify the passphrase and recover the derived key
	id, key, cerr := h.store.Users().Check(ctx, request)
	switch {
	case errors.Is(cerr, database.ErrUserNotFound):
		// no such user; record under bogus table
		go func() { h.bogus(request) }()
		// respond with the generic "incorrect credentials" message so
		// usernames cannot be enumerated from the response
		return problem(w, database.ErrIncorrectCredentials)
	case cerr != nil:
		go func() { h.audit(request, false) }()
		return problem(w, cerr)
	default:
		go func() { h.audit(request, true) }()
	}

	// set the key cookie
	http.SetCookie(w, h.cookie(key))

	// set the session cookie
	http.SetCookie(w, h.sessions.Create(id, cookieTTL))

	// send user to the homepage
	return htmx.SetRedirect(w, "/")
}

func (h *loginHTTP) signup(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	request := h.signupFields(r)

	// generate the per-user salt
	request.Salt = encrypt.Salt()

	// ensure the request contains valid parameters
	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	// ensure the user actually typed their passphrase twice and got a match.
	// the form posts both signup-passphrase and signup-confirm;
	// we compare them here against the same validation-style error as any
	// other invalid input.
	if !request.Passphrase.Equal(request.Confirm) {
		return problem(w, errors.New("passphrases do not match"))
	}

	// derive the encryption key for the user
	key := encrypt.DeriveKey(request.Passphrase, request.Salt)
	verifier := encrypt.MakeVerifier(key)

	// insert the newly created user into the database
	userID, uerr := h.store.Users().Create(ctx, request, verifier)
	if errors.Is(uerr, database.ErrUsernameTaken) {
		return problem(w, database.ErrUsernameTaken)
	} else if uerr != nil {
		return problem(w, uerr)
	}

	// set the encryption cookie
	http.SetCookie(w, h.cookie(key))

	// set the session cookie
	http.SetCookie(w, h.sessions.Create(userID, cookieTTL))

	// send user to the homepage
	return htmx.SetRedirect(w, "/")
}

const cookieTTL = 10 * time.Minute

func (h *loginHTTP) cookie(key *conceal.Bytes) *http.Cookie {
	return &http.Cookie{
		Name:     "onlylogins-encryption",
		Value:    base64.StdEncoding.EncodeToString(key.Unveil()),
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(cookieTTL),
		SameSite: http.SameSiteStrictMode,
		Secure:   h.environment.NonLocal(),
	}
}

func (h *loginHTTP) audit(request *users.LoginRequest, success bool) {
	ctx, cancel := scope.TTL(3 * time.Second)
	defer cancel()
	_ = h.store.Audit().Record(ctx, request, success)
}

func (h *loginHTTP) bogus(request *users.LoginRequest) {
	ctx, cancel := scope.TTL(3 * time.Second)
	defer cancel()
	_ = h.store.Audit().RecordBogus(ctx, request)
}
