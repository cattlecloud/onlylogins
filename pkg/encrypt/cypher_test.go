// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package encrypt

import (
	"testing"

	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func TestCypher_EncryptDecrypt(t *testing.T) {
	t.Parallel()

	v := New(conceal.NewBytes([]byte{1, 2, 3, 4, 5}))
	original := conceal.New("hello world")

	encrypted, err := v.Encrypt(original)
	must.NoError(t, err)

	decrypted, err := v.Decrypt(encrypted)
	must.NoError(t, err)
	must.Eq(t, original.Unveil(), decrypted.Unveil())
}

func TestCypher_NewFromConfig(t *testing.T) {
	t.Parallel()

	// two cyphers built from the same passphrase must round-trip
	a := NewFromConfig(&Config{Passphrase: "11111111-2222-3333-4444-555555555555"})
	b := NewFromConfig(&Config{Passphrase: "11111111-2222-3333-4444-555555555555"})

	original := conceal.New("a top secret hush-hush thing")
	encrypted, err := a.Encrypt(original)
	must.NoError(t, err)

	decrypted, err := b.Decrypt(encrypted)
	must.NoError(t, err)
	must.Eq(t, original.Unveil(), decrypted.Unveil())
}

func TestCypher_NewFromConfig_DifferentPassphrasesDoNotMatch(t *testing.T) {
	t.Parallel()

	a := NewFromConfig(&Config{Passphrase: "11111111-2222-3333-4444-555555555555"})
	b := NewFromConfig(&Config{Passphrase: "66666666-7777-8888-9999-aaaaaaaaaaaa"})

	original := conceal.New("a top secret hush-hush thing")
	encrypted, err := a.Encrypt(original)
	must.NoError(t, err)

	_, derr := b.Decrypt(encrypted)
	must.Error(t, derr)
}

func TestCypher_ConfigValidate(t *testing.T) {
	t.Parallel()

	t.Run("nil config", func(t *testing.T) {
		var c *Config
		err := c.Validate()
		must.Error(t, err)
	})

	t.Run("neither passphrase nor sealed", func(t *testing.T) {
		err := (&Config{}).Validate()
		must.Error(t, err)
	})

	t.Run("both passphrase and sealed", func(t *testing.T) {
		err := (&Config{
			Passphrase: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			Sealed:     "/etc/onlylogins/sealed",
		}).Validate()
		must.Error(t, err)
	})

	t.Run("sealed with device", func(t *testing.T) {
		err := (&Config{
			Sealed: "/etc/onlylogins/sealed",
			Device: "/dev/tpmrm0",
		}).Validate()
		must.NoError(t, err)
	})

	t.Run("sealed without device", func(t *testing.T) {
		err := (&Config{Sealed: "/etc/onlylogins/sealed"}).Validate()
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

func TestValidatePassphrase(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		must.NoError(t, ValidatePassphrase("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	})

	t.Run("invalid", func(t *testing.T) {
		must.Error(t, ValidatePassphrase("anything"))
	})

	t.Run("empty", func(t *testing.T) {
		must.Error(t, ValidatePassphrase(""))
	})
}
