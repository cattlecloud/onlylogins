package users

import (
	"slices"
	"testing"

	"cattlecloud.net/go/scope"
	"github.com/shoenig/test/must"
)

type ourStore struct {
	mogs []Identity
}

func (s *ourStore) TransmogCapable(_ scope.C, id Identity) (bool, error) {
	return slices.Contains(s.mogs, id), nil
}

func TestCouncil_IsAdmin(t *testing.T) {
	t.Parallel()

	store := &ourStore{mogs: nil}
	c := NewCouncil([]Identity{2, 98}, store)

	must.True(t, c.IsAdmin(2))
	must.True(t, c.IsAdmin(98))
	must.False(t, c.IsAdmin(1))
	must.False(t, c.IsAdmin(101))
}

func TestCouncil_AllowTransmog(t *testing.T) {
	t.Parallel()

	store := &ourStore{mogs: []Identity{2, 3, 9}}
	c := NewCouncil([]Identity{4, 8}, store)
	must.False(t, c.AllowTransmog(t.Context(), 1))
	must.True(t, c.AllowTransmog(t.Context(), 2))
	must.True(t, c.AllowTransmog(t.Context(), 3))
	must.False(t, c.AllowTransmog(t.Context(), 5))
	must.True(t, c.AllowTransmog(t.Context(), 9))
	must.False(t, c.AllowTransmog(t.Context(), 10))
}
