// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package agent

import (
	"errors"
	"fmt"

	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/agent/web"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/crons"
	"cattlecloud.net/onlylogins/pkg/memcache"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/BurntSushi/toml"
)

func ConfigFrom(filepath string) (*Config, error) {
	cfg := new(Config)

	if _, err := toml.DecodeFile(filepath, cfg); err != nil {
		return nil, fmt.Errorf("unable to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

type Config struct {
	Runtime  *runtime.Config  `toml:"runtime"`
	Users    *users.Config    `toml:"users"`
	HTTP     *web.Config      `toml:"http"`
	Database *database.Config `toml:"database"`
	Cache    *memcache.Config `toml:"memcache"`
	Crons    *crons.Configs   `toml:"crons"`
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("agent config is not set")
	}
	if err := c.Runtime.Validate(); err != nil {
		return err
	}
	if err := c.Database.Validate(); err != nil {
		return err
	}
	if err := c.HTTP.Validate(); err != nil {
		return err
	}
	if err := c.Cache.Validate(); err != nil {
		return err
	}
	if err := c.Crons.Validate(); err != nil {
		return err
	}
	return nil
}
