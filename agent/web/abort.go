package web

import (
	"errors"
	"io"
	"net/http"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools"
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
