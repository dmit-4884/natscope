// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package mcpcmd implements "natscope mcp": a stdio MCP server that relays every tool call to the /mcp endpoint of a
// running natscope, for MCP clients that can only launch a command.
package mcpcmd

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/MakeNowJust/heredoc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/altessa-s/go-atlas/core/errors"
	"github.com/altessa-s/go-atlas/core/runtime/appinfo"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

// Command is the "mcp" command.
type Command struct {
	*cobra.Command
}

// New creates the "mcp" command.
func New() *cobra.Command {
	c := &Command{
		Command: &cobra.Command{
			Use:   "mcp",
			Short: "Serve natscope's MCP tools over stdio",
			Long: heredoc.Doc(`
				Serve natscope's MCP tools over stdio, relaying every call to the /mcp endpoint of a
				running natscope server. Use it for MCP clients that can only launch a command.

				Start natscope first: the bridge exits when the server is unreachable. Clients that
				speak Streamable HTTP can skip the bridge and use http://127.0.0.1:4280/mcp directly.
			`),
			Example: heredoc.Doc(`
				# MCP client config for a stdio-only client
				{"mcpServers": {"natscope": {"command": "natscope", "args": ["mcp"]}}}

				# Claude Code over HTTP, no bridge needed
				claude mcp add --transport http natscope http://127.0.0.1:4280/mcp

				# Bridge to a natscope on another address
				natscope mcp --url http://127.0.0.1:9090/mcp
			`),
			SilenceUsage: true,
		},
	}
	c.Flags().String("url", "", "MCP endpoint of the running natscope (default: http://<grpcWebAddress>/mcp from the config)")
	c.Args = cobra.NoArgs
	c.RunE = c.run
	return c.Command
}

func (c *Command) run(cmd *cobra.Command, _ []string) error {
	configFile, _ := cmd.Flags().GetString("config") //nolint:errcheck
	if configFile != "" {
		configFile = appinfo.ExpandPath(configFile)
	}
	if cf := appinfo.GetEnvVar("CONFIG_FILE"); cf != "" {
		configFile = cf
	}
	cfg, err := appconfig.Load(configFile, appconfig.LoggerDefaults(false))
	if err != nil {
		return errors.WrapOperation(err, "load config")
	}

	endpoint, _ := cmd.Flags().GetString("url") //nolint:errcheck
	if endpoint == "" {
		endpoint = endpointFor(cfg.GRPCWebAddress)
	}
	client := http.DefaultClient
	if cfg.WebAuth.Enabled() {
		client = &http.Client{Transport: basicAuth{user: cfg.WebAuth.Username, pass: cfg.WebAuth.Password, next: http.DefaultTransport}}
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return Bridge(ctx, endpoint, client, &mcp.StdioTransport{})
}

// Bridge serves the tools of the MCP endpoint at endpoint over downstream until ctx ends or the client disconnects.
func Bridge(ctx context.Context, endpoint string, client *http.Client, downstream mcp.Transport) error {
	upstream, err := mcp.NewClient(&mcp.Implementation{Name: "natscope-mcp-bridge", Version: appinfo.Version}, nil).
		Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: client, DisableStandaloneSSE: true}, nil)
	if err != nil {
		return errors.Wrapf(err, "natscope is not reachable at %s; start it first (natscope, or brew services start natscope)", endpoint)
	}
	defer upstream.Close() //nolint:errcheck

	var instructions string
	if res := upstream.InitializeResult(); res != nil {
		instructions = res.Instructions
	}
	srv := mcp.NewServer(
		&mcp.Implementation{Name: "natscope", Title: "Natscope", Version: appinfo.Version},
		&mcp.ServerOptions{Instructions: instructions, Capabilities: &mcp.ServerCapabilities{}},
	)
	for tool, listErr := range upstream.Tools(ctx, nil) {
		if listErr != nil {
			return errors.WrapOperation(listErr, "list natscope tools")
		}
		srv.AddTool(tool, relay(upstream, tool.Name))
	}
	return srv.Run(ctx, downstream)
}

func relay(upstream *mcp.ClientSession, name string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := &mcp.CallToolParams{Name: name}
		if len(req.Params.Arguments) > 0 {
			params.Arguments = req.Params.Arguments
		}
		res, err := upstream.CallTool(ctx, params)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "natscope did not answer: " + err.Error()}},
			}, nil
		}
		return res, nil
	}
}

func endpointFor(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host, port = "127.0.0.1", "4280"
	}
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + mcptransport.Path
}

type basicAuth struct {
	user, pass string
	next       http.RoundTripper
}

func (b basicAuth) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.SetBasicAuth(b.user, b.pass)
	return b.next.RoundTrip(r)
}
