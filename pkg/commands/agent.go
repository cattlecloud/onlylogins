// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package commands

import (
	"os"
	"syscall"

	"cattlecloud.net/go/babycli"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/runtime/halt"
	"cattlecloud.net/onlylogins/agent"
)

func agentCmd() *babycli.Component {
	return &babycli.Component{
		Name: "agent",
		Help: "run the onlylogins service",
		Description: `
The agent is the long-running onlylogins service.
It reads a TOML configuration file, connects to the
database and cache, serves the web application over
HTTP, and runs scheduled background jobs. It runs in
the foreground and shuts down cleanly on SIGTERM.`,
		Flags: babycli.Flags{

			{
				Type:  babycli.StringFlag,
				Long:  "config",
				Short: "c",
				Help:  "path to TOML config file",
			},
		},
		Function: func(c *babycli.Component) babycli.Code {
			log := ulog.New("cli")

			// load the configuration file
			filename := c.GetString("config")
			config, cerr := agent.ConfigFrom(filename)
			if cerr != nil {
				log.E.Fmt("unable to startup due to configuration: %v", cerr)
				return babycli.Failure
			}

			// create a context for the life of the agent
			ctx, cancel := scope.WithCancel(c.Context())
			stop := func(s os.Signal) {
				log.W.Fmt("got signal: %v", s)
				cancel()
			}

			// capture SIGTERM and SIGINT
			halt.On(stop, syscall.SIGTERM, os.Interrupt)

			sb := agent.New(config)
			if err := sb.Start(ctx); err != nil {
				log.E.Fmt("unable to setup agent: %v", err)
				return babycli.Failure
			}

			return babycli.Success
		},
	}
}
