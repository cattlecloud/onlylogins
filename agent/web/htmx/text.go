package htmx

import (
	"fmt"
	"io"
	"net/http"

	"cattlecloud.net/go/webtools"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/pkg/users"
)

type text[T ~string] struct {
	content T
}

func (t *text[T]) Render(w http.ResponseWriter, _ identity.UserSession[users.Identity]) error {
	// set text/plain content type
	webtools.SetContentType(w, webtools.ContentTypeText)

	// write the text
	s := string(t.content)
	if _, err := io.WriteString(w, s); err != nil {
		panic(fmt.Sprintf("htmx: unable to render plain text: %v", err))
	}

	// the text has been written
	return nil
}
