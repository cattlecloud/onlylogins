package encrypt

import (
	"testing"

	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func TestVault_EncryptDecrypt(t *testing.T) {
	t.Parallel()

	v := New(&Inputs{
		Secret:       conceal.NewBytes([]byte{1, 2, 3, 4, 5}),
		Intermediate: conceal.UUIDv4(),
	})
	original := conceal.New("hello world")

	encrypted, err := v.Encrypt(original)
	must.NoError(t, err)

	decrypted, err := v.Decrypt(encrypted)
	must.NoError(t, err)
	must.Eq(t, original.Unveil(), decrypted.Unveil())
}
