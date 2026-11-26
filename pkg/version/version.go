// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package version

import "runtime"

// Version is the onlylogins release version. It is overridden at build time via
// -ldflags "-X cattlecloud.net/onlylogins/pkg/version.Version=<tag>".
var Version = "dirty"

// Info describes the compiled application.
type Info struct {
	Version      string // release version string
	GoVersion    string // Go toolchain used to build
	OS           string // operating system the binary was built for
	Architecture string // target CPU architecture
}

// Build reports information about the compiled application.
func Build() Info {
	return Info{
		Version:      Version,
		GoVersion:    runtime.Version(),
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
}
