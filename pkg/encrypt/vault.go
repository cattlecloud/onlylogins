// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/shoenig/go-conceal"
)

type Vault interface {
	Encrypt(*conceal.Text) (*conceal.Bytes, error)
	Decrypt(*conceal.Bytes) (*conceal.Text, error)
}

type vault struct {
	_   [0]func()
	key [32]byte
}

func (v *vault) String() string {
	return "(vault)"
}

func (v *vault) GoString() string {
	return "encrypt.vault{}"
}

func New(key *conceal.Bytes) Vault {
	var k [32]byte
	copy(k[:], key.Unveil())
	return &vault{key: k}
}

// NewFromConfig derives a Vault from a server-wide passphrase.
//
// The 32-byte AES-GCM key is the SHA-256 of a fixed domain-separation
// string concatenated with the passphrase. This intentionally avoids
// Argon2id (which is used for per-user keys via DeriveKey) because the
// passphrase is held in a config file and not subject to slow brute force
// from the operator.
//
// The passphrase must already have been validated via Config.Validate.
func NewFromConfig(c *Config) Vault {
	sum := sha256.Sum256([]byte("onlylogins-vault-salt+" + c.Passphrase))
	var key [32]byte
	copy(key[:], sum[:])
	return &vault{key: key}
}

func (v *vault) Encrypt(message *conceal.Text) (*conceal.Bytes, error) {
	block, berr := aes.NewCipher(v.key[:])
	if berr != nil {
		return nil, fmt.Errorf("unable to create encryption cipher: %w", berr)
	}

	gcm, gerr := cipher.NewGCM(block)
	if gerr != nil {
		return nil, fmt.Errorf("unable to create encryption gcm: %w", gerr)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("unable to create encryption nonce: %w", err)
	}

	bytes := gcm.Seal(nonce, nonce, []byte(message.Unveil()), nil)
	return conceal.NewBytes(bytes), nil
}

func (v *vault) Decrypt(ciphertext *conceal.Bytes) (*conceal.Text, error) {
	block, berr := aes.NewCipher(v.key[:])
	if berr != nil {
		return nil, fmt.Errorf("unable to create decrpytion cipher: %w", berr)
	}

	gcm, gerr := cipher.NewGCM(block)
	if gerr != nil {
		return nil, fmt.Errorf("unable to create decryption gcm: %w", gerr)
	}

	nsize := gcm.NonceSize()
	if ciphertext.Len() < nsize {
		return nil, fmt.Errorf("ciphertext is too short: %d", ciphertext.Len())
	}

	raw := ciphertext.Unveil()
	nonce, encrypted := raw[:nsize], raw[nsize:]
	plaintext, perr := gcm.Open(nil, nonce, encrypted, nil)
	if perr != nil {
		return nil, fmt.Errorf("unable to create decryption plaintext: %w", perr)
	}

	return conceal.New(string(plaintext)), nil
}
