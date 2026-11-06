package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/ui"
	"github.com/shoenig/test/must"
)

func TestFolders_ServeHTTP_GET_not_logged_in(t *testing.T) {
	t.Parallel()

	handler := &foldersHTTP{
		log:   ulog.New("test"),
		store: mockdb.Open(t),
		html:  ui.Load("folders.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)

	request := requestGet(t, "/folders", nil)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, request)

	must.Eq(t, http.StatusTemporaryRedirect, w.Code)
}

func TestFolders_ServeHTTP_GET_logged_in(t *testing.T) {
	t.Parallel()

	handler := &foldersHTTP{
		log:   ulog.New("test"),
		store: mockdb.Open(t),
		html:  ui.Load("folders.html"),
	}

	scache := mocks.NewSessions()
	ss := session(handler, scache)
	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	request := requestGet(t, "/folders", cookie)
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, request)

	must.Eq(t, http.StatusOK, w.Code)
	must.StrContains(t, w.Body.String(), `<div class="middle folders">`)
}

func TestFolders_ServeHTTP_POST_success(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	handler := &foldersHTTP{
		log:   ulog.New("test"),
		store: store,
		html:  ui.Load("folders.html"),
	}

	ctx, cancel := scope.TTL(testwait)
	t.Cleanup(cancel)

	scache := mocks.NewSessions()
	ss := session(handler, scache)
	cookie := ss.Sessions.Create(mocks.Nobody, 1*time.Minute)

	request := requestPost(t, "/folders", cookie, url.Values{
		"label": {"work"},
		"icon":  {"0"},
	})
	w := httptest.NewRecorder()
	ss.ServeHTTP(w, request)

	must.Eq(t, http.StatusFound, w.Code)
	must.Eq(t, "/", w.Header().Get("HX-Redirect"))

	// verify the folder was actually persisted to the store
	list, err := store.Folders().List(ctx, mocks.Nobody)
	must.NoError(t, err)
	must.SliceLen(t, 2, list) // 1 default ("miscellaneous") + 1 new ("work")
	must.Eq(t, "miscellaneous", list[0].Label)
	must.Eq(t, "work", list[1].Label)
}
