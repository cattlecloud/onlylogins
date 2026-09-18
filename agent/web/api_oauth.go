package web

import (
	"net/http"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools"
	"cattlecloud.net/go/webtools/middles/oauth/nonces"
)

type oauthAPI struct {
	log    *ulog.Log
	config Oauth
	mint   nonces.Mint
}

func (h *oauthAPI) Path() string {
	return "/login/oauth"
}

func (h *oauthAPI) Methods() []string {
	// IDPs expect a particular method on redirect, and they do not agree.
	return []string{http.MethodPost, http.MethodGet}
}

func (h *oauthAPI) Consumer() consumer {
	return robotic
}

func (h *oauthAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch h.idpFrom(r) {
	case "google":
		h.google(w, r)
	case "apple":
		h.apple(w, r)
	case "microsoft":
		h.microsoft(w, r)
	default:
		http.Error(w, "no such provider", http.StatusNotFound)
	}
}

func (h *oauthAPI) google(w http.ResponseWriter, r *http.Request) {
	h.log.D.Fmt("got google login request")

	// create a nonce specific to this authentication request
	nonce := h.mint.Create()

	// create redirect to the google auth endpoint
	const domain = "https://accounts.google.com"
	const endpoint = "/o/oauth2/v2/auth"
	u := webtools.CreateURL(domain, endpoint, map[string]string{
		"client_id":     h.config.Google.ClientID,
		"redirect_uri":  h.config.Google.CallbackURI,
		"response_type": "code",
		"access_type":   "online",
		"scope":         "openid email",
		"state":         "nonce=" + nonce.Unveil(),
	}).String()

	// execute the redirect; next we receive a request on the callback
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
}

func (h *oauthAPI) apple(_ http.ResponseWriter, _ *http.Request) {
	h.log.D.Fmt("got apple login request")

	// TODO
}

func (h *oauthAPI) microsoft(_ http.ResponseWriter, _ *http.Request) {
	h.log.D.Fmt("got microsoft login request")

	// TODO
}

func (h *oauthAPI) idpFrom(r *http.Request) string {
	var idp string
	_ = forms.Parse(r, forms.Schema{
		"idp": forms.String(&idp),
	})
	return idp
}
