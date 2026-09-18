package users

import "errors"

type Config struct {
	Administrators []Identity `toml:"admins"`
}

func (c *Config) Validate() error {
	switch {
	case c == nil:
		return errors.New("users config must be set")
	case len(c.Administrators) == 0:
		return errors.New("users administrators config must be set")
	default:
		return nil
	}
}
