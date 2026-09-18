package htmx

import (
	"net/http/httptest"
	"testing"

	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/test/must"
)

func TestTemplate_Render_forbidden(t *testing.T) {
	t.Parallel()

	sample := &templte[*mocks.Session]{
		// no template files
	}

	loggedOut := &mocks.Session{
		SetIdentity: 0,
		SetActive:   false,
	}

	w := httptest.NewRecorder()
	err := sample.Render(w, loggedOut)
	must.NoError(t, err)
}

func TestTemplate_Render_okay(t *testing.T) {
	t.Parallel()

	sample := &templte[*mocks.Session]{
		filenames: []*Template{
			{
				Filename: "profile_edit_username.html",
				Fields: &sitecontent.UserInfo{
					Profile: users.Profile{Username: "bob"},
				},
			},
		},
	}

	loggedIn := &mocks.Session{
		SetIdentity: 1,
		SetActive:   true,
	}

	w := httptest.NewRecorder()
	err := sample.Render(w, loggedIn)
	must.NoError(t, err)

	body := w.Body.String()
	must.StrContains(t, body, `value="bob"`)
}
