package web

import (
	"html/template"
	"net/http"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
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

type foldersContent struct {
	Icons   map[sitecontent.Icon]string
	Folders []*sitecontent.Folder
}

type foldersHTTP struct {
	log         *ulog.Log
	store       database.Store
	environment runtime.Environment
	html        *template.Template
}

func (h *foldersHTTP) Path() string {
	return "/folders"
}

func (h *foldersHTTP) Methods() []string {
	return []string{http.MethodGet, http.MethodPost}
}

func (h *foldersHTTP) Metric() string {
	return "folders"
}

func (h *foldersHTTP) Consumer() consumer {
	return normal
}

func (h *foldersHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet:  h.get,
			http.MethodPost: h.post,
		},
	)
}

func (h *foldersHTTP) get(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
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

	folders, ferr := h.store.Folders().List(ctx, session.Identity())
	if ferr != nil {
		return abort(h.log, w, ferr)
	}

	head := &pages.Head{
		Title:       format.PageTitle("Manage folders"),
		Styles:      []string{"folders.css"},
		Description: h.description(),
		Profile:     profile,
	}

	middle := &foldersContent{
		Icons:   sitecontent.FolderIcons,
		Folders: folders,
	}

	return pages.New[identity.UserSession[users.Identity]](h.html, head, middle, nil).Execute(w, session)
}

func (h *foldersHTTP) post(ctx scope.C, w http.ResponseWriter, r *http.Request) error {
	session := middles.GetSession[users.Identity](r)

	// if user is not logged in, redirect to login page
	if !session.Active() {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil
	}

	// extract the fields supplied by the create-folder form submission
	request := h.fields(r, session.Identity())

	// validate business rules for the folder label
	if err := request.Validate(); err != nil {
		return problem(w, err)
	}

	if cerr := h.store.Folders().Create(ctx, request); cerr != nil {
		return abort(h.log, w, cerr)
	}

	// send the user back to the home page where folders are listed
	return htmx.SetRedirect(w, "/")
}

func (h *foldersHTTP) description() string {
	const text = `
	Create folders to store passwords into.`
	return format.PageDescription(text)
}

func (*foldersHTTP) fields(r *http.Request, user users.Identity) *sitecontent.CreateFolderRequest {
	var label string
	var icon int
	forms.MustParse(r, forms.Schema{
		"label": forms.String(&label),
		"icon":  forms.Int(&icon),
	})
	return &sitecontent.CreateFolderRequest{
		User:  user,
		Label: label,
		Icon:  sitecontent.Icon(icon),
	}
}
