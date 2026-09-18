package web

import (
	"fmt"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/webtools/middles/identity"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

func getUserInfo(ctx scope.C, u database.Users, session identity.UserSession[users.Identity]) (*sitecontent.UserInfo, error) {
	if !session.Active() {
		return nil, nil //nolint: nilnil
	}

	ui, uerr := u.Info(ctx, session.Identity())
	if uerr != nil {
		return nil, fmt.Errorf("unable to get profile from session: %w", uerr)
	}

	return ui, nil
}

func getUserProfile(ctx scope.C, u database.Users, session identity.UserSession[users.Identity]) (*sitecontent.UserProfile, error) {
	if !session.Active() {
		return nil, nil // nolint: nilnil
	}

	profile, uerr := u.Profile(ctx, session.Identity())
	if uerr != nil {
		return nil, fmt.Errorf("unable to get profile from session: %w", uerr)
	}

	return profile, nil
}
