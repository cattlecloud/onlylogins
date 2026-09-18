package web

import (
	"net/http/httptest"
	"testing"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/ui"
	"github.com/gorilla/mux"
	"github.com/shoenig/test/must"
)

func TestModUsers_ServeHTTP(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)
	routes := mux.NewRouter()
	routes.Handle("/moderate/users", &modUsersHTTP{
		log:   ulog.New("test"),
		store: store,
		html:  ui.Mod("mod_users.html"),
	})

	request := requestGet(t, "/moderate/users")
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, request)

	body := w.Body.String()
	must.StrContains(t, body, `<div class="modpage users">`)
}
