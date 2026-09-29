// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package schema holds the MCP tools over the Protobuf registry: message types, subject mappings, decoding and validation.
package schema

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	mappingssvc "github.com/dmit-4884/natscope/internal/services/mappings"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

const (
	defaultTypesLimit = 200
	maxTypesLimit     = 2000
)

// Toolset serves the schema tools.
type Toolset struct {
	registry protosvc.Registry
	codec    protosvc.Codec
	mappings mappingssvc.Service
}

// New creates the schema toolset.
func New(registry protosvc.Registry, codec protosvc.Codec, mappings mappingssvc.Service) *Toolset {
	return &Toolset{registry: registry, codec: codec, mappings: mappings}
}

// Register adds the toolset's tools to s.
func (t *Toolset) Register(s *mcp.Server) {
	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "list_message_types",
		Description: "List the Protobuf message types natscope has compiled from its proto sources, optionally filtered by name.",
		Annotations: mcptransport.ReadOnly("List message types"),
	}, t.listTypes)

	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "describe_message_type",
		Description: "Show a Protobuf message type's fields and an example JSON payload that encodes to it.",
		Annotations: mcptransport.ReadOnly("Describe message type"),
	}, t.describeType)

	mcptransport.AddTool(s, &mcp.Tool{
		Name: "resolve_subject",
		Description: "Explain how natscope decodes a subject: the matching subject mapping, its Protobuf type and source, " +
			"and whether that mapping currently resolves. Use it when payloads come back undecoded.",
		Annotations: mcptransport.ReadOnly("Resolve subject"),
	}, t.resolveSubject)

	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "list_mappings",
		Description: "List every subject mapping (subject pattern → Protobuf message type).",
		Annotations: mcptransport.ReadOnly("List mappings"),
	}, t.listMappings)

	mcptransport.AddTool(s, &mcp.Tool{
		Name: "decode_payload",
		Description: "Decode a base64 Protobuf payload to JSON, either as an explicit message type or through the mapping of a subject. " +
			"Handy for payloads found in logs or tests.",
		Annotations: mcptransport.ReadOnly("Decode payload"),
	}, t.decode)

	mcptransport.AddTool(s, &mcp.Tool{
		Name: "validate_payload",
		Description: "Check a JSON payload against a Protobuf message type: it must encode and pass the type's buf.validate rules. " +
			"Nothing is published.",
		Annotations: mcptransport.ReadOnly("Validate payload"),
	}, t.validate)

	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "get_schema_status",
		Description: "Report how many Protobuf message types are loaded, any load error, and conflicts between proto sources.",
		Annotations: mcptransport.ReadOnly("Get schema status"),
	}, t.status)
}
