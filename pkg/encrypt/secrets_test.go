package encrypt

import (
	"testing"

	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func TestDeriveKey(t *testing.T) {
	t.Parallel()

	passphrase := conceal.New("correct horse battery staple")
	salt := conceal.NewBytes([]byte("0123456789abcdef"))

	key := DeriveKey(passphrase, salt)
	must.Eq(t, 32, key.Len())
}

func TestMakeVerifier_Verify(t *testing.T) {
	t.Parallel()

	key := conceal.NewBytes([]byte("0123456789abcdef0123456789abcdef"))

	verifier := MakeVerifier(key)
	must.True(t, Verify(key, verifier))

	other := conceal.NewBytes([]byte("fedcba9876543210fedcba9876543210"))
	must.False(t, Verify(other, verifier))
}

func TestNewSalt(t *testing.T) {
	t.Parallel()

	salt := Salt()
	must.Eq(t, saltLength, salt.Len())
}
