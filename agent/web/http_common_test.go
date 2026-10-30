package web

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/shoenig/test/must"
)

// testwait is the timeout used for test cases
//
// many tests create / verify users which burns a lot of CPU due to argon
// cryptographic operations; we need to allow for small ci runners
const testwait = 1 * time.Minute

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

func requestDelete(t *testing.T, path string, cookie *http.Cookie) *http.Request {
	r, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, path, nil)
	must.NoError(t, err)

	if cookie != nil {
		r.AddCookie(cookie)
	}
	return r
}
