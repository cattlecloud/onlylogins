package web

import (
	"html/template"
	"net/http"
	"path"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/urlpath"
	"cattlecloud.net/go/webtools/htmx"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/agent/web/pages"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/format"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type folderContent struct {
	Name      string
	Documents []*sitecontent.DocumentListing
}

type folderHTTP struct {
	log         *ulog.Log
	store       database.Store
	environment runtime.Environment
	html        *template.Template
}

func (h *folderHTTP) Path() string {
	return "/folders/{folder}"
}

func (h *folderHTTP) Methods() []string {
	return []string{http.MethodGet, http.MethodPost}
}

func (h *folderHTTP) Metric() string {
	return "folder"
}

func (h *folderHTTP) Consumer() consumer {
	return normal
}

func (h *folderHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:  h.get,
			http.MethodPost: h.post,
		},
	)
}

func (h *folderHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if user is not logged in, redirect to login page
	if !session.Active() {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil
	}

	profile, perr := getUserProfile(ctx, h.store.Users(), session)
	if perr != nil {
		return abort(h.log, w, perr)
	}

	name := h.folder(r)

	documents, lerr := h.store.Documents().List(ctx, &sitecontent.ListDocumentsRequest{
		User:   session.Identity(),
		Folder: name,
	})
	if lerr != nil {
		return abort(h.log, w, lerr)
	}

	head := &pages.Head{
		Title:       "Manage folder | OnlyLogins",
		Styles:      []string{"folder.css"},
		Description: h.description(),
		Profile:     profile,
	}

	middle := &folderContent{
		Name:      name,
		Documents: documents,
	}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, middle, nil).Execute(w, session)
}

func (h *folderHTTP) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if user is not logged in, redirect to login page
	if !session.Active() {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil
	}

	request := h.fields(r, session.Identity())

	// validate business rules for the document label
	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if _, derr := h.store.Documents().Create(ctx, request); derr != nil {
		return abort(h.log, w, derr)
	}

	// send the user to the document page (handler does not exist yet)
	target := path.Join("/folders", request.Folder, request.Label)
	return htmx.SetRedirect(w, target)
}

func (h *folderHTTP) fields(r *http.Request, user users.Identity) *sitecontent.CreateDocumentRequest {
	var label string
	forms.MustParse(r, forms.Schema{
		"label": forms.String(&label),
	})
	return &sitecontent.CreateDocumentRequest{
		User:   user,
		Folder: h.folder(r),
		Label:  label,
	}
}

func (h *folderHTTP) description() string {
	const text = `
	Manage contents of a folder.`
	return format.PageDescription(text)
}

func (h *folderHTTP) folder(r *http.Request) string {
	var folder string
	urlpath.MustParse(r, urlpath.Schema{
		"folder": urlpath.String(&folder),
	})
	return folder
}
