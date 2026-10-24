// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
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
