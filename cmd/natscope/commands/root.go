// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"

	"github.com/altessa-s/go-atlas/core/runtime/appinfo"

	"github.com/dmit-4884/natscope/cmd/natscope/commands/server"
	"github.com/dmit-4884/natscope/cmd/natscope/commands/server/run"
	"github.com/dmit-4884/natscope/cmd/natscope/commands/version"
)

// Command is the root command, run when invoked without subcommands.
type Command struct {
	*cobra.Command
}

// New creates the root command.
func New() *cobra.Command {
	root := &Command{
		Command: &cobra.Command{
			Use:   "natscope",
			Short: "NATS JetStream message browser with protobuf support",
			Long: heredoc.Doc(`
				natscope browses NATS JetStream messages with protobuf decoding support.

				Runs as a single binary with an embedded React frontend.
				Invoked without a subcommand, it starts the server.
			`),
			Example: heredoc.Doc(`
				# Start the natscope server
				natscope

				# Start with a custom listen address
				GRPC_WEB_ADDRESS=127.0.0.1:9090 natscope

				# Show version information
				natscope version

				# Show detailed help for server commands
				natscope server --help
			`),
			Version: appinfo.Version,
		},
	}

	root.configure()

	return root.Command
}

// configure wires streams, suggestions, flags, and subcommands.
func (c *Command) configure() {
	c.SetErr(os.Stderr)
	c.SetOut(os.Stdout)

	c.PersistentFlags().StringP("config", "c", "", "path to config file")

	c.RunE = c.run

	c.SuggestionsMinimumDistance = 1
	c.SilenceUsage = true

	c.SetVersionTemplate(heredoc.Doc(`
		{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
	`))

	c.AddCommand(version.New())
	c.AddCommand(server.New())
}

// run starts the server, making a bare "natscope" equivalent to "natscope server run".
func (c *Command) run(cmd *cobra.Command, args []string) error {
	app := run.NewApp()
	return app.Run(cmd, args)
}

// Run executes the root command with explicit args, recovering from panics
// raised during command execution.
func Run(args []string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Panic recovered: %v\n", r)
			_, _ = fmt.Fprintf(os.Stderr, "Stack trace:\n%s\n", debug.Stack())
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	cmd := New()
	cmd.SetArgs(args)

	return cmd.Execute()
}
