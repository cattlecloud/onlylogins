package htmx

import (
	"html/template"
	"net/http"

	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/pkg/users"
)

// RenderInterface is implemented by every mode of HTMX implementation.
//
// We use htmx to render full pages, individual templates, small html
// snippets, and bits of plain text.
type RenderInterface[S identity.UserSession[users.Identity]] interface {
	Render(http.ResponseWriter, S) error
}

// Page is used to render a full HTML page with a head, middle component, by
// making use of the given HTML template file.
func Page[S identity.UserSession[users.Identity], M any](html *template.Template, head *Head, middle M) RenderInterface[S] {
	return &page[S, M]{
		html:   html,
		head:   head,
		middle: middle,
	}
}

// Components is used to render one or more HTML snippets.
func Components(c ...*Component) RenderInterface[identity.UserSession[users.Identity]] {
	return &component[identity.UserSession[users.Identity]]{
		components: c,
	}
}

// Templates is used to render one or more HTML files.
//
// The template file(s) must be included in the ui.HTMX embedded filesystem.
// (i.e. they must be in the parts/ directory)
func Templates(t ...*Template) RenderInterface[identity.UserSession[users.Identity]] {
	return &templte[identity.UserSession[users.Identity]]{
		filenames: t,
	}
}

// Text is used to render plain text.
func Text[T ~string](content T) RenderInterface[identity.UserSession[users.Identity]] {
	return &text[T]{
		content: content,
	}
}
