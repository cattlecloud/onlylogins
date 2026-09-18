package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/runtime"
	"github.com/shoenig/test/must"
)

func TestRobots_Production(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "example.org",
		Environment: "production",
	})

	h := &robotsAPI{
		log:         ulog.New("test"),
		environment: env,
	}

	w := httptest.NewRecorder()
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/robots.txt", nil)
	must.NoError(t, rerr)

	h.ServeHTTP(w, r)

	body := w.Body.String()
	must.StrContains(t, body, "Allow: /$")
	must.StrContains(t, body, "Disallow: /*")
}

func TestRobots_Local(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	h := &robotsAPI{
		log:         ulog.New("test"),
		environment: env,
	}

	w := httptest.NewRecorder()
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/robots.txt", nil)
	must.NoError(t, rerr)

	h.ServeHTTP(w, r)

	body := w.Body.String()
	must.StrContains(t, body, "Disallow: /")
}
