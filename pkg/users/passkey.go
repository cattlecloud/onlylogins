package users

import (
	"errors"
	"strings"

	"github.com/shoenig/go-conceal"
)

const (
	minimumPasskeyLength = 24
	saltLength           = 16
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
		return errors.New("passkey must not be nil")
	}

	if len(text.Unveil()) < minimumPasskeyLength {
		return errors.New("passkey must be at least 24 characters long")
	}

	return nil
}
