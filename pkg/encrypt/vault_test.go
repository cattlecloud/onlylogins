package encrypt

import (
	"testing"

	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func TestVault_EncryptDecrypt(t *testing.T) {
	t.Parallel()

	v := New(&Config{
		Salt:    conceal.UUIDv4(),
		Passkey: conceal.UUIDv4(),
	})
	original := conceal.New("hello world")

	encrypted, err := v.Encrypt(original)
	must.NoError(t, err)

	decrypted, err := v.Decrypt(encrypted)
	must.NoError(t, err)
	must.Eq(t, original.Unveil(), decrypted.Unveil())
}
