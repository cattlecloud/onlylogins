// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"errors"
	"html/template"
	"net/http"
	"time"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/agent/web/pages"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/sessions/cookies"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

type passphraseHTTP struct {
	log     *ulog.Log
	store   database.Store
	session middles.Sessions[users.Identity]
	html    *template.Template
}

func (h *passphraseHTTP) Path() string {
	return "/profile/passphrase"
}

func (h *passphraseHTTP) Methods() []string {
	return []string{http.MethodGet, http.MethodPost}
}

func (h *passphraseHTTP) Metric() string {
	return "change_passphrase"
}

func (h *passphraseHTTP) Consumer() consumer {
	return normal
}

func (h *passphraseHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:  h.get,
			http.MethodPost: h.post,
		},
	)
}

func (h *passphraseHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if user is not logged in, redirect to login page
	if !session.Active() {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil
	}

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return perr
	}

	head := &pages.Head{
		Title:       format.PageTitle("Change Passphrase"),
		Styles:      []string{"passphrase.css"},
		Description: h.description(),
		Profile:     profile,
	}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, struct{}{}).Execute(w, session)
}

func (h *passphraseHTTP) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if user is not logged in, redirect to login page
	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	request := h.requestFrom(r)

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if err := h.store.Users().ChangePassphrase(ctx, session.Identity(), request); err != nil {
		if errors.Is(err, database.ErrIncorrectCredentials) {
			return problem(w, database.ErrIncorrectCredentials)
		}
		return abort(h.log, w, err)
	}

	// the passphrase changed; end this session and clear its cookies so the
	// user must log in again with the new passphrase
	h.expire(w, session)

	// render the success response in place of the form
	return htmx.Write(w, &htmx.Component{HTML: `
<section id="passphrase" class="passphrase success">
  <h1>passphrase has been updated</h1>
  <p><a href="/login">please login</a></p>
</section>`})
}

func (h *passphraseHTTP) requestFrom(r *http.Request) *users.ChangePassphraseRequest {
	var previous, passphrase, confirm *conceal.Text
	forms.MustParse(r, forms.Schema{
		"previous":   forms.Secret(&previous),
		"passphrase": forms.Secret(&passphrase),
		"confirm":    forms.Secret(&confirm),
	})

	return &users.ChangePassphraseRequest{
		Previous:   previous,
		Passphrase: passphrase,
		Confirm:    confirm,
	}
}

func (h *passphraseHTTP) expire(w http.ResponseWriter, session identity.UserSession[users.Identity]) {
	// delete the session token server-side
	if err := h.session.Delete(session.Token()); err != nil {
		h.log.E.Fmt("unable to delete session: %v", err)
	}

	// delete the user encryption cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cookies.Encryption,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	// delete the user session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cookies.Session,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
}

func (h *passphraseHTTP) description() string {
	const text = `
	Change the passphrase used to encrypt your data.`
	return format.PageDescription(text)
}
