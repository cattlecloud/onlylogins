package web

import (
	"html/template"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/users"
)

type modUsersContent struct {
	Profiles []*users.Profile
}

type modUsersHTTP struct {
	log   *ulog.Log
	store database.Store
	html  *template.Template
}

func (h *modUsersHTTP) Path() string {
	return "/moderate/users"
}

func (h *modUsersHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *modUsersHTTP) Consumer() consumer {
	return moderate
}

func (h *modUsersHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *modUsersHTTP) get(ctx scope.C, w http.ResponseWriter, _ *http.Request) error {
	_ = ctx

	content := &modUsersContent{
		// TODO
	}

	return h.html.ExecuteTemplate(w, "layout", content)
}
