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
