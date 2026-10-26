package crons

import (
	"errors"
	"fmt"

	"github.com/hashicorp/cronexpr"
)

type Configs struct {
	Snapshots  *Config `toml:"db-snapshots"`
	DailyDBM   *Config `toml:"db-daily"`
	WeeklyDBM  *Config `toml:"db-weekly"`
	MonthlyDBM *Config `toml:"db-monthly"`
}

func (c *Configs) Validate() error {
	if c == nil {
		return errors.New("crons config must be set")
	}

	if err := c.Snapshots.Validate(); err != nil {
		return err
	}

	if err := c.DailyDBM.Validate(); err != nil {
		return err
	}

	if err := c.WeeklyDBM.Validate(); err != nil {
		return err
	}

	if err := c.MonthlyDBM.Validate(); err != nil {
		return err
	}

	return nil
}

type Config struct {
	// supports common cron expressions
	//
	// https://en.wikipedia.org/wiki/Cron#CRON_expression
	Expression string `toml:"schedule"`
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("cron config must be set")
	}

	if _, err := cronexpr.Parse(c.Expression); err != nil {
		return fmt.Errorf("cron schedule expression: %w", err)
	}

	return nil
}
