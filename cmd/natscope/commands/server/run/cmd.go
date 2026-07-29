// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

type Command struct {
	*cobra.Command
}

func New() *cobra.Command {
	run := &Command{
		Command: &cobra.Command{
			Use:   "run [flags]",
			Short: "Start and run the natscope server",
			Long: heredoc.Doc(`
				Start the natscope server.

				Reads config from ~/.natscope/ (or --config), then serves the browser UI
				and Connect-RPC API on the configured address. Stops on SIGTERM or SIGINT.
			`),
			Example: heredoc.Doc(`
				# Start the server with default settings (http://127.0.0.1:4280)
				natscope server run

				# Start the server on a custom port
				GRPC_WEB_ADDRESS=127.0.0.1:9090 natscope server run

				# Expose beyond loopback (requires ALLOW_REMOTE; add basic auth)
				GRPC_WEB_ADDRESS=0.0.0.0:4280 ALLOW_REMOTE=true \
					WEB_AUTH__USERNAME=admin WEB_AUTH__PASSWORD=secret natscope server run

				# Start with an explicit config file
				natscope server run --config /path/to/config.yaml
			`),
			SilenceUsage: true,
		},
	}

	run.configure()
	return run.Command
}

func (c *Command) configure() {
	c.RunE = c.run
}

func (c *Command) run(cmd *cobra.Command, args []string) error {
	app := NewApp()
	return app.Run(cmd, args)
}
