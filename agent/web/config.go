package web

import (
	"errors"
)

type Config struct {
	Address   string `toml:"address"`
	Port      int    `toml:"port"`
	Advertise string `toml:"advertise"`
}

func (c *Config) Validate() error {
	switch {
	case c == nil:
		return errors.New("web config not set")
	case c.Address == "":
		return errors.New("web config http address must be set")
	case c.Port <= 1024:
		return errors.New("web config http port must be > 1024")
	case c.Advertise == "":
		return errors.New("web config http advertise must be set")
	default:
		return nil
	}
}
