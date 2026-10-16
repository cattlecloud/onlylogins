package web

import (
	"html/template"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/users"
)

type modUserContent struct {
	User *users.Profile
}

type modUserHTTP struct {
	log   *ulog.Log
	store database.Store
	html  *template.Template
}

func (h *modUserHTTP) Path() string {
	return "/moderate/users/{id}"
}

func (h *modUserHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *modUserHTTP) Consumer() consumer {
	return moderate
}

func (h *modUserHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *modUserHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	_ = ctx
	_ = r

	content := &modUserContent{}

	return h.html.ExecuteTemplate(w, "layout", content)
}
