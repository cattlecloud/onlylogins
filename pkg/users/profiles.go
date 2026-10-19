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
	Email         string
	Username      string
	Intermediate  *conceal.Text
	Passphrase    *conceal.Text
	Notifications bool
	Spam          bool
}

func (c *CreateUserRequest) Validate() error {
	if c == nil {
		return errors.New("must not be nil")
	}

	if err := ValidateEmail(c.Email); err != nil {
		return err
	}

	if err := ValidateIntermediate(c.Intermediate); err != nil {
		return err
	}

	if err := ValidatePassphrase(c.Passphrase); err != nil {
		return err
	}

	return nil
}
