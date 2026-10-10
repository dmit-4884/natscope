// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package mcptransport

import (
	"encoding/json"
)

// Body is a raw payload rendered for an agent: JSON when it parses as an object or array, else UTF-8 text, else base64.
type Body struct {
	JSON   json.RawMessage `json:"json,omitempty" jsonschema:"payload parsed as JSON"`
	Text   string          `json:"text,omitempty" jsonschema:"payload as UTF-8 text"`
	Base64 string          `json:"base64,omitempty" jsonschema:"binary payload, base64-encoded"`
}

// ConnectionArg is the `connection` argument shared by the tools that talk to NATS.
type ConnectionArg struct {
	Connection string `json:"connection,omitempty" jsonschema:"saved connection name or id; may be omitted when only one connection is saved"`
}
