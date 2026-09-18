package web

import (
	"html/template"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/agent/web/htmx"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/users"
)

type privacyHTTP struct {
	log         *ulog.Log
	environment runtime.Environment
	store       database.Store
	html        *template.Template
}

func (h *privacyHTTP) Path() string {
	return "/privacy"
}

func (h *privacyHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *privacyHTTP) Metric() string {
	return "privacy"
}

func (h *privacyHTTP) Consumer() consumer {
	return normal
}

func (h *privacyHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *privacyHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	profile, perr := getUserInfo(ctx, h.store.Users(), session)
	if perr != nil {
		return perr
	}

	head := &htmx.Head{
		Title:       "Privacy | OnlyLogins",
		Styles:      []string{"basics.css"},
		Canonical:   h.environment.Canonical("%s", h.Path()),
		Description: h.description(),
		Profile:     profile,
	}

	middle := struct{}{}

	return htmx.Page[identity.UserSession[users.Identity]](h.html, head, middle).Render(w, session)
}

func (h *privacyHTTP) description() string {
	const text = `
	Store login information such as usernames, passwords, backup codes, etc.`
	return format.PageDescription(text)
}
