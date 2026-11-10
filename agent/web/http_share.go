// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"html/template"
	"net/http"
	"time"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/agent/web/pages"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

type shareHTTP struct {
	log         *ulog.Log
	environment runtime.Environment
	store       database.Store
	vault       encrypt.Vault
	html        *template.Template
}

func (h *shareHTTP) Path() string {
	return "/share"
}

func (h *shareHTTP) Methods() []string {
	return []string{http.MethodGet, http.MethodPut}
}

func (h *shareHTTP) Metric() string {
	return "share"
}

func (h *shareHTTP) Consumer() consumer {
	return normal
}

func (h *shareHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
			http.MethodPut: h.put,
		},
	)
}

func (h *shareHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return perr
	}

	head := &pages.Head{
		Title:       format.PageTitle("Share"),
		Styles:      []string{"share.css"},
		Canonical:   h.environment.Canonical("%s", h.Path()),
		Description: h.description(),
		Profile:     profile,
	}

	middle := struct{}{}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, middle).Execute(w, session)
}

// put handles the creation of a new share.
func (h *shareHTTP) put(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// redirect to login page if not logged in
	if !session.Active() {
		return htmx.SetRedirect(w, "/login")
	}

	content, ttl := h.getCreation(r)

	request := &sitecontent.CreateShareRequest{
		User:   session.Identity(),
		Secret: content,
		TTL:    ttl,
		Now:    time.Now(),
	}

	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	uid, cerr := h.store.Shares(h.vault).Create(ctx, request)
	if cerr != nil {
		return abort(h.log, w, cerr)
	}

	return htmx.Write(w,
		&htmx.Component{
			HTML: `
		{{$address := printf "%s/share/open/%s" .Host .UID}}
		<article>
		<label>
		  <span>share link</span>
		  <button type="button" class="copy-btn" onclick="copylink(this, 2000, '{{$address}}')">Copy</button>
		</label>
		<input id="uuid" type="text" value="{{$address}}" disabled>
		</article>`,
			Fields: struct {
				UID  records.SharesUnique
				Host template.URL
			}{
				UID:  uid,
				Host: h.environment.Canonical("/"),
			},
		},
	)
}

func (h *shareHTTP) getCreation(r *http.Request) (*conceal.Text, time.Duration) {
	var content *conceal.Text
	var ttl int
	forms.MustParse(r, forms.Schema{
		"content": forms.Secret(&content),
		"ttl":     forms.Int(&ttl),
	})
	return content, getTTL(ttl)
}

// getTTL ensures we accept only ttl values from the pre-set list of options,
// defaulting to 10 minutes if some improper value is supplied.
func getTTL(seconds int) time.Duration {
	switch seconds {
	case 900:
	case 3600:
	case 86400:
	case 432000:
	case 2592000:
	default:
		seconds = 900
	}
	return time.Duration(seconds) * time.Second
}

func (h *shareHTTP) description() string {
	const text = `
	Create encrypted, self-destructing secret messages with Share on OnlyLogins.
	Set expiration timers and share sensitive information securely with a one-time link.`
	return format.PageDescription(text)
}
