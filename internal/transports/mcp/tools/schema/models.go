// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
)

type typeView struct {
	FullName       string `json:"fullName"`
	File           string `json:"file"`
	Comment        string `json:"comment,omitempty"`
	SourceID       string `json:"sourceId"`
	SourceRevision string `json:"sourceRevision,omitempty"`
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
	Package string        `json:"package"`
	Fields  []fieldView   `json:"fields"`
	Related []messageView `json:"relatedMessages,omitempty" jsonschema:"messages the type reaches through its fields"`
	Enums   []enumView    `json:"enums,omitempty" jsonschema:"enums the type reaches through its fields"`
	Example any           `json:"example,omitempty" jsonschema:"example JSON payload for this type"`
}

type messageView struct {
	FullName string      `json:"fullName"`
	Comment  string      `json:"comment,omitempty"`
	Fields   []fieldView `json:"fields"`
}

type fieldView struct {
	Name       string `json:"name"`
	JSONName   string `json:"jsonName"`
	Number     int32  `json:"number"`
	Kind       string `json:"kind" jsonschema:"protobuf kind of the element or map value: string, int64, bool, message, enum..."`
	TypeName   string `json:"typeName,omitempty" jsonschema:"message or enum full name, described in relatedMessages or enums"`
	Repeated   bool   `json:"repeated,omitempty"`
	MapKey     string `json:"mapKey,omitempty" jsonschema:"key kind of a map field"`
	Optional   bool   `json:"optional,omitempty"`
	Required   bool   `json:"required,omitempty"`
	Oneof      string `json:"oneof,omitempty" jsonschema:"oneof group; set at most one field of a group"`
	Deprecated bool   `json:"deprecated,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

type enumView struct {
	FullName string          `json:"fullName"`
	Comment  string          `json:"comment,omitempty"`
	Values   []enumValueView `json:"values"`
}

type enumValueView struct {
	Name    string `json:"name"`
	Number  int32  `json:"number"`
	Comment string `json:"comment,omitempty"`
}

type mappingView struct {
	Pattern           string  `json:"pattern"`
	MessageType       string  `json:"messageType"`
	SourceID          string  `json:"sourceId"`
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
	Revision string `json:"revision,omitempty"`
	File     string `json:"file"`
}
