package users

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		c := (*Config)(nil)
		err := c.Validate()
		must.ErrorContains(t, err, "users config must be set")
	})

	t.Run("no administrators", func(t *testing.T) {
		c := &Config{}
		err := c.Validate()
		must.ErrorContains(t, err, "users administrators config must be set")
	})

	t.Run("ok", func(t *testing.T) {
		c := &Config{
			Administrators: []Identity{42},
		}
		err := c.Validate()
		must.NoError(t, err)
	})
}
