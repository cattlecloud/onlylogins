package users

import (
	"errors"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/webtools/middles/identity"
	"github.com/shoenig/go-conceal"
)

const (
	// Inactive is a sentinel value representing no user.
	Inactive = -1
)

// Identitiy is the unique database id assigned to each user profile.
//
// It is just an auto increment, but that is an implementation detail.
type Identity litesql.ID

type Profile struct {
	ID       Identity
	Username string
	Standing Standing
}

func (up *Profile) Active() bool {
	return up != nil
}

func (up *Profile) Identity() identity.UserIdentity {
	if up == nil {
		return Inactive
	}
	return up.ID
}

type CreateUserRequest struct {
	Username   string
	Shibboleth *conceal.Text
	Passkey    *conceal.Text
}

func (c *CreateUserRequest) Validate() error {
	if c == nil {
		return errors.New("must not be nil")
	}

	if err := ValidateShibboleth(c.Shibboleth); err != nil {
		return err
	}

	if err := ValidatePasskey(c.Passkey); err != nil {
		return err
	}

	return nil
}
