// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package publish holds publish_message and request_message, the MCP write tools, registered only when mcp.allowWrites is on.
package publish

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"

	mappingssvc "github.com/dmit-4884/natscope/internal/services/mappings"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	publishsvc "github.com/dmit-4884/natscope/internal/services/publish"
	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

const (
	encodingProtobuf = "protobuf"
	encodingRaw      = "raw"

	maxRequestTimeoutMs = 60_000
	replyPayloadBudget  = 64 << 10
)

// Toolset serves publish_message and request_message.
type Toolset struct {
	enabled  bool
	conns    *mcptransport.Connections
	publish  publishsvc.Service
	mappings mappingssvc.Service
	registry protosvc.Registry
}

// New creates the publish toolset; it registers nothing unless mcp.allowWrites is on.
func New(
	cfg *appconfig.Config,
	conns *mcptransport.Connections,
	publish publishsvc.Service,
	mappings mappingssvc.Service,
	registry protosvc.Registry,
) *Toolset {
	return &Toolset{enabled: cfg.MCPAllowWrites(), conns: conns, publish: publish, mappings: mappings, registry: registry}
}

// Register adds publish_message and request_message to s when writes are allowed.
func (t *Toolset) Register(s *mcp.Server) {
	if !t.enabled {
		return
	}
	mcptransport.AddTool(s, &mcp.Tool{
		Name: "publish_message",
		Description: "Publish a message to a JetStream subject. A `json` payload is encoded to Protobuf when `type` is given or the " +
			"subject is mapped (unless raw is set); otherwise it is sent as JSON text. Use `text` for any other raw payload. " +
			"Check the payload with validate_payload first. The publish is recorded in the natscope publish history.",
		Annotations: &mcp.ToolAnnotations{Title: "Publish message", DestructiveHint: new(false)},
	}, t.publishMessage)

	mcptransport.AddTool(s, &mcp.Tool{
		Name: "request_message",
		Description: "Send a core NATS request to a service subject and return the first reply. The payload works like publish_message; " +
			"omit json and text for an empty request. Fails at once with NATS_NO_RESPONDERS when nothing listens on the subject and " +
			"with NATS_TIMEOUT when no reply arrives within timeoutMs. Decode a Protobuf reply body with decode_payload. " +
			"Requests are not recorded in the publish history.",
		Annotations: &mcp.ToolAnnotations{Title: "Send request", DestructiveHint: new(false)},
	}, t.requestMessage)
}
