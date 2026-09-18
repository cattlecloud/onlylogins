package web

import (
	"errors"
	"time"
)

const Annual = 365 * 24 * time.Hour

type Oauth struct {
	Google struct {
		ClientID     string `toml:"client-id"`
		ClientSecret string `toml:"client-secret"`
		CallbackURI  string `toml:"callback-uri"`
	} `toml:"google"`
	Microsoft struct {
		Tenant       string `toml:"tenant"`
		ClientID     string `toml:"client-id"`
		ClientSecret string `toml:"client-secret"`
		CallbackURI  string `toml:"callback-uri"`
	} `toml:"microsoft"`
	Apple struct {
		TeamID         string `toml:"team-id"`
		ClientID       string `toml:"client-id"`
		KeyID          string `toml:"key-id"`
		PrivateKeyPath string `toml:"private-key-path"`
		CallbackURI    string `toml:"callback-uri"`
	} `toml:"apple"`
}

type Config struct {
	Address   string `toml:"address"`
	Port      int    `toml:"port"`
	Advertise string `toml:"advertise"`
	Oauth     Oauth  `toml:"oauth"`
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
