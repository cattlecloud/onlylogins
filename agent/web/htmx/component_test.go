package htmx

import (
	"net/http/httptest"
	"testing"

	"cattlecloud.net/onlylogins/hack/mocks"
	"github.com/shoenig/test/must"
)

func TestComponent_Render_forbidden(t *testing.T) {
	t.Parallel()

	sample := &component[*mocks.Session]{
		// no components
	}

	loggedOut := &mocks.Session{
		SetIdentity: 0,
		SetActive:   false,
	}

	w := httptest.NewRecorder()
	err := sample.Render(w, loggedOut)
	must.NoError(t, err)
}

func TestComponent_Render_okay(t *testing.T) {
	t.Parallel()

	sample := &component[*mocks.Session]{
		components: []*Component{
			{
				HTML: `<p>Hello, {{.Name}}!</p>`,
				Fields: struct {
					Name string
				}{Name: "Carol"},
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
	must.StrContains(t, body, "Hello, Carol!")
}
