// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package format

import (
	"testing"
	"time"

	"github.com/shoenig/test/must"
)

func TestAgo(t *testing.T) {
	t.Parallel()

	fma := time.Now().Add(-5 * time.Minute)
	result := Ago(fma)
	must.Eq(t, "5 minutes ago", result)
}

func TestDateTime(t *testing.T) {
	t.Parallel()

	// a fixed reference time far enough into the future that timezone
	// abbreviations shift but the textual format remains stable.
	t0 := time.Date(2026, time.November, 2, 13, 4, 0, 0, time.UTC)
	result := DateTime(t0)
	must.Eq(t, "Monday 02, Nov 2026 13:04 UTC", result)
}
