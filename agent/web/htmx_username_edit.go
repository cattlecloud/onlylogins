package web

import (
	"net/http"
	"strings"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/onlylogins/agent/web/htmx"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/users"
)

type usernameEditHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *usernameEditHTMX) Path() string {
	return "/internal/username/edit"
}

func (h *usernameEditHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPut}
}

func (h *usernameEditHTMX) Consumer() consumer {
	return component
}

func (h *usernameEditHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
			http.MethodPut: h.put,
		},
	)
}

func (h *usernameEditHTMX) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if not logged in, abort request
	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	// get the info of the user
	info, ierr := getUserInfo(ctx, h.store.Users(), session)
	if ierr != nil {
		return abort(h.log, w, ErrPageProblem)
	}

	return htmx.Templates(
		&htmx.Template{
			Filename: "profile_edit_username.html",
			Fields:   info,
		},
	).Render(w, session)
}

func (h *usernameEditHTMX) put(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if not logged in, abort request
	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	// prepare handler for setting an error message
	problem := func(message string) error {
		// leave the form in place
		_ = htmx.SetNoSwap(w)

		// set the active error message
		return htmx.Components(
			&htmx.Component{
				HTML:   `<span id="error" hx-swap-oob="true">{{.}}</span>`,
				Fields: message,
			},
		).Render(w, session)
	}

	// extract the chosen username from request
	chosen := h.chosenFrom(r)
	if !users.AllowUsername(chosen) {
		return problem("username is forbidden")
	}

	// set the username in the database
	u := h.store.Users()
	if err := u.SetUsername(ctx, session.Identity(), chosen); err != nil {
		return problem(err.Error())
	}

	// TODO: want to log latest name change
	// query for latest name change for this user; print it here
	// do this as a list, which we can reuse later for auditing a profile

	// just reload profile page
	return htmx.SetRedirect(w, "/profile")
}

func (h *usernameEditHTMX) chosenFrom(r *http.Request) string {
	var chosen string
	_ = forms.Parse(r, forms.Schema{
		"chosen": forms.String(&chosen),
	})
	return strings.TrimSpace(chosen)
}
