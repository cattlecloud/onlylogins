package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"github.com/shoenig/test/must"
)

func TestUsernameEdit_get(t *testing.T) {
	t.Parallel()

	const userID = 2

	store := mockdb.Open(t)

	target := &usernameEditHTMX{
		log:   ulog.New("test"),
		store: store,
	}

	scache := mocks.NewSessions()
	ss := sessions(target, scache)
	cookie := ss.Sessions.Create(userID, 1*time.Minute)
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/profile/edit", nil)
	must.NoError(t, rerr)
	r.AddCookie(cookie)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<div class="username" id="edit-username">`)
}

func TestUsernameEdit_put_forbidden(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	target := &usernameEditHTMX{
		log:   ulog.New("test"),
		store: store,
	}

	scache := mocks.NewSessions()
	ss := sessions(target, scache)
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodPut, "/profile/edit?chosen=abc", nil)
	must.NoError(t, rerr)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.StrContains(t, w.Body.String(), "unable to process request")
}

func TestUsernameEdit_put_allow(t *testing.T) {
	t.Parallel()

	const userID = 2 // a test user
	store := mockdb.Open(t)

	target := &usernameEditHTMX{
		log:   ulog.New("test"),
		store: store,
	}

	scache := mocks.NewSessions()
	ss := sessions(target, scache)
	cookie := ss.Sessions.Create(userID, 1*time.Minute)
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodPut, "/profile/edit?chosen=abc", nil)
	must.NoError(t, rerr)
	r.AddCookie(cookie)
	_ = r.ParseForm()

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusFound, w.Code)
}
