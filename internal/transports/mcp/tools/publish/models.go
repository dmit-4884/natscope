// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

type publishInput struct {
	mcptransport.ConnectionArg
	Subject  string            `json:"subject" jsonschema:"literal subject captured by a JetStream stream"`
	JSON     map[string]any    `json:"json,omitempty" jsonschema:"JSON object payload, in protojson form when it is encoded to Protobuf"`
	Text     string            `json:"text,omitempty" jsonschema:"raw text payload, sent as-is"`
	Type     string            `json:"type,omitempty" jsonschema:"fully-qualified Protobuf type to encode json as; defaults to the subject's mapping"`
	SourceID string            `json:"sourceId,omitempty" jsonschema:"proto source id; needed only when several sources define the type"`
	Raw      bool              `json:"raw,omitempty" jsonschema:"send json as JSON text even when the subject is mapped"`
	Headers  map[string]string `json:"headers,omitempty" jsonschema:"NATS message headers"`
}

type publishOutput struct {
	Stream      string `json:"stream"`
	Sequence    uint64 `json:"seq"`
	Duplicate   bool   `json:"duplicate,omitempty" jsonschema:"JetStream dropped it as a duplicate (Nats-Msg-Id)"`
	Encoding    string `json:"encoding" jsonschema:"protobuf or raw"`
	MessageType string `json:"messageType,omitempty"`
}
