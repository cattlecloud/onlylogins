package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"cattlecloud.net/go/forms"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools"
	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/go/webtools/middles/oauth/googlekeys"
	"cattlecloud.net/go/webtools/middles/oauth/nonces"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/pkg/users/oauths"
	"github.com/shoenig/go-conceal"
)

type googleAPI struct {
	log      *ulog.Log
	config   Oauth
	httpc    *http.Client
	gkeys    googlekeys.Validator
	store    database.Store
	sessions middles.Sessions[users.Identity]
	mint     nonces.Mint
}

func (h *googleAPI) Path() string {
	return "/login/oauth/google"
}

func (h *googleAPI) Methods() []string {
	return []string{http.MethodGet}
}

func (h *googleAPI) Consumer() consumer {
	return robotic
}

func (h *googleAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// the full oauth flow is multiple requests and some database writes;
	// use a long ttl because some connections are just going to be slow
	ctx, cancel := scope.WithTTL(r.Context(), 1*time.Minute)
	defer cancel()

	code, state, cerr := h.authCodeFrom(r)
	if cerr != nil {
		h.log.E.Fmt("unable to read oauth phase-1 parameters: %v", cerr)
		return
	}

	// ensure the state nonce is legit
	nonce := h.nonceFrom(state)
	if err := h.mint.Consume(nonce); err != nil {
		h.log.W.Fmt("caught attempt to use unknown oauth nonce: %v", err)
		return
	}

	// exchange authorization code for refresh / access tokens
	const domain = "https://oauth2.googleapis.com"
	const endpoint = "/token"
	u := webtools.CreateURL(domain, endpoint, map[string]string{
		"client_id":     h.config.Google.ClientID,
		"client_secret": h.config.Google.ClientSecret,
		"code":          code,
		"grant_type":    "authorization_code",
		"redirect_uri":  h.config.Google.CallbackURI,
	}).String()

	post, rerr := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if rerr != nil {
		h.log.E.Fmt("unable to create oauth exchange request: %v", rerr)
		return
	}
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// google step 5
	cookie, path, xerr := h.exchange(ctx, post)
	if xerr != nil {
		h.log.W.Fmt("unable to complete oauth exchange: %v", xerr)
		return
	}

	// set the profile cookie and redirect
	http.SetCookie(w, cookie)
	http.Redirect(w, r, path, http.StatusFound)
}

var nonceRe = regexp.MustCompile(`nonce=([a-f0-9-]{36})`)

func (*googleAPI) nonceFrom(state string) *conceal.Text {
	// should be in the form nonce=<uuid>; return empty string if not
	results := nonceRe.FindStringSubmatch(state)
	if len(results) != 2 {
		return conceal.New("")
	}
	return conceal.New(results[1])
}

func (*googleAPI) authCodeFrom(request *http.Request) (string, string, error) {
	var (
		code  string
		state string
		fail  string
	)

	if err := forms.Parse(request, forms.Schema{
		"code":  forms.String(&code),
		"state": forms.String(&state),
		"error": forms.StringOr(&fail, ""),
	}); err != nil {
		return "", "", err
	}

	if fail == "" {
		return code, state, nil
	}

	return "", "", errors.New(fail)
}

func (h *googleAPI) exchange(ctx scope.C, post *http.Request) (*http.Cookie, string, error) {
	// Step 5. Exchange authorization code for refresh and access tokens
	// https://developers.google.com/identity/protocols/oauth2/web-server#exchange-authorization-code

	response, rerr := h.httpc.Do(post)
	if rerr != nil {
		return nil, "", fmt.Errorf("failure in token exchange: %w", rerr)
	}
	defer func() { _ = response.Body.Close() }()

	var tokens struct {
		ID      string `json:"id_token"`
		Access  string `json:"access_token"`
		Refresh string `json:"refresh_token"`
		Expires int    `json:"expires_in"`
		Type    string `json:"token_type"`
	}

	if err := json.NewDecoder(response.Body).Decode(&tokens); err != nil {
		return nil, "", fmt.Errorf("failure in decoding token payload: %w", err)
	}

	claims, verr := h.gkeys.Validate(tokens.ID)
	if verr != nil {
		return nil, "", fmt.Errorf("failure in validating token claims: %w", verr)
	}

	// execute redirection stage
	return h.redirect(ctx, claims)
}

func (h *googleAPI) redirect(ctx scope.C, claims *googlekeys.Claims) (*http.Cookie, string, error) {
	// create profile if necessary then redirect to home or profile page

	store := h.store.Users()
	sub := oauths.Subject(claims.Subject)

	// lookup the user; need to know if they already exist
	id, exists, lerr := store.Exists(ctx, oauths.Google, sub)
	if lerr != nil {
		return nil, "", fmt.Errorf("failure in looking up user: %w", lerr)
	}

	// if the user already exists, just bake them a cookie and send them on
	// their way (currently only to the homepage)
	if exists {
		oid := users.Identity(id)
		cookie := h.sessions.Create(oid, Annual)
		return cookie, "/", nil
	}

	// otherwise we generate a new profile for the user & store it in database
	profile := users.InitialProfile(oauths.Google, sub, claims.Email)
	auto, ierr := store.Create(ctx, profile)
	if ierr != nil {
		return nil, "", fmt.Errorf("failure in creating user: %w", ierr)
	}

	// bake a cookie for our new user and redirect to profile page
	oid := users.Identity(auto)
	cookie := h.sessions.Create(oid, Annual)
	return cookie, "/profile", nil
}
