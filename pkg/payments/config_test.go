// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package payments

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		c := (*Config)(nil)
		err := c.Validate()
		must.NoError(t, err)
		must.Empty(t, c)
	})

	t.Run("missing secret_key", func(t *testing.T) {
		c := &Config{
			SecretKey:        "",
			PublishableKey:   "pk_test_123",
			WebhookSecret:    "whsec_test_123",
			PatreonProductID: "price_patreon",
		}
		err := c.Validate()
		must.ErrorContains(t, err, "payments secret_key not set")
	})

	t.Run("missing publishable_key", func(t *testing.T) {
		c := &Config{
			SecretKey:        "sk_test_123",
			PublishableKey:   "",
			WebhookSecret:    "whsec_test_123",
			PatreonProductID: "price_patreon",
		}
		err := c.Validate()
		must.ErrorContains(t, err, "payments publishable_key not set")
	})

	t.Run("missing webhook_secret", func(t *testing.T) {
		c := &Config{
			SecretKey:        "sk_test_123",
			PublishableKey:   "pk_test_123",
			WebhookSecret:    "",
			PatreonProductID: "price_patreon",
		}
		err := c.Validate()
		must.ErrorContains(t, err, "payments webhook_secret not set")
	})

	t.Run("missing patreon_product_id", func(t *testing.T) {
		c := &Config{
			SecretKey:        "sk_test_123",
			PublishableKey:   "pk_test_123",
			WebhookSecret:    "whsec_test_123",
			PatreonProductID: "",
		}
		err := c.Validate()
		must.ErrorContains(t, err, "payments patreon_product_id not set")
	})

	t.Run("valid", func(t *testing.T) {
		c := &Config{
			SecretKey:        "sk_test_123",
			PublishableKey:   "pk_test_123",
			WebhookSecret:    "whsec_test_123",
			PatreonProductID: "price_patreon",
		}
		err := c.Validate()
		must.NoError(t, err)
	})
}
