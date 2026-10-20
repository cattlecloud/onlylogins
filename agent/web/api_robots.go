package web

import (
	"io"
	"net/http"
	"strings"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools"
	"cattlecloud.net/go/webtools/runtime"
)

type robotsAPI struct {
	log         *ulog.Log
	environment runtime.Environment
}

func (h *robotsAPI) Path() string {
	return "/robots.txt"
}

func (h *robotsAPI) Methods() []string {
	return []string{http.MethodGet}
}

func (h *robotsAPI) Consumer() consumer {
	return robotic
}

func (h *robotsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	webtools.SetContentType(w, webtools.ContentTypeText)
	w.WriteHeader(http.StatusOK)

	switch h.environment.Get() {
	case runtime.Production:
		_, _ = io.WriteString(w, robotsTXT)
	default:
		_, _ = io.WriteString(w, developmentTXT)
	}

	// track who requested robots.txt file
	ua := webtools.Origins(r)
	h.log.I.Fmt("request by [%s] from (%s)", ua.String(), ua.From())
}

var developmentTXT = strings.TrimSpace(`
# Created by CattleCloud LLC

User-Agent: *
Disallow: /
`)

var robotsTXT = strings.TrimSpace(`
# Created by CattleCloud LLC

User-Agent: *
Crawl-delay: 10
Allow: /about
Allow: /privacy
Allow: /terms
Allow: /help
Allow: /$
Disallow: /*

Sitemap: https://onlylogins.net/sitemap.xml
`)
