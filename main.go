// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package main

import (
	"os"

	"cattlecloud.net/go/babycli"
	"cattlecloud.net/onlylogins/agent/commands"
)

func main() {
	args := babycli.Arguments()
	rc := commands.Invoke(args)
	os.Exit(rc)
}
