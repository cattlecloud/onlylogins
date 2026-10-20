package pages

import (
	"fmt"
	"html/template"
	"net/http"

	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/lang"
)

// New is used to render a full HTML page with a head, middle component,
// and right component, by executing the given HTML template file with a
// "layout" template as the root.
func New[S identity.UserSession[users.Identity], M any](
	html *template.Template,
	head *Head,
	middle M,
	right *Right,
) *Page[S, M] {
	return &Page[S, M]{
		html:   html,
		head:   head,
		middle: middle,
		right:  right,
	}
}

type Head struct {
	Title       string
	NoIndex     bool
	Styles      []string
	Canonical   template.URL
	Description string
	Profile     *users.Profile
	Branch      string
}

func (h *Head) Robots() string {
	return lang.Maybe(h.NoIndex, "noindex", "all")
}

type Page[S identity.UserSession[users.Identity], M any] struct {
	html   *template.Template
	head   *Head
	middle M
	right  *Right
}

func (p *Page[S, M]) Execute(w http.ResponseWriter, session S) error {
	// we need to be able to reference the session from M; so we create a
	// union type to be fed downstream to templates

	// combine component with session
	type union[T any] struct {
		Session S
		Section T
	}

	// a union of the union types
	type total struct {
		Head   union[*Head]
		Middle union[M]
		Right  union[*Right]
	}

	// assemble the parts together
	content := &total{
		Head:   union[*Head]{Session: session, Section: p.head},
		Middle: union[M]{Session: session, Section: p.middle},
		Right:  union[*Right]{Session: session, Section: p.right},
	}

	// finally render the page with our compilation of data
	if err := p.html.ExecuteTemplate(w, "layout", content); err != nil {
		// if we get here there is a template logic bug that must be fixed;
		// expectations of working are out the window
		panic(fmt.Sprintf("htmx: unable to render template: %v", err))
	}

	// the page has been rendered
	return nil
}
