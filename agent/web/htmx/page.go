package htmx

import (
	"fmt"
	"html/template"
	"net/http"

	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/lang"
)

type Head struct {
	Title       string
	NoIndex     bool
	Styles      []string
	Canonical   template.URL
	Description string
	Profile     *users.Profile
}

func (h *Head) Robots() string {
	return lang.Maybe(h.NoIndex, "noindex", "all")
}

type page[S identity.UserSession[users.Identity], M any] struct {
	html   *template.Template
	head   *Head
	middle M
}

func (p *page[S, M]) Render(w http.ResponseWriter, session S) error {
	// we need to be able to reference the session from M; so we create a
	// union type to be fed downstream to templates

	// combine head with session
	type union[T any] struct {
		Session S
		Section T
	}

	// a type of the union types
	type total struct {
		Head   union[*Head]
		Middle union[M]
	}

	// assemble the parts together
	content := &total{
		Head:   union[*Head]{Session: session, Section: p.head},
		Middle: union[M]{Session: session, Section: p.middle},
	}

	// finally render the page with our compilation of data
	if err := p.html.ExecuteTemplate(w, "layout", content); err != nil {
		// if we get here there is a template logic bug that must be fixed;
		// expectationf of working are out the window
		panic(fmt.Sprintf("htmx: unable to render template: %v", err))
	}

	// the page has been rendered
	return nil
}
