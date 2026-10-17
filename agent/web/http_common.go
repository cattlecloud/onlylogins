package web

import (
	"fmt"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/users"
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
