// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
)

type typeView struct {
	FullName  string `json:"fullName"`
	ProtoFile string `json:"file"`
	SourceID  string `json:"sourceId"`
	SourceTag string `json:"sourceTag,omitempty"`
}

type listTypesInput struct {
	Filter string `json:"filter,omitempty" jsonschema:"case-insensitive substring of the full type name"`
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum types to return, 1-2000 (default 200)"`
}

type listTypesOutput struct {
	Types []typeView `json:"types"`
	Total int        `json:"total" jsonschema:"types matching the filter before the limit"`
}

type typeInput struct {
	Type     string `json:"type" jsonschema:"fully-qualified message type, e.g. orders.v1.OrderCreated"`
	SourceID string `json:"sourceId,omitempty" jsonschema:"proto source id; needed only when several sources define the type"`
}

type describeOutput struct {
	typeView
	Package string         `json:"package"`
	Fields  []fieldView    `json:"fields"`
	Example map[string]any `json:"example,omitempty" jsonschema:"example JSON payload for this type"`
}

type fieldView struct {
	Name      string `json:"name"`
	Number    int32  `json:"number"`
	Type      string `json:"type"`
	Label     string `json:"label,omitempty"`
	IsMessage bool   `json:"isMessage,omitempty" jsonschema:"the field is itself a message; describe its type for nested fields"`
}

type mappingView struct {
	Pattern           string  `json:"pattern"`
	MessageType       string  `json:"messageType"`
	SourceID          string  `json:"sourceId"`
	PinnedTag         *string `json:"pinnedTag,omitempty"`
	PinnedFingerprint *string `json:"pinnedFingerprint,omitempty"`
}

type subjectInput struct {
	Subject string `json:"subject" jsonschema:"concrete subject, e.g. orders.eu.created"`
}

type resolveOutput struct {
	Subject      string       `json:"subject"`
	Mapped       bool         `json:"mapped"`
	Mapping      *mappingView `json:"mapping,omitempty"`
	Health       string       `json:"health,omitempty" jsonschema:"ok, or why the mapping cannot decode right now"`
	HealthDetail string       `json:"healthDetail,omitempty"`
}

type listMappingsOutput struct {
	Mappings []mappingView `json:"mappings"`
}

type decodeInput struct {
	Base64   string `json:"base64" jsonschema:"payload bytes, base64-encoded"`
	Type     string `json:"type,omitempty" jsonschema:"fully-qualified message type to decode as"`
	SourceID string `json:"sourceId,omitempty" jsonschema:"proto source id; needed only when several sources define the type"`
	Subject  string `json:"subject,omitempty" jsonschema:"decode through this subject's mapping instead of an explicit type"`
}

type decodeOutput struct {
	MessageType string          `json:"messageType,omitempty"`
	Decoded     json.RawMessage `json:"decoded,omitempty"`
	Error       string          `json:"error,omitempty"`
}

type validateInput struct {
	Type     string         `json:"type" jsonschema:"fully-qualified message type"`
	SourceID string         `json:"sourceId,omitempty" jsonschema:"proto source id; needed only when several sources define the type"`
	Payload  map[string]any `json:"payload" jsonschema:"JSON object to check, in protojson form"`
}

type validateOutput struct {
	Valid      bool            `json:"valid"`
	Violations []violationView `json:"violations,omitempty"`
	Error      string          `json:"error,omitempty" jsonschema:"why the payload does not encode, when it does not"`
}

type violationView struct {
	FieldPath    string `json:"field"`
	Message      string `json:"message"`
	ConstraintId string `json:"constraint,omitempty"`
}

type statusOutput struct {
	MessageTypes int            `json:"messageTypes"`
	Error        string         `json:"error,omitempty"`
	Conflicts    []conflictView `json:"conflicts,omitempty"`
}

type conflictView struct {
	Kind     string        `json:"kind"`
	Severity string        `json:"severity"`
	Symbol   string        `json:"symbol"`
	Winner   schemaRefView `json:"winner"`
	Loser    schemaRefView `json:"loser"`
	Reason   string        `json:"reason,omitempty"`
}

type schemaRefView struct {
	SourceID string `json:"sourceId"`
	Tag      string `json:"tag,omitempty"`
	File     string `json:"file"`
}
