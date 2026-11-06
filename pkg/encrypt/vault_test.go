// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package encrypt

import (
	"testing"

	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func TestVault_EncryptDecrypt(t *testing.T) {
	t.Parallel()

	v := New(conceal.NewBytes([]byte{1, 2, 3, 4, 5}))
	original := conceal.New("hello world")

	encrypted, err := v.Encrypt(original)
	must.NoError(t, err)

	decrypted, err := v.Decrypt(encrypted)
	must.NoError(t, err)
	must.Eq(t, original.Unveil(), decrypted.Unveil())
}

func TestVault_NewFromConfig(t *testing.T) {
	t.Parallel()

	// two vaults built from the same passphrase must round-trip
	a := NewFromConfig(&Config{Passphrase: "11111111-2222-3333-4444-555555555555"})
	b := NewFromConfig(&Config{Passphrase: "11111111-2222-3333-4444-555555555555"})

	original := conceal.New("a top secret hush-hush thing")
	encrypted, err := a.Encrypt(original)
	must.NoError(t, err)

	decrypted, err := b.Decrypt(encrypted)
	must.NoError(t, err)
	must.Eq(t, original.Unveil(), decrypted.Unveil())
}

func TestVault_NewFromConfig_DifferentPassphrasesDoNotMatch(t *testing.T) {
	t.Parallel()

	a := NewFromConfig(&Config{Passphrase: "11111111-2222-3333-4444-555555555555"})
	b := NewFromConfig(&Config{Passphrase: "66666666-7777-8888-9999-aaaaaaaaaaaa"})

	original := conceal.New("a top secret hush-hush thing")
	encrypted, err := a.Encrypt(original)
	must.NoError(t, err)

	_, derr := b.Decrypt(encrypted)
	must.Error(t, derr)
}

func TestVault_ConfigValidate(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		var c *Config
		err := c.Validate()
		must.Error(t, err)
	})

	t.Run("empty passphrase", func(t *testing.T) {
		err := (&Config{}).Validate()
		must.Error(t, err)
	})

	t.Run("not quasi-uuid4", func(t *testing.T) {
		err := (&Config{Passphrase: "anything"}).Validate()
		must.Error(t, err)
	})

	t.Run("wrong length", func(t *testing.T) {
		err := (&Config{Passphrase: "11111111-2222-3333-4444-55555555555"}).Validate()
		must.Error(t, err)
	})

	t.Run("non-hex characters", func(t *testing.T) {
		err := (&Config{Passphrase: "11111111-2222-3333-4444-55555555555g"}).Validate()
		must.Error(t, err)
	})

	t.Run("valid", func(t *testing.T) {
		err := (&Config{Passphrase: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}).Validate()
		must.NoError(t, err)
	})
}
