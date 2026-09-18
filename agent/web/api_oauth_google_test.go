package web

import (
	"net/http"
	"testing"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/webtools"
	"github.com/shoenig/test/must"
)

func TestGoogleAPI_nonceFrom(t *testing.T) {
	t.Parallel()

	h := ((*googleAPI)(nil))

	t.Run("ok", func(t *testing.T) {
		result := h.nonceFrom("nonce=ee58aaf0-8c78-4a28-9361-48779cf22d67")
		must.UUIDv4(t, result.Unveil())
	})

	t.Run("malformed", func(t *testing.T) {
		result := h.nonceFrom("nonce=abc123")
		must.Eq(t, "", result.Unveil())
	})
}

func TestGoogleAPI_authCodeFrom(t *testing.T) {
	t.Parallel()

	ctx := scope.New()
	h := (*googleAPI)(nil)

	makeRequest := func(params map[string]string) *http.Request {
		u := webtools.CreateURL("localhost", "/path", params).String()
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		must.NoError(t, err)
		must.NoError(t, r.ParseForm())
		return r
	}

	t.Run("ok", func(t *testing.T) {
		request := makeRequest(map[string]string{
			"code":  "code1",
			"state": "state1",
		})
		code, state, err := h.authCodeFrom(request)
		must.NoError(t, err)
		must.Eq(t, "code1", code)
		must.Eq(t, "state1", state)
	})

	t.Run("missing code", func(t *testing.T) {
		request := makeRequest(map[string]string{
			"state": "state1",
		})
		_, _, err := h.authCodeFrom(request)
		must.ErrorIs(t, err, forms.ErrNoValue)
	})

	t.Run("missing state", func(t *testing.T) {
		request := makeRequest(map[string]string{
			"code": "code1",
		})
		_, _, err := h.authCodeFrom(request)
		must.ErrorIs(t, err, forms.ErrNoValue)
	})

	t.Run("error is set", func(t *testing.T) {
		request := makeRequest(map[string]string{
			"code":  "code1",
			"state": "state1",
			"error": "problem1",
		})
		code, state, err := h.authCodeFrom(request)
		must.ErrorContains(t, err, "problem1")
		must.Eq(t, "", code)
		must.Eq(t, "", state)
	})
}
