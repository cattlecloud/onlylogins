// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package sessions

import (
	"testing"
	"time"

	"github.com/shoenig/test/must"
)

func Test_remainingTTL(t *testing.T) {
	t.Parallel()

	s := &Store{
		now: func() time.Time {
			return time.Date(2026, 2, 2, 9, 30, 0, 0, time.UTC)
		},
	}

	t.Run("normal", func(t *testing.T) {
		insertion := time.Date(2026, 2, 2, 9, 29, 0, 0, time.UTC)
		result := s.remainingTTL(insertion)
		must.Eq(t, result, 9*time.Minute) // insert was 1 minute ago
	})

	t.Run("grace", func(t *testing.T) {
		insertion := time.Date(2025, 2, 2, 9, 30, 0, 0, time.UTC)
		result := s.remainingTTL(insertion)
		must.Eq(t, result, 1*time.Minute) // grace period
	})
}
