// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package format

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestPageTitle(t *testing.T) {
	t.Parallel()

	result := PageTitle("Share")
	must.Eq(t, "Share | OnlyLogins", result)
}
