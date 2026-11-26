// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package version

import (
	"runtime"
	"testing"

	"github.com/shoenig/test/must"
)

func TestBuild(t *testing.T) {
	t.Parallel()

	info := Build()

	must.Eq(t, Version, info.Version)
	must.Eq(t, runtime.Version(), info.GoVersion)
	must.Eq(t, runtime.GOOS, info.OS)
	must.Eq(t, runtime.GOARCH, info.Architecture)
}
