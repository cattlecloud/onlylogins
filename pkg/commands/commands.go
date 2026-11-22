// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package commands

import (
	"cattlecloud.net/go/babycli"
	"cattlecloud.net/onlylogins/version"
)

func Invoke(args []string) babycli.Code {
	return babycli.New(&babycli.Configuration{
		Arguments: args,
		Version:   version.Version,
		Top: &babycli.Component{
			Name: "onlylogins",
			Help: "self-hosted password manager",
			Description: `
onlylogins is an open source password manager
designed to be simple to operate on your own
private infrastructure, in the cloud, colocation,
or a homelab.
	https://onlylogins.net`,
			Components: babycli.Components{
				agentCmd(),
				tpmCmd(),
			},
		},
	}).Run()
}
