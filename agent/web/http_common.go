// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

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

// getCypher reconstructs the user's Cypher from the key cookie set during
// login.
//
// Returns ErrMissingCypher when there is no encryption cookie, or
// ErrInvalidCypher when the cookie value is malformed or has the wrong length.
func getCypher(r *http.Request) (encrypt.Cypher, error) {
	ck, cerr := r.Cookie(cookies.Encryption)
	if errors.Is(cerr, http.ErrNoCookie) {
		return nil, ErrMissingCypher
	} else if cerr != nil {
		return nil, fmt.Errorf("unable to read cypher cookie: %w", cerr)
	}

	raw, derr := base64.StdEncoding.DecodeString(ck.Value)
	if derr != nil {
		return nil, fmt.Errorf("unable to decode cypher cookie: %w", derr)
	}

	if len(raw) != 32 {
		return nil, ErrInvalidCypher
	}

	return encrypt.New(conceal.NewBytes(raw)), nil
}
