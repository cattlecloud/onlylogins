// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/runtime"
	"github.com/shoenig/test/must"
)

func TestSitemap_Production(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "onlylogins.net",
		Environment: "production",
	})

	h := &sitemapAPI{
		log:         ulog.New("test"),
		environment: env,
	}

	w := httptest.NewRecorder()
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/sitemap.xml", nil)
	must.NoError(t, rerr)

	h.ServeHTTP(w, r)

	body := w.Body.String()
	must.StrContains(t, body, `<loc>https://onlylogins.net</loc>`)
}

func TestSitemap_Local(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "onlylogins.net",
		Environment: "local",
	})

	h := &sitemapAPI{
		log:         ulog.New("test"),
		environment: env,
	}

	w := httptest.NewRecorder()
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/sitemap.xml", nil)
	must.NoError(t, rerr)

	h.ServeHTTP(w, r)

	body := w.Body.String()
	must.StrContains(t, body, `<loc>http://localhost:3000</loc>`)
}
