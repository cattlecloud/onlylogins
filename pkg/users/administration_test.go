package users

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestCouncil_IsAdmin(t *testing.T) {
	t.Parallel()

	c := NewCouncil([]Identity{2, 98})

	must.True(t, c.IsAdmin(2))
	must.True(t, c.IsAdmin(98))
	must.False(t, c.IsAdmin(1))
	must.False(t, c.IsAdmin(101))
}
