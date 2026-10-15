package web

import (
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/users"
)

func getUserProfile(ctx scope.C, u database.Users, session identity.UserSession[users.Identity]) (*users.Profile, error) {
	if !session.Active() {
		return nil, nil //nolint: nilnil
	}

	_ = ctx
	_ = u
	_ = session

	// todo get from database
	//	ui, uerr := u.Info(ctx, session.Identity())
	//	if uerr != nil {
	//		return nil, fmt.Errorf("unable to get profile from session: %w", uerr)
	//	}

	// return ui, nil
	return nil, nil //nolint: nilnil
}
