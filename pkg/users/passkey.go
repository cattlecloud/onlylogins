package users

import (
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/shoenig/go-conceal"
)

const (
	minimumPasskeyLength = 24
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

func ValidateIntermediate(secret *conceal.Text) error {
	if secret == nil {
		return errors.New("intermediate must not be nil")
	}

	if _, err := uuid.Parse(secret.Unveil()); err != nil {
		return fmt.Errorf("intermediate must be uuid: %w", err)
	}

	return nil
}

func ValidatePassphrase(text *conceal.Text) error {
	if text == nil {
		return errors.New("passkey must not be nil")
	}

	if len(text.Unveil()) < minimumPasskeyLength {
		return errors.New("passkey must be at least 16 characters long")
	}

	return nil
}
