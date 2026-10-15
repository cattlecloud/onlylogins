package users

import (
	"errors"
	"fmt"
	"uuid"

	"github.com/shoenig/go-conceal"
)

const (
	minimumPasskeyLength = 24
)

func ValidateShibboleth(text *conceal.Text) error {
	if text == nil {
		return errors.New("shibboleth must not be nil")
	}

	s := text.Unveil()

	if _, err := uuid.Parse(s); err != nil {
		return fmt.Errorf("shibboleth must be uuid: %w", err)
	}

	return nil
}

func ValidatePasskey(text *conceal.Text) error {
	if text == nil {
		return errors.New("passkey must not be nil")
	}

	s := text.Unveil()

	if len(s) < minimumPasskeyLength {
		return errors.New("passkey must be at least 16 characters long")
	}

	return nil
}
