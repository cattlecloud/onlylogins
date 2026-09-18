package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/agent/web/htmx"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/users"
)

type profileEditHTMX struct {
	log   *ulog.Log
	store database.Store
}

func (h *profileEditHTMX) Path() string {
	return "/internal/profile/edit"
}

func (h *profileEditHTMX) Methods() []string {
	return []string{http.MethodGet, http.MethodPut}
}

func (h *profileEditHTMX) Consumer() consumer {
	return component
}

func (h *profileEditHTMX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
			http.MethodPut: h.put,
		},
	)
}

func (h *profileEditHTMX) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if not logged in, abort request
	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return abort(h.log, w, ErrPageProblem)
	}

	// extract the key/value we are updating
	k, _ := h.keyValueFrom(r)

	// return the appropriate html for editing desired field
	switch k {
	case "editloc":
		return htmx.Templates(
			&htmx.Template{
				Filename: "profile_edit_location.html",
				Fields:   profile.Location,
			},
		).Render(w, session)
	case "editlink":
		return htmx.Templates(
			&htmx.Template{
				Filename: "profile_edit_link.html",
				Fields:   profile.Link,
			},
		).Render(w, session)
	case "editabout":
		return htmx.Templates(
			&htmx.Template{
				Filename: "profile_edit_about.html",
				Fields:   profile.About,
			},
		).Render(w, session)
	default:
		return errors.New("no such actionable value")
	}
}

func (h *profileEditHTMX) put(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if not logged in, abort request
	if !session.Active() {
		return abort(h.log, w, ErrMissingSession)
	}

	// extract the key/value we are updating
	k, v := h.keyValueFrom(r)

	// do the update
	switch k {
	case "showemail":
		return h.toggleShowEmail(ctx, w, session, v)
	case "editloc":
		return h.editLocation(ctx, w, session, v)
	case "editlink":
		return h.editLink(ctx, w, session, v)
	case "editabout":
		return h.editAbout(ctx, w, session, v)
	default:
		return errors.New("no such key is recognizable")
	}
}

func (h *profileEditHTMX) problem(session identity.UserSession[users.Identity], w http.ResponseWriter, message string) error {
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

func (h *profileEditHTMX) toggleShowEmail(ctx scope.C, w http.ResponseWriter, session identity.UserSession[users.Identity], value string) error {
	uid := session.Identity()
	show, _ := strconv.ParseBool(value)
	h.log.D.Fmt("set show email to %v for user:%v", show, uid)
	if err := h.store.Users().ProfileSetShowEmail(ctx, uid, show); err != nil {
		return fmt.Errorf("unable to set show email: %w", err)
	}
	return htmx.Templates(
		&htmx.Template{
			Filename: "profile_edit_showemail.html",
			Fields:   show,
		},
	).Render(w, session)
}

func (h *profileEditHTMX) editLocation(ctx scope.C, w http.ResponseWriter, session identity.UserSession[users.Identity], loc string) error {
	uid := session.Identity()

	if len(loc) > 32 {
		return h.problem(session, w, "location too long")
	}

	if loc != "" && len(loc) < 3 {
		return h.problem(session, w, "location too short")
	}

	h.log.D.Fmt("set location to %s for user:%v", loc, uid)

	if err := h.store.Users().ProfileSetLocation(ctx, uid, loc); err != nil {
		return fmt.Errorf("unable to set location: %w", err)
	}
	return htmx.SetRedirect(w, "/profile")
}

func (h *profileEditHTMX) editLink(ctx scope.C, w http.ResponseWriter, session identity.UserSession[users.Identity], link string) error {
	uid := session.Identity()

	if link != "" && !strings.HasPrefix(link, "https://") {
		return h.problem(session, w, "link must start with https://")
	}

	h.log.D.Fmt("set link to %s for user:%v", link, uid)

	if err := h.store.Users().ProfileSetLink(ctx, uid, link); err != nil {
		return fmt.Errorf("unable to set link: %w", err)
	}

	return htmx.SetRedirect(w, "/profile")
}

func (h *profileEditHTMX) editAbout(ctx scope.C, w http.ResponseWriter, session identity.UserSession[users.Identity], about string) error {
	uid := session.Identity()

	if about != "" && len(about) > 512 {
		return h.problem(session, w, "about is too long")
	}

	if err := h.store.Users().ProfileSetAbout(ctx, uid, about); err != nil {
		return fmt.Errorf("unable to set about: %w", err)
	}

	return htmx.SetRedirect(w, "/profile")
}

func (h *profileEditHTMX) keyValueFrom(r *http.Request) (string, string) {
	var key, value string
	_ = forms.Parse(r, forms.Schema{
		"key":   forms.String(&key),
		"value": forms.String(&value),
	})
	return key, value
}
