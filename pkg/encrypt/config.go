// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package encrypt

import (
	"errors"
	"regexp"
)

// globalPassphraseRe matches a uuid4(ish) string, where we ignore version bits
var globalPassphraseRe = regexp.MustCompile(`^[[:xdigit:]-]{36}$`)

// Config is the configuration for the server-wide Vault used to encrypt
// features whose ciphertext is meant to be readable by anyone (e.g. share
// links received by a third party).
//
// The Passphrase is hashed (with a fixed domain-separation string) to derive
// the 32-byte AES-GCM key stored in the Vault.
type Config struct {
	// Passphrase is the plaintext passphrase, suitable only for local
	// development. It is mutually exclusive with Sealed.
	Passphrase string `toml:"passphrase"`

	// Sealed is the path to a passphrase sealed to a TPM. It is mutually
	// exclusive with Passphrase and required in production.
	Sealed string `toml:"sealed"`
}

// ValidatePassphrase reports whether passphrase matches the required format.
func ValidatePassphrase(passphrase string) error {
	if !globalPassphraseRe.MatchString(passphrase) {
		return errors.New("vault passphrase must be in quasi-uuid4 format")
	}
	return nil
}

func (c *Config) Validate() error {
	switch {
	case c == nil:
		return errors.New("vault config must be set")
	case c.Passphrase != "" && c.Sealed != "":
		return errors.New("vault passphrase and sealed are mutually exclusive")
	case c.Passphrase == "" && c.Sealed == "":
		return errors.New("vault passphrase or sealed must be set")
	case c.Passphrase != "":
		return ValidatePassphrase(c.Passphrase)
	default:
		return nil
	}
}
