// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package mocks

import (
	"encoding/base64"
	"net/http"

	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/sessions/cookies"
	"github.com/shoenig/go-conceal"
)

// TestCypher bundles a deterministic 32-byte AES key with the Cypher.
type TestCypher struct {
	key    []byte
	cypher encrypt.Cypher
}

// NewTestCypher returns a TestCypher whose 32-byte key is the deterministic
// sequence [0, 1, 2, ..., 31].
func NewTestCypher() TestCypher {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return TestCypher{
		key:    append([]byte(nil), k...),
		cypher: encrypt.New(conceal.NewBytes(k)),
	}
}

// Cypher returns the encrypt.Cypher derived from the test key.
func (c TestCypher) Cypher() encrypt.Cypher {
	return c.cypher
}

// Cookie returns the onlylogins-encryption cookie.
func (c TestCypher) Cookie() *http.Cookie {
	return &http.Cookie{
		Name:  cookies.Encryption,
		Value: base64.StdEncoding.EncodeToString(c.key),
	}
}
