package web

import (
	"fmt"
	"html/template"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/runtime"
)

type loginHTTP struct {
	log         *ulog.Log
	environment runtime.Environment
	html        *template.Template
}

func (h *loginHTTP) Path() string {
	return "/login"
}

func (h *loginHTTP) Methods() []string {
	return []string{http.MethodGet}
}

func (h *loginHTTP) Metric() string {
	return "login"
}

func (h *loginHTTP) Consumer() consumer {
	return normal
}

func (h *loginHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *loginHTTP) get(_ scope.C, w http.ResponseWriter, _ *http.Request) error {
	type page struct {
		Canonical template.URL
	}

	content := &page{
		Canonical: h.environment.Canonical("%s", h.Path()),
	}

	// render the login page
	if err := h.html.Execute(w, content); err != nil {
		return fmt.Errorf("unable to render login page: %w", err)
	}

	return nil
}
