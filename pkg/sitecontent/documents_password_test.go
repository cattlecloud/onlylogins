// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package sitecontent

import (
	"strings"
	"testing"

	"github.com/shoenig/test/must"
)

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	t.Run("empty is rejected", func(t *testing.T) {
		err := ValidatePassword("")
		must.Error(t, err)
	})

	t.Run("whitespace only is accepted", func(t *testing.T) {
		// Whitespace can legitimately be part of a password secret and
		// must not be dropped silently — a literal whitespace-only
		// password is therefore a valid (if unlikely) value.
		err := ValidatePassword("   ")
		must.NoError(t, err)
	})

	t.Run("normal value is accepted", func(t *testing.T) {
		err := ValidatePassword("s3cret!")
		must.NoError(t, err)
	})

	t.Run("max length is accepted", func(t *testing.T) {
		err := ValidatePassword(strings.Repeat("a", maxPasswordLength))
		must.NoError(t, err)
	})

	t.Run("over max length is rejected", func(t *testing.T) {
		err := ValidatePassword(strings.Repeat("a", maxPasswordLength+1))
		must.Error(t, err)
	})
}
