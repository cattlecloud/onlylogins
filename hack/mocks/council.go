package mocks

import (
	"slices"

	"cattlecloud.net/onlylogins/pkg/users"
)

// Council is a mock implementation of users.Council.
type Council struct {
	Admins []users.Identity
}

func (c *Council) IsAdmin(id users.Identity) bool {
	return slices.Contains(c.Admins, id)
}
