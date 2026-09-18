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

func Invoke(args []string) babycli.Code {
	return babycli.New(&babycli.Configuration{
		Arguments: args,
		Version:   "v0.0.0-dev",
		Top: &babycli.Component{
			Name:        "onlylogins",
			Help:        "run the onlylogins agent",
			Description: "a website of community chat",
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
		},
	}).Run()
}
