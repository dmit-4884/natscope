// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package streams holds the MCP tools describing JetStream streams and their consumers.
package streams

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

const (
	defaultConsumersLimit = 200
	maxConsumersLimit     = 1000
)

// Toolset serves list_streams, get_stream, get_stream_relations and list_consumers.
type Toolset struct {
	conns   *mcptransport.Connections
	streams natssvc.StreamReader
	stats   natssvc.StatsReader
}

// New creates the streams toolset.
func New(conns *mcptransport.Connections, streams natssvc.StreamReader, stats natssvc.StatsReader) *Toolset {
	return &Toolset{conns: conns, streams: streams, stats: stats}
}

// Register adds the toolset's tools to s.
func (t *Toolset) Register(s *mcp.Server) {
	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "list_streams",
		Description: "List the JetStream streams of a connection with their subjects, retention, message count, size and sequence range.",
		Annotations: mcptransport.ReadOnly("List streams"),
	}, t.listStreams)

	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "get_stream",
		Description: "Show one stream's full configuration (limits, retention, mirror/sources, flags) and current state.",
		Annotations: mcptransport.ReadOnly("Get stream"),
	}, t.getStream)

	mcptransport.AddTool(s, &mcp.Tool{
		Name: "get_stream_relations",
		Description: "Show how streams feed each other: sources, mirrors and republish targets, with each link's filters, " +
			"subject transforms, lag, last activity and error. Pass `stream` for the links that touch it; omit it for all of them.",
		Annotations: mcptransport.ReadOnly("Get stream relations"),
	}, t.getStreamRelations)

	mcptransport.AddTool(s, &mcp.Tool{
		Name: "list_consumers",
		Description: "List consumers with their delivery progress: numPending (not yet delivered), numAckPending (delivered, unacked), " +
			"redeliveries and ack floor, most pending first. Omit `stream` to scan every stream, e.g. to find lagging consumers.",
		Annotations: mcptransport.ReadOnly("List consumers"),
	}, t.listConsumers)
}
