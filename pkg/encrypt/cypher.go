// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"

	"cattlecloud.net/onlylogins/pkg/tpm"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/ignore"
)

type Cypher interface {
	Encrypt(*conceal.Text) (*conceal.Bytes, error)
	Decrypt(*conceal.Bytes) (*conceal.Text, error)
}

type cypher struct {
	_   [0]func()
	key [32]byte
}

func (v *cypher) String() string {
	return "(cypher)"
}

func (v *cypher) GoString() string {
	return "encrypt.cypher{}"
}

func New(key *conceal.Bytes) Cypher {
	var k [32]byte
	copy(k[:], key.Unveil())
	return &cypher{key: k}
}

// NewFromConfig derives a Cypher from the server-wide passphrase held in c.
//
// When c.Passphrase is set (local development) it is used directly. Otherwise
// the passphrase is unsealed from the TPM at c.Device using the blob at
// c.Sealed. The 32-byte AES-GCM key is the SHA-256 of a fixed
// domain-separation string concatenated with the passphrase. This
// intentionally avoids Argon2id (which is used for per-user keys via
// DeriveKey) because the passphrase is not subject to slow brute force from
// the operator.
//
// If the passphrase cannot be resolved, the process exits with a non-zero
// code.
func NewFromConfig(c *Config) Cypher {
	passphrase, err := resolvePassphrase(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to load vault: %v\n", err)
		os.Exit(1)
	}
	return newFromPassphrase(passphrase)
}

// newFromPassphrase derives the cypher key from passphrase.
func newFromPassphrase(passphrase string) Cypher {
	// NOTE: the salt string is frozen; changing it would invalidate the
	// ciphertext of every share link already in the wild
	sum := sha256.Sum256([]byte("onlylogins-vault-salt+" + passphrase))
	var key [32]byte
	copy(key[:], sum[:])
	return &cypher{key: key}
}

// resolvePassphrase returns the cypher passphrase from c, either directly or by
// unsealing it from the TPM.
func resolvePassphrase(c *Config) (string, error) {
	if c.Passphrase != "" {
		return c.Passphrase, nil
	}

	data, err := os.ReadFile(c.Sealed)
	if err != nil {
		return "", fmt.Errorf("unable to read sealed blob: %w", err)
	}

	sealed, err := tpm.DecodeSealed(data)
	if err != nil {
		return "", err
	}

	conn, err := tpm.Open(c.Device)
	if err != nil {
		return "", err
	}
	defer ignore.Close(conn)

	unsealed, err := tpm.Unseal(conn, sealed)
	if err != nil {
		return "", err
	}

	passphrase := strings.TrimRight(string(unsealed), "\r\n")
	if err := ValidatePassphrase(passphrase); err != nil {
		return "", err
	}

	return passphrase, nil
}

func (v *cypher) Encrypt(message *conceal.Text) (*conceal.Bytes, error) {
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

func (v *cypher) Decrypt(ciphertext *conceal.Bytes) (*conceal.Text, error) {
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
