// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package version

// Version is the onlylogins release version. It is overridden at build time via
// -ldflags "-X cattlecloud.net/onlylogins/pkg/version.Version=<tag>".
var Version = "dirty"
