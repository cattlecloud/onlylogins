package htmx

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/pkg/users"
)

// A Component is a raw HTML string.
type Component struct {
	HTML   string
	Fields any
}

type component[S identity.UserSession[users.Identity]] struct {
	components []*Component
}

var funcs = template.FuncMap{
	// no funcs yet
}

func (c *component[S]) Render(w http.ResponseWriter, session identity.UserSession[users.Identity]) error {
	if !session.Active() {
		w.WriteHeader(http.StatusForbidden)
		return nil
	}

	// render one or more components into the buffer, which we then dump into
	// the response writer when ready
	//
	// htmx automagically extracts each HTML element
	buff := new(bytes.Buffer)

	for i, f := range c.components {
		name := fmt.Sprintf("htmx[%d]", i)
		t, terr := template.New(name).Funcs(funcs).Parse(f.HTML)
		if terr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			panic(fmt.Sprintf("htmx: unable to render component: %v", terr))
		}
		xerr := t.Execute(buff, f.Fields)
		if xerr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			panic(fmt.Sprintf("htmx: unable to execute component: %v", xerr))
		}
	}

	// actually write the entire payload
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, buff.String())

	return nil
}
