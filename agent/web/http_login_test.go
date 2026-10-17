package web

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/shoenig/test/must"
)

func makeLoginRequest(t *testing.T, values url.Values) *http.Request {
	t.Helper()

	r, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/login", nil)
	must.NoError(t, err)

	r.PostForm = values
	return r
}

func TestSignupFields(t *testing.T) {
	t.Parallel()

	t.Run("all fields present", func(t *testing.T) {
		request := makeLoginRequest(t, url.Values{
			"signup-email":      {"you@example.com"},
			"signup-username":   {"alice"},
			"signup-passphrase": {"correct horse battery staple"},
			"notifications":     {"true"},
			"marketing":         {"true"},
		})

		got := (&loginHTTP{}).signupFields(request)

		must.Eq(t, "you@example.com", got.Email)
		must.Eq(t, "alice", got.Username)
		must.Eq(t, "correct horse battery staple", got.Passphrase.Unveil())
		must.True(t, got.Notifications)
		must.True(t, got.Spam)
		must.NotNil(t, got.Intermediate)
	})

	t.Run("checkboxes unchecked", func(t *testing.T) {
		request := makeLoginRequest(t, url.Values{
			"signup-email":      {"you@example.com"},
			"signup-username":   {"alice"},
			"signup-passphrase": {"correct horse battery staple"},
		})

		got := (&loginHTTP{}).signupFields(request)

		must.False(t, got.Notifications)
		must.False(t, got.Spam)
	})

	t.Run("missing required field", func(t *testing.T) {
		request := makeLoginRequest(t, url.Values{
			"signup-username":   {"alice"},
			"signup-passphrase": {"correct horse battery staple"},
		})

		must.Panic(t, func() {
			(&loginHTTP{}).signupFields(request)
		})
	})

	t.Run("checkbox on", func(t *testing.T) {
		request := makeLoginRequest(t, url.Values{
			"signup-email":      {"you@example.com"},
			"signup-username":   {"alice"},
			"signup-passphrase": {"correct horse battery staple"},
			"notifications":     {"on"},
			"marketing":         {"on"},
		})

		got := (&loginHTTP{}).signupFields(request)

		must.True(t, got.Notifications)
		must.True(t, got.Spam)
	})

	t.Run("extra fields ignored", func(t *testing.T) {
		request := makeLoginRequest(t, url.Values{
			"signup-email":      {"you@example.com"},
			"signup-username":   {"alice"},
			"signup-passphrase": {"correct horse battery staple"},
			"unique":            {"true"},
			"no-recovery":       {"true"},
		})

		got := (&loginHTTP{}).signupFields(request)

		must.Eq(t, "you@example.com", got.Email)
		must.Eq(t, "alice", got.Username)
		must.Eq(t, "correct horse battery staple", got.Passphrase.Unveil())
	})
}
