package users

import (
	"testing"

	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func TestValidatePasskey(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		p := conceal.New("abcdefghijklmnopqrstuvwxyz")
		err := ValidatePasskey(p)
		must.NoError(t, err)
	})

	t.Run("short", func(t *testing.T) {
		p := conceal.New("abc123")
		err := ValidatePasskey(p)
		must.Error(t, err)
	})
}

func TestProfile_Active(t *testing.T) {
	t.Parallel()

	t.Run("yes", func(t *testing.T) {
		p := &Profile{
			ID: 1,
		}
		must.True(t, p.Active())
	})

	t.Run("no", func(t *testing.T) {
		p := (*Profile)(nil)
		must.False(t, p.Active())
	})
}

func TestProfile_Identity(t *testing.T) {
	t.Parallel()

	t.Run("active", func(t *testing.T) {
		p := &Profile{
			ID: 3,
		}
		must.Eq(t, 3, p.Identity().(Identity))
	})

	t.Run("inactive", func(t *testing.T) {
		p := (*Profile)(nil)
		must.Eq(t, Inactive, p.Identity())
	})
}
