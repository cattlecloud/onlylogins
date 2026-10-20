package web

import (
	"errors"
	"io"
	"net/http"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools"
	"cattlecloud.net/go/webtools/htmx"
)

var (
	ErrPageProblem    = errors.New("problem loading page")
	ErrMissingSession = errors.New("no such session")
)

func abort(log *ulog.Log, w http.ResponseWriter, err error) error {
	log.E.Fmt("aborting request: %v", err)
	webtools.SetContentType(w, webtools.ContentTypeText)
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = io.WriteString(w, "unable to process request")
	return errors.Join(ErrPageProblem, err)
}

func deny(log *ulog.Log, w http.ResponseWriter, err error) error {
	log.W.Fmt("blocking request: %v", err)
	webtools.SetContentType(w, webtools.ContentTypeText)
	w.WriteHeader(http.StatusForbidden)
	_, _ = io.WriteString(w, "potentially malicious request denied")
	return nil
}

func lost(log *ulog.Log, w http.ResponseWriter, r *http.Request) error {
	log.W.Fmt("request of page that does not exist %q", r.URL.Path)
	webtools.SetContentType(w, webtools.ContentTypeText)
	w.WriteHeader(http.StatusNotFound)
	_, _ = io.WriteString(w, "page not found")
	return nil
}

// problem sets the <id:error> element with the content of err.
//
// it also explicitly sets no-swap, so that the htmx component originating
// the request is left alone; i.e. retaining the problematic input
func problem(w http.ResponseWriter, err error) error {
	// leave the form content in place
	_ = htmx.SetNoSwap(w)

	// set the active error message; out of bounds update
	return htmx.Write(w,
		&htmx.Component{
			HTML:   `<span id="error" hx-swap-oob="true">{{.}}</span>`,
			Fields: err.Error(),
		},
	)
}
