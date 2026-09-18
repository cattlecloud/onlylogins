package htmx

import (
	"net/http/httptest"
	"testing"

	"github.com/shoenig/test/must"
)

func TestText_Render(t *testing.T) {
	t.Parallel()

	sample := &text[string]{
		content: "hello!",
	}

	w := httptest.NewRecorder()
	err := sample.Render(w, nil)
	must.NoError(t, err)

	contentType := w.Header().Get("Content-Type")
	must.Eq(t, "text/plain; charset=utf8", contentType)
	body := w.Body.String()
	must.Eq(t, "hello!", body)
}
