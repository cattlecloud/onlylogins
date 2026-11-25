// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package users

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"cattlecloud.net/onlylogins/pkg/entropy"
	"github.com/shoenig/go-conceal"
)

const (
	// NIST SP 800-63 lays out guidelines for password authenticators.
	//
	// NIST requires verifiers to demand at least 15 characters of a password
	// used as a single authentication factor; we require one more. NIST sets
	// no entropy floor for user-chosen passwords.
	// Appendix A of the publication deliberately works from length instead,
	// since the entropy of a chosen passphrase is difficult to measure, so the
	// entropy requirement below is a house rule of our own.
	//
	// Some casual reading on the subject:
	// https://pages.nist.gov/800-63-4/sp800-63b.html
	minimumPassphraseLength  = 16 // NIST SP 800-63B requires 15
	minimumPassphraseEntropy = 72 // house rule; see above
	saltLength               = 16 // house rule
)

func ValidateEmail(s string) error {
	if len(s) < 3 {
		return errors.New("email is too short")
	}

	if len(s) >= 128 {
		return errors.New("email is too long")
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

	if utf8.RuneCountInString(text.Unveil()) < minimumPassphraseLength {
		return fmt.Errorf("passphrase must be at least %d characters long", minimumPassphraseLength)
	}

	if entropy.Sequence(text.Unveil()) {
		return errors.New("passphrase contains 5 or more repeating or sequential characters")
	}

	if entropy.Bits(text.Unveil()) < minimumPassphraseEntropy {
		return errors.New("passphrase is not random enough")
	}

	return nil
}

// ChangePassphraseRequest captures the values needed to change a user's
// passphrase: the existing passphrase (verified before anything changes) and
// the new passphrase entered twice for confirmation.
type ChangePassphraseRequest struct {
	Previous   *conceal.Text
	Passphrase *conceal.Text
	Confirm    *conceal.Text
}

func (r *ChangePassphraseRequest) Validate() error {
	if r == nil {
		return errors.New("must not be nil")
	}

	if r.Previous.IsZero() {
		return errors.New("old passphrase must be set")
	}

	if err := ValidatePassphrase(r.Passphrase); err != nil {
		return err
	}

	if !r.Passphrase.Equal(r.Confirm) {
		return errors.New("passphrases do not match")
	}

	return nil
}
