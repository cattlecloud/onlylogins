// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package users

import (
	"testing"

	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

func TestValidatePassphrase(t *testing.T) {
	t.Parallel()

	t.Run("ValidRandom", func(t *testing.T) {
		// 24 distinct code points, no adjacent deltas in {-1, 0, 1},
		// H = log2(24) * 24 ~= 110 bits of entropy (well above the 80 floor).
		p := conceal.New("aB3!cD4#eF5$gH6&iJ7*kL8@")
		err := ValidatePassphrase(p)
		must.NoError(t, err)
	})

	t.Run("TooShort", func(t *testing.T) {
		p := conceal.New("abc123")
		err := ValidatePassphrase(p)
		must.Error(t, err)
	})

	t.Run("LowEntropy", func(t *testing.T) {
		// 24 runes, only 4 distinct alphabet letters (A, ~, q, 1) repeating;
		// deltas are large so no sequence run, but Bits = 48 (below 80).
		p := conceal.New("A~q1A~q1A~q1A~q1A~q1A~q1")
		err := ValidatePassphrase(p)
		must.Error(t, err)
	})

	t.Run("SequenceRun", func(t *testing.T) {
		// 24 runes, embeds "qrstuv" (5 ascending runes) inside a high-entropy
		// string; length and entropy are fine, but the sequence is rejected.
		p := conceal.New("K3$M9pL7v!qrstuv!8B@XzA9")
		err := ValidatePassphrase(p)
		must.Error(t, err)
	})

	t.Run("Nil", func(t *testing.T) {
		var p *conceal.Text
		err := ValidatePassphrase(p)
		must.Error(t, err)
	})
}

func TestProfile_Active(t *testing.T) {
	t.Parallel()

	t.Run("yes", func(t *testing.T) {
		p := &Profile{
			ID: 1,
		}
		must.True(t, p.Active())
	})

	t.Run("no", func(t *testing.T) {
		p := (*Profile)(nil)
		must.False(t, p.Active())
	})
}

func TestProfile_Identity(t *testing.T) {
	t.Parallel()

	t.Run("active", func(t *testing.T) {
		p := &Profile{
			ID: 3,
		}
		must.Eq(t, 3, p.Identity().(Identity))
	})

	t.Run("inactive", func(t *testing.T) {
		p := (*Profile)(nil)
		must.Eq(t, Inactive, p.Identity())
	})
}
