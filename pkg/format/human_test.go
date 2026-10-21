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
