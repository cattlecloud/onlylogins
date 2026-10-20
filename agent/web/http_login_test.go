package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func makeLoginRequest(t *testing.T, values url.Values) *http.Request {
	t.Helper()

	r, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/login?do=create", nil)
	must.NoError(t, err)

	r.PostForm = values
	return r
}

// newSignupRouter builds a mux router with a /login endpoint backed by an
// in-memory store; consistent with the per-test setup used elsewhere.
func newSignupRouter(t *testing.T) (*mux.Router, database.Store) {
	t.Helper()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	store := mockdb.Open(t)

	// pre-load authorize table by inserting one user so unique-constraint
	// conditions can be exercised; not required at this point
	routes := mux.NewRouter()
	routes.Handle("/login", &loginHTTP{
		log:         ulog.New("test"),
		store:       store,
		sessions:    mocks.NewSessions(),
		environment: env,
		html:        ui.Load("login.html"),
	})

	return routes, store
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
		must.Nil(t, got.Salt)
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

func TestSignup_NewUser_Redirects(t *testing.T) {
	t.Parallel()

	routes, _ := newSignupRouter(t)

	r := makeLoginRequest(t, url.Values{
		"signup-email":      {"alice@example.com"},
		"signup-username":   {"alice"},
		"signup-passphrase": {"correct horse battery staple"},
	})
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, r)

	// htmx.SetRedirect writes StatusFound along with the Hx-Redirect header
	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/", w.Header().Get("Hx-Redirect"))
}

func TestSignup_DuplicateUsername_RendersNiceError(t *testing.T) {
	t.Parallel()

	routes, _ := newSignupRouter(t)

	// first signup succeeds and creates the row
	r1 := makeLoginRequest(t, url.Values{
		"signup-email":      {"alice@example.com"},
		"signup-username":   {"alice"},
		"signup-passphrase": {"correct horse battery staple"},
	})
	w1 := httptest.NewRecorder()
	routes.ServeHTTP(w1, r1)
	must.Eq(t, http.StatusFound, w1.Code)
	must.Eq(t, "/", w1.Header().Get("Hx-Redirect"))

	// second signup with the same username should fail with the friendly
	// ErrUsernameTaken message
	r2 := makeLoginRequest(t, url.Values{
		"signup-email":      {"someone-else@example.com"},
		"signup-username":   {"alice"},
		"signup-passphrase": {"different passphrase but long enough"},
	})
	w2 := httptest.NewRecorder()
	routes.ServeHTTP(w2, r2)

	must.Eq(t, http.StatusOK, w2.Code)
	body := w2.Body.String()
	must.StrContains(t, body, `<span id="error" hx-swap-oob="true">`)
	must.StrContains(t, body, database.ErrUsernameTaken.Error())
	// must not leak any underlying sqlite / litesql error text
	must.StrNotContains(t, body, "litesql")
}

func TestSignup_InvalidEmail_RendersValidationError(t *testing.T) {
	t.Parallel()

	routes, _ := newSignupRouter(t)

	r := makeLoginRequest(t, url.Values{
		"signup-email":      {"not-an-email"},
		"signup-username":   {"alice"},
		"signup-passphrase": {"correct horse battery staple"},
	})
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, r)

	must.Eq(t, http.StatusOK, w.Code)
	body := w.Body.String()
	must.StrContains(t, body, `<span id="error" hx-swap-oob="true">`)
	must.StrNotContains(t, body, database.ErrUsernameTaken.Error())
}
