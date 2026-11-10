// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package users

import (
	"errors"
	"strings"

	"cattlecloud.net/onlylogins/pkg/entropy"
	"github.com/shoenig/go-conceal"
)

const (
	// NIST SP 800-63 lays out guidelines for password authenticators, which we abide by and
	// exceed by a wide margin.
	//
	// Some casual reading on the subject:
	// https://pages.nist.gov/800-63-4/sp800-63b.html
	minimumPasskeyLength  = 24 // NIST is 15
	minimumPasskeyEntropy = 80 // NIST is 64
	saltLength            = 16 // custom
)

func ValidateEmail(s string) error {
	if len(s) < 3 {
		return errors.New("email is too short")
	}

	if !strings.Contains(s, "@") {
		return errors.New("email must contain @ symbol")
	}

	return nil
}

func ValidateSalt(salt *conceal.Bytes) error {
	if salt == nil {
		return errors.New("salt must not be nil")
	}

	if salt.Len() != saltLength {
		return errors.New("salt must be 16 bytes")
	}

	return nil
}

func ValidatePassphrase(text *conceal.Text) error {
	if text == nil {
		return errors.New("passphrase must not be nil")
	}

	if len(text.Unveil()) < minimumPasskeyLength {
		return errors.New("passphrase must be at least 24 characters long")
	}

	if entropy.Sequence(text.Unveil()) {
		return errors.New("passphrase contains 5 or more repeating or sequential characters")
	}

	if entropy.Bits(text.Unveil()) < minimumPasskeyEntropy {
		return errors.New("passphrase does not contain enough entropy")
	}

	return nil
}
