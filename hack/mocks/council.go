package mocks

import (
	"slices"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/pkg/users"
)

// Council is a mock implementation of users.Council.
type Council struct {
	Admins    []users.Identity
	Transmogs []users.Identity
}

func (c *Council) IsAdmin(id users.Identity) bool {
	return slices.Contains(c.Admins, id)
}

func (c *Council) AllowTransmog(_ scope.C, id users.Identity) bool {
	return slices.Contains(c.Transmogs, id) || c.IsAdmin(id)
}
