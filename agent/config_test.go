// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package agent

import (
	"testing"

	"cattlecloud.net/go/webtools/runtime"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"github.com/shoenig/test/must"
)

func TestConfigFrom(t *testing.T) {
	t.Parallel()

	result, err := ConfigFrom("../hack/dev.toml")
	must.NoError(t, err)
	must.NotNil(t, result)
}

func TestValidateCryptEnvironment(t *testing.T) {
	t.Parallel()

	local := &runtime.Config{Environment: "local"}
	staging := &runtime.Config{Environment: "staging"}
	prod := &runtime.Config{Environment: "production"}

	t.Run("local with passphrase", func(t *testing.T) {
		c := &Config{
			Runtime: local,
			Cypher:  &encrypt.Config{Passphrase: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
		}
		must.NoError(t, c.validateEnvironment())
	})

	t.Run("local with sealed", func(t *testing.T) {
		c := &Config{
			Runtime: local,
			Cypher:  &encrypt.Config{Sealed: "/etc/onlylogins/sealed"},
		}
		must.NoError(t, c.validateEnvironment())
	})

	t.Run("staging with passphrase", func(t *testing.T) {
		c := &Config{
			Runtime: staging,
			Cypher:  &encrypt.Config{Passphrase: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
		}
		must.Error(t, c.validateEnvironment())
	})

	t.Run("staging with sealed", func(t *testing.T) {
		c := &Config{
			Runtime: staging,
			Cypher:  &encrypt.Config{Sealed: "/etc/onlylogins/sealed"},
		}
		must.NoError(t, c.validateEnvironment())
	})

	t.Run("production with passphrase", func(t *testing.T) {
		c := &Config{
			Runtime: prod,
			Cypher:  &encrypt.Config{Passphrase: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
		}
		must.Error(t, c.validateEnvironment())
	})

	t.Run("production with sealed", func(t *testing.T) {
		c := &Config{
			Runtime: prod,
			Cypher:  &encrypt.Config{Sealed: "/etc/onlylogins/sealed"},
		}
		must.NoError(t, c.validateEnvironment())
	})

	t.Run("production with neither", func(t *testing.T) {
		c := &Config{
			Runtime: prod,
			Cypher:  &encrypt.Config{},
		}
		must.Error(t, c.validateEnvironment())
	})
}
