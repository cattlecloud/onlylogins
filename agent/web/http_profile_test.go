package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/ui"
	"github.com/shoenig/test/must"
)

func TestProfile_ServeHTTP_GET_not_logged_in(t *testing.T) {
	t.Parallel()

	handler := &profileHTTP{
		log:   ulog.New("test"),
		store: mockdb.Open(t),
		html:  ui.Load("profile.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	request := requestGet(t, "/profile", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, request)

	must.Eq(t, http.StatusTemporaryRedirect, w.Code)
	must.Eq(t, "/login", w.Header().Get("Location"))
}

func TestProfile_ServeHTTP_GET_logged_in(t *testing.T) {
	t.Parallel()

	handler := &profileHTTP{
		log:   ulog.New("test"),
		store: mockdb.Open(t),
		html:  ui.Load("profile.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)
	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	request := requestGet(t, "/profile", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<div class="middle profile">`)
	must.StrContains(t, w.Body.String(), `_nobody`)
	must.StrContains(t, w.Body.String(), `<div class="value plan">free</div>`)
	must.StrContains(t, w.Body.String(), `0 / 256`)
}

func TestProfile_ServeHTTP_GET_logged_in_with_documents(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &profileHTTP{
		log:   ulog.New("test"),
		store: store,
		html:  ui.Load("profile.html"),
	}

	ctx, cancel := scope.TTL(testwait)
	t.Cleanup(cancel)

	// the sentinel user gets a default "miscellaneous" folder on creation
	_, derr := store.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
		User:   mocks.Nobody,
		Folder: "miscellaneous",
		Label:  "github",
	})
	must.NoError(t, derr)

	scache := mocks.NewSessions()
	ss := session(handler, scache)
	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	request := requestGet(t, "/profile", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `1 / 256`)
}
