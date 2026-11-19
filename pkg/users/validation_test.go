// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package users

import (
	"strings"
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

func TestValidateEmail(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		must.NoError(t, ValidateEmail("alice@example.com"))
	})

	t.Run("too short", func(t *testing.T) {
		must.Error(t, ValidateEmail("a@"))
	})

	t.Run("missing at", func(t *testing.T) {
		must.Error(t, ValidateEmail("alice.example.com"))
	})

	t.Run("too long", func(t *testing.T) {
		long := strings.Repeat("a", 128)
		must.Error(t, ValidateEmail(long))
	})
}
