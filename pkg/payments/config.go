// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package payments

import (
	"errors"
)

type Config struct {
	SecretKey        string `toml:"secret_key"`
	PublishableKey   string `toml:"publishable_key"`
	WebhookSecret    string `toml:"webhook_secret"`
	PatreonProductID string `toml:"patreon_product_id"`
}

func (c *Config) Empty() bool {
	if c == nil {
		return true
	}
	secret := c.SecretKey == ""
	publishable := c.PublishableKey == ""
	webhook := c.WebhookSecret == ""
	patreon := c.PatreonProductID == ""
	return secret && publishable && webhook && patreon
}

func (c *Config) Validate() error {
	if c.Empty() {
		// the unset case is special, where we disable the patreon feature
		// e.g. for open source users running locally
		return nil
	}

	if c.SecretKey == "" {
		return errors.New("payments secret_key not set")
	}

	if c.PublishableKey == "" {
		return errors.New("payments publishable_key not set")
	}

	if c.WebhookSecret == "" {
		return errors.New("payments webhook_secret not set")
	}

	if c.PatreonProductID == "" {
		return errors.New("payments patreon_product_id not set")
	}

	return nil
}
