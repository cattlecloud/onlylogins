// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package mocks

import (
	"encoding/base64"
	"net/http"

	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/sessions/cookies"
	"github.com/shoenig/go-conceal"
)

// TestCrypt bundles a deterministic 32-byte AES key with the Vault.
type TestCrypt struct {
	key   []byte
	vault encrypt.Vault
}

// NewTestCrypt returns a TestCrypt whose 32-byte key is the deterministic
// sequence [0, 1, 2, ..., 31].
func NewTestCrypt() TestCrypt {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return TestCrypt{
		key:   append([]byte(nil), k...),
		vault: encrypt.New(conceal.NewBytes(k)),
	}
}

// Vault returns the encrypt.Vault derived from the test key.
func (c TestCrypt) Vault() encrypt.Vault {
	return c.vault
}

// Cookie returns the onlylogins-encryption cookie.
func (c TestCrypt) Cookie() *http.Cookie {
	return &http.Cookie{
		Name:  cookies.Encryption,
		Value: base64.StdEncoding.EncodeToString(c.key),
	}
}
