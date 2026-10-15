package htmx

import (
	"net/http/httptest"
	"testing"

	"cattlecloud.net/onlylogins/hack/mocks"
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

	t.Skip("FIXME")

	// sample := &templte[*mocks.Session]{
	// 	filenames: []*Template{
	// 		{
	// 			// nothing?
	// 		},
	// 	},
	// }

	// loggedIn := &mocks.Session{
	// 	SetIdentity: 1,
	// 	SetActive:   true,
	// }

	// w := httptest.NewRecorder()
	// err := sample.Render(w, loggedIn)
	// must.NoError(t, err)

	// body := w.Body.String()
	// must.StrContains(t, body, `value="bob"`)
}
