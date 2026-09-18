package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func requestGet(t *testing.T, path string) *http.Request {
	r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	must.NoError(t, err)
	return r
}

func TestAbout_ServeHTTP(t *testing.T) {
	t.Parallel()

	env := runtime.Setup(&runtime.Config{
		Domain:      "test.local",
		Environment: "local",
	})

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/about", &aboutHTTP{
		log:         ulog.New("test"),
		environment: env,
		store:       store,
		html:        ui.Load("about.html"),
	})

	request := requestGet(t, "/about")
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<div class="middle about">`)
}
