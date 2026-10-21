// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestConfig_Valid(t *testing.T) {
	t.Parallel()

	t.Run("missing address", func(t *testing.T) {
		c := &Config{Port: 9999}
		err := c.Validate()
		must.EqError(t, err, "web config http address must be set")
	})

	t.Run("missing port", func(t *testing.T) {
		c := &Config{Address: "10.0.0.1"}
		err := c.Validate()
		must.EqError(t, err, "web config http port must be > 1024")
	})

	t.Run("missing advertise", func(t *testing.T) {
		c := &Config{
			Address: "10.0.0.1",
			Port:    3030,
		}
		err := c.Validate()
		must.EqError(t, err, "web config http advertise must be set")
	})

	t.Run("ok", func(t *testing.T) {
		c := &Config{
			Address:   "10.0.0.1",
			Port:      3030,
			Advertise: "http://localhost:8000",
		}
		err := c.Validate()
		must.NoError(t, err)
	})
}
