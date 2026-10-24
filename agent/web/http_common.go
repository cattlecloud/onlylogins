// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/sessions/cookies"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

func getUserProfile(ctx scope.C, u database.Users, session identity.UserSession[users.Identity]) (*users.Profile, error) {
	if !session.Active() {
		return nil, nil //nolint: nilnil
	}

	p, perr := u.Lookup(ctx, session.Identity())
	if perr != nil {
		return nil, fmt.Errorf("unable to get user profile: %w", perr)
	}

	return p, nil
}

// getVault reconstructs the user's encryption Vault from the key cookie
// set during login.
//
// Returns ErrMissingVault when there is no encryption cookie, or
// ErrInvalidVault when the cookie value is malformed or has the wrong length.
func getVault(r *http.Request) (encrypt.Vault, error) {
	ck, cerr := r.Cookie(cookies.Encryption)
	if errors.Is(cerr, http.ErrNoCookie) {
		return nil, ErrMissingVault
	} else if cerr != nil {
		return nil, fmt.Errorf("unable to read vault cookie: %w", cerr)
	}

	raw, derr := base64.StdEncoding.DecodeString(ck.Value)
	if derr != nil {
		return nil, fmt.Errorf("unable to decode vault cookie: %w", derr)
	}

	if len(raw) != 32 {
		return nil, ErrInvalidVault
	}

	return encrypt.New(conceal.NewBytes(raw)), nil
}
