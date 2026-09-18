package htmx

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/ui"
)

type Template struct {
	Filename string
	Fields   any
}

type templte[S identity.UserSession[users.Identity]] struct {
	filenames []*Template
}

func (t *templte[S]) Render(w http.ResponseWriter, session identity.UserSession[users.Identity]) error {
	if !session.Active() {
		w.WriteHeader(http.StatusForbidden)
		return nil
	}

	// render one or more template files into the buffer, which we then dump
	// into the response writer at the end
	//
	// htmx automagically extracts each HTML element
	buff := new(bytes.Buffer)

	funcs := template.FuncMap{
		// no funcs yet
	}

	for _, f := range t.filenames {
		files, ferr := fs.Sub(ui.HTMX, "parts")
		if ferr != nil {
			panic(fmt.Sprintf("htmx: unable to load html part files: %v", ferr))
		}

		root := template.New("root").Funcs(funcs)
		tree, terr := root.ParseFS(files, f.Filename)
		if terr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			panic(fmt.Sprintf("htmx: unable to render file: %v", terr))
		}

		x := tree.Lookup(strings.TrimSuffix(f.Filename, ".html"))

		xerr := x.Execute(buff, f.Fields)
		if xerr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			panic(fmt.Sprintf("htmx: unable to execute file: %v", xerr))
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, buff.String())
	return nil
}
