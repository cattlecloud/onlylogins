// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

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
	Email    string
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
	Salt          *conceal.Bytes
	Passphrase    *conceal.Text
	Confirm       *conceal.Text
	Notifications bool
	Spam          bool
}

func (r *CreateUserRequest) Validate() error {
	if r == nil {
		return errors.New("must not be nil")
	}

	if err := ValidateEmail(r.Email); err != nil {
		return err
	}

	if !AllowUsername(r.Username) {
		return errors.New("username is not allowed")
	}

	if err := ValidateSalt(r.Salt); err != nil {
		return err
	}

	if err := ValidatePassphrase(r.Passphrase); err != nil {
		return err
	}

	return nil
}

type LoginRequest struct {
	Username   string
	Passphrase *conceal.Text
	UserAgent  string
	IP         string
}

func (r *LoginRequest) Validate() error {
	if r == nil {
		return errors.New("must not be nil")
	}

	if r.Username == "" {
		return errors.New("username must be set")
	}

	if err := ValidatePassphrase(r.Passphrase); err != nil {
		return err
	}

	return nil
}
