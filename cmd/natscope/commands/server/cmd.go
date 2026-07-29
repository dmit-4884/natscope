// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"

	"github.com/dmit-4884/natscope/cmd/natscope/commands/server/run"
)

type Command struct {
	*cobra.Command
}

func New() *cobra.Command {
	server := &Command{
		Command: &cobra.Command{
			Use:   "server [command]",
			Short: "Commands to manage the natscope server",
			Long: heredoc.Doc(`
				Server commands provide control over the natscope service lifecycle.

				The server runs a web interface for browsing NATS JetStream messages
				with protobuf decoding support. It embeds a React frontend and provides
				a REST API for interacting with NATS streams.
			`),
			Example: heredoc.Doc(`
				# Start the server with default settings
				natscope server run

				# Start the server on a custom listen address
				GRPC_WEB_ADDRESS=127.0.0.1:9090 natscope server run

				# Show help for server subcommands
				natscope server --help

				# Show help for the run command
				natscope server run --help
			`),
			SilenceUsage: true,
		},
	}

	server.configure()
	return server.Command
}

func (c *Command) configure() {
	c.AddCommand(run.New())
}
