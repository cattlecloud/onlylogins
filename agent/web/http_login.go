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
	var passphrase *conceal.Text
	var notifications, spam bool
	forms.MustParse(r, forms.Schema{
		"signup-email":      forms.String(&email),
		"signup-username":   forms.String(&username),
		"signup-passphrase": forms.Secret(&passphrase),
		"notifications":     forms.BoolOr(&notifications, false),
		"marketing":         forms.BoolOr(&spam, false),
	})
	return &users.CreateUserRequest{
		Email:         email,
		Username:      username,
		Intermediate:  conceal.UUIDv4(),
		Passphrase:    passphrase,
		Notifications: notifications,
		Spam:          spam,
	}
}

func (*loginHTTP) loginFields(r *http.Request) (string, *conceal.Text) {
	var username string
	var passphrase *conceal.Text
	forms.MustParse(r, forms.Schema{
		"username":   forms.String(&username),
		"passphrase": forms.Secret(&passphrase),
	})
	return username, passphrase
}

func (h *loginHTTP) login(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	fmt.Println("login to existing account")

	_ = ctx
	_ = w
	_, _ = h.loginFields(r)

	return nil
}

func (h *loginHTTP) signup(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	fmt.Println("create an account")

	request := h.signupFields(r)

	if err := request.Validate(); err != nil {
		// TODO: graceful set error on the page so user can correct
		// their input fields
		return err
	}

	secret := encrypt.Combine(request.Passphrase, request.Intermediate)
	expect, eerr := encrypt.Expect(request.Passphrase, request.Intermediate)
	if eerr != nil {
		// TODO: encryption is broken, now what?
		return eerr
	}

	// insert the newly created user into the database
	userID, cerr := h.store.Users().Create(ctx, request, expect)
	if cerr != nil {
		return cerr
	}

	// set the encryption cookie
	http.SetCookie(w, cookie(secret))

	// set the session cookie
	http.SetCookie(w, h.sessions.Create(userID, cookieTTL))

	// set to the homepage
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}

const cookieTTL = 10 * time.Minute

func cookie(secret *conceal.Bytes) *http.Cookie {
	return &http.Cookie{
		Name:     "onlylogins-encryption",
		Value:    base64.StdEncoding.EncodeToString(secret.Unveil()),
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(cookieTTL),
		SameSite: http.SameSiteStrictMode,
		Secure:   false, // todo
	}
}
