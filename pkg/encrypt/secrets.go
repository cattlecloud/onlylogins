package encrypt

import (
	"crypto/sha256"

	"github.com/shoenig/go-conceal"
)

var (
	Litmus = conceal.New("litmus_test")
)

// Combine the passphrase and the intermediate secret to create the fundamental
// encryption secret for the user of the passphrase.
func Combine(passphrase, intermediate *conceal.Text) *conceal.Bytes {
	a := passphrase.Unveil()
	b := intermediate.Unveil()
	s := "+combination+"
	z := sha256.Sum256([]byte(a + s + b))
	return conceal.NewBytes(z[:])
}

type Inputs struct {
	Secret       *conceal.Bytes
	Intermediate *conceal.Text
}

func Expect(passphrase, intermediate *conceal.Text) (*conceal.Bytes, error) {
	secret := Combine(passphrase, intermediate)
	v := New(&Inputs{
		Secret:       secret,
		Intermediate: intermediate,
	})
	return v.Encrypt(Litmus)
}
