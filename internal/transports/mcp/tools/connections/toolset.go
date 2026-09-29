// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package connections holds the MCP tools describing saved connections and the NATS servers behind them.
package connections

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

// Toolset serves list_connections and get_server_info.
type Toolset struct {
	conns *mcptransport.Connections
	stats natssvc.StatsReader
}

// New creates the connections toolset.
func New(conns *mcptransport.Connections, stats natssvc.StatsReader) *Toolset {
	return &Toolset{conns: conns, stats: stats}
}

// Register adds the toolset's tools to s.
func (t *Toolset) Register(s *mcp.Server) {
	mcptransport.AddTool(s, &mcp.Tool{
		Name: "list_connections",
		Description: "List the NATS connections saved in natscope, with their server URLs and the result of the last connection test. " +
			"Credentials are never returned. Use a connection's name or id as the `connection` argument of other tools.",
		Annotations: mcptransport.ReadOnly("List connections"),
	}, t.listConnections)

	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "get_server_info",
		Description: "Describe the NATS server behind a connection: version, cluster, max payload, JetStream availability and account usage.",
		Annotations: mcptransport.ReadOnly("Get server info"),
	}, t.serverInfo)
}
