package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/test/must"
)

func TestMids_moderationGate_allow(t *testing.T) {
	t.Parallel()

	const adminID = 999

	reachable := false
	target := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		reachable = true
	})

	council := &mocks.Council{Admins: []users.Identity{adminID}}

	gate := &moderationGate{
		log:     ulog.New("test"),
		next:    target,
		council: council,
	}

	scache := mocks.NewSessions()
	ss := sessions(gate, scache)
	cookie := ss.Sessions.Create(adminID, 1*time.Minute)
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/moderate", nil)
	must.NoError(t, rerr)
	r.AddCookie(cookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusOK, w.Code)
	must.True(t, reachable)
}

func TestMids_moderationGate_logged_out(t *testing.T) {
	t.Parallel()

	const adminID = 999

	reachable := false
	target := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		reachable = true
	})

	council := &mocks.Council{Admins: []users.Identity{adminID}}

	gate := &moderationGate{
		log:     ulog.New("test"),
		next:    target,
		council: council,
	}

	scache := mocks.NewSessions()
	ss := sessions(gate, scache)
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/moderate", nil)
	must.NoError(t, rerr)
	// no session cookie; we are not logged in

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusNotFound, w.Code)
	must.False(t, reachable)
}

func TestMids_moderationGate_not_admin(t *testing.T) {
	t.Parallel()

	const adminID = 999
	const userID = 312

	reachable := false
	target := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		reachable = true
	})

	council := &mocks.Council{Admins: []users.Identity{adminID}}

	gate := &moderationGate{
		log:     ulog.New("test"),
		next:    target,
		council: council,
	}

	scache := mocks.NewSessions()
	ss := sessions(gate, scache)
	cookie := ss.Sessions.Create(userID, 1*time.Minute) // regular user
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/moderate", nil)
	must.NoError(t, rerr)
	r.AddCookie(cookie) // session is not an admin user

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusNotFound, w.Code)
	must.False(t, reachable)
}

func TestMids_mogrifyGate_allow(t *testing.T) {
	t.Parallel()

	const testID = 888

	reachable := false
	target := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		reachable = true
	})

	council := &mocks.Council{Transmogs: []users.Identity{testID}}

	gate := &mogrifyGate{
		log:     ulog.New("test"),
		next:    target,
		council: council,
	}

	scache := mocks.NewSessions()
	ss := sessions(gate, scache)
	cookie := ss.Sessions.Create(testID, 1*time.Minute) // testing user
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/moderate", nil)
	must.NoError(t, rerr)
	r.AddCookie(cookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusOK, w.Code)
	must.True(t, reachable)
}

func TestMids_mogrifyGate_logged_out(t *testing.T) {
	t.Parallel()

	const testID = 888

	reachable := false
	target := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		reachable = true
	})

	council := &mocks.Council{Transmogs: []users.Identity{testID}}

	gate := &mogrifyGate{
		log:     ulog.New("test"),
		next:    target,
		council: council,
	}

	scache := mocks.NewSessions()
	ss := sessions(gate, scache)
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/moderate", nil)
	must.NoError(t, rerr)
	// no session cookie; we are logged out

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusNotFound, w.Code)
	must.False(t, reachable)
}

func TestMids_mogrifyGate_not_capable(t *testing.T) {
	t.Parallel()

	const testID = 888
	const userID = 312

	reachable := false
	target := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		reachable = true
	})

	council := &mocks.Council{Transmogs: []users.Identity{testID}}

	gate := &mogrifyGate{
		log:     ulog.New("test"),
		next:    target,
		council: council,
	}

	scache := mocks.NewSessions()
	ss := sessions(gate, scache)
	cookie := ss.Sessions.Create(userID, 1*time.Minute) // just a regular user
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/moderate", nil)
	must.NoError(t, rerr)
	r.AddCookie(cookie) // must block regular users

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusNotFound, w.Code)
	must.False(t, reachable)
}
