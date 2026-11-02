// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package encrypt

import "errors"

// Config is the configuration for the server-wide Vault used to encrypt
// features whose ciphertext is meant to be readable by anyone (e.g. share
// links received by a third party).
//
// The Passphrase is hashed (with a fixed domain-separation string) to derive
// the 32-byte AES-GCM key stored in the Vault.
type Config struct {
	Passphrase string `toml:"passphrase"`
}

func (c *Config) Validate() error {
	switch {
	case c == nil:
		return errors.New("vault config must be set")
	case c.Passphrase == "":
		return errors.New("vault passphrase must be set")
	default:
		return nil
	}
}
