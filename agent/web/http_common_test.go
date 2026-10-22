package web

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/shoenig/test/must"
)

func requestGet(t *testing.T, path string, cookie *http.Cookie) *http.Request {
	r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	must.NoError(t, err)

	if cookie != nil {
		r.AddCookie(cookie)
	}
	return r
}

func requestPost(t *testing.T, path string, cookie *http.Cookie, values url.Values) *http.Request {
	r, err := http.NewRequestWithContext(t.Context(), http.MethodPost, path, nil)
	must.NoError(t, err)

	if cookie != nil {
		r.AddCookie(cookie)
	}

	r.PostForm = values
	return r
}
