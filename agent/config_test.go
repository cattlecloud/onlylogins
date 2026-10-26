// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package agent

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestConfigFrom(t *testing.T) {
	t.Parallel()

	result, err := ConfigFrom("../hack/dev.toml")
	must.NoError(t, err)
	must.NotNil(t, result)
}
