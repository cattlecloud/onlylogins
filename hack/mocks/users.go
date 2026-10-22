package mocks

import (
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

const Nobody = users.Identity(1)

var Pass = conceal.New("abcdefghijklmnopqrstuvwxyz")
