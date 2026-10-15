package encrypt

import "github.com/shoenig/go-conceal"

type Config struct {
	Passkey *conceal.Text
	Salt    *conceal.Text
}
