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

func TestProfileEditHTMX_toggleShowEmail(t *testing.T) {
	t.Parallel()

	const userID = 4
	store := mockdb.Open(t)

	target := &profileEditHTMX{
		log:   ulog.New("test"),
		store: store,
	}

	scache := mocks.NewSessions()
	ss := sessions(target, scache)
	cookie := ss.Sessions.Create(userID, 1*time.Minute)
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodPut, "/profile/edit?key=showemail&value=true", nil)
	must.NoError(t, rerr)
	r.AddCookie(cookie)
	_ = r.ParseForm()

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, r)
	must.Eq(t, http.StatusOK, w.Code)
}
