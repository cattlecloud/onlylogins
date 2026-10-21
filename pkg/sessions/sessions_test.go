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
			return time.Date(2026, 2, 2, 9, 20, 0, 0, time.UTC)
		},
	}

	t.Run("normal", func(t *testing.T) {
		insertion := time.Date(2026, 2, 1, 9, 20, 0, 0, time.UTC)
		result := s.remainingTTL(insertion)
		must.Eq(t, result, 364*24*time.Hour) // 1 year - 1 day
	})

	t.Run("grace", func(t *testing.T) {
		insertion := time.Date(2025, 2, 2, 8, 0, 0, 0, time.UTC)
		result := s.remainingTTL(insertion)
		must.Eq(t, result, 3*time.Hour) // grace period
	})
}
