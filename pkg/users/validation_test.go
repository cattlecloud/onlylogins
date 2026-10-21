// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package users

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestAllowUsername(t *testing.T) {
	t.Parallel()

	t.Run("FooBar", func(t *testing.T) {
		ok := AllowUsername("FooBar")
		must.True(t, ok)
	})

	t.Run("admin", func(t *testing.T) {
		ok := AllowUsername("admin")
		must.False(t, ok)
	})

	t.Run("AdmIn", func(t *testing.T) {
		ok := AllowUsername("AdmIn")
		must.False(t, ok)
	})

	t.Run("4abc", func(t *testing.T) {
		ok := AllowUsername("4abc")
		must.False(t, ok)
	})

	t.Run("abc4", func(t *testing.T) {
		ok := AllowUsername("abc4")
		must.True(t, ok)
	})
}
