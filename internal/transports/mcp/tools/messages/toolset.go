// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package messages holds the MCP tools that read stream messages and tail live subjects, decoding Protobuf payloads.
package messages

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	livesvc "github.com/dmit-4884/natscope/internal/services/live"
	messagessvc "github.com/dmit-4884/natscope/internal/services/messages"
	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

const (
	defaultPageSize       = 20
	maxPageSize           = 100
	defaultPagePayload    = 4 << 10
	defaultMessagePayload = 64 << 10
	maxPayload            = 256 << 10
	responsePayloadBudget = 256 << 10
	minMessagePayload     = 512
	defaultTailSeconds    = 10
	maxTailSeconds        = 30
	defaultTailMessages   = 20
	maxTailMessages       = 200

	directionForward  = "forward"
	directionBackward = "backward"
)

// Toolset serves find_messages, get_message and tail_subject.
type Toolset struct {
	conns    *mcptransport.Connections
	messages messagessvc.Service
	live     livesvc.Service
}

// New creates the messages toolset.
func New(conns *mcptransport.Connections, messages messagessvc.Service, live livesvc.Service) *Toolset {
	return &Toolset{conns: conns, messages: messages, live: live}
}

// Register adds the toolset's tools to s.
func (t *Toolset) Register(s *mcp.Server) {
	mcptransport.AddTool(s, &mcp.Tool{
		Name: "find_messages",
		Description: "Read a page of messages stored in a JetStream stream, newest first by default. Payloads come back as `decoded` JSON " +
			"when the subject maps to a Protobuf type, otherwise as a raw `body`. Filter by subject (wildcards allowed), start at a " +
			"sequence or a point in time, and page with nextSeq. With contains or header the server searches the whole stream, " +
			"up to 100000 messages per call; pass nextSeq back as startSeq, with the same direction, to search further.",
		Annotations: mcptransport.ReadOnly("Find messages"),
	}, t.findMessages)

	mcptransport.AddTool(s, &mcp.Tool{
		Name:        "get_message",
		Description: "Fetch one stream message by sequence with its headers and full payload, decoded from Protobuf when the subject is mapped.",
		Annotations: mcptransport.ReadOnly("Get message"),
	}, t.getMessage)

	mcptransport.AddTool(s, &mcp.Tool{
		Name: "tail_subject",
		Description: "Listen to a subject for a few seconds and return the messages published meanwhile, decoded like find_messages. " +
			"Only new messages are seen: trigger the action you want to observe, then tail. Bind `stream` to read through JetStream.",
		Annotations: mcptransport.ReadOnly("Tail subject"),
	}, t.tailSubject)
}
