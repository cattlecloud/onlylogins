package web

import (
	"html/template"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/urlpath"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type modUserContent struct {
	User        *sitecontent.UserInfo
	NameChanges []*sitecontent.NameChange
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
	id := h.idFrom(r)
	u := h.store.Users()

	info, ierr := u.Info(ctx, id)
	if ierr != nil {
		return abort(h.log, w, ierr)
	}

	changes, cerr := u.ListUsernameChanges(ctx, id)
	if cerr != nil {
		return abort(h.log, w, cerr)
	}

	content := &modUserContent{
		User:        info,
		NameChanges: changes,
	}

	return h.html.ExecuteTemplate(w, "layout", content)
}

func (h *modUserHTTP) idFrom(request *http.Request) users.Identity {
	var id users.Identity
	_ = urlpath.Parse(request, urlpath.Schema{
		"id": urlpath.Int(&id),
	})
	return id
}
