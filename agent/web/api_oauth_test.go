package web

import (
	"net/http"
	"testing"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/webtools"
	"github.com/shoenig/test/must"
)

func TestOauth_idpFrom(t *testing.T) {
	t.Parallel()

	ctx := scope.New()
	h := (*oauthAPI)(nil)

	makeRequest := func(params map[string]string) *http.Request {
		u := webtools.CreateURL("localhost", "/path", params).String()
		r, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
		must.NoError(t, err)
		_ = r.ParseForm()
		return r
	}

	t.Run("ok", func(t *testing.T) {
		r := makeRequest(map[string]string{"idp": "google"})
		idp := h.idpFrom(r)
		must.Eq(t, idp, "google")
	})

	t.Run("other", func(t *testing.T) {
		r := makeRequest(map[string]string{
			"data": "abc123",
			"idp":  "other",
			"foo":  "bar",
		})
		idp := h.idpFrom(r)
		must.Eq(t, idp, "other")
	})

	t.Run("missing", func(t *testing.T) {
		r := makeRequest(nil)
		idp := h.idpFrom(r)
		must.Eq(t, idp, "")
	})
}
