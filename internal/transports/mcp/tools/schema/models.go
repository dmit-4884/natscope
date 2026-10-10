// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

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
	Filter string `json:"filter,omitempty" jsonschema:"case-insensitive substring of the full type name" normalize:"trim,lowercase"`
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum types to return, 1-2000 (default 200)"`
}

type listTypesOutput struct {
	Types []typeView `json:"types"`
	Total int        `json:"total" jsonschema:"types matching the filter before the limit"`
}

type typeInput struct {
	Type     string `json:"type" jsonschema:"fully-qualified message type, e.g. orders.v1.OrderCreated" normalize:"trim"`
	SourceID string `json:"sourceId,omitempty" jsonschema:"proto source id; needed only when several sources define the type" normalize:"trim"`
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
	FramingKind       string  `json:"framing,omitempty" jsonschema:"wrapper around the protobuf message: grpc, confluent, varint_delimited or custom"`
}

type subjectInput struct {
	Subject string `json:"subject" jsonschema:"concrete subject, e.g. orders.eu.created" normalize:"trim"`
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
	Base64   string `json:"base64" jsonschema:"payload bytes, base64-encoded" normalize:"trim"`
	Type     string `json:"type,omitempty" jsonschema:"fully-qualified message type to decode as; omit with subject to dump wire fields" normalize:"trim"`
	SourceID string `json:"sourceId,omitempty" jsonschema:"proto source id; needed only when several sources define the type" normalize:"trim"`
	Subject  string `json:"subject,omitempty" jsonschema:"decode through this subject's mapping instead of an explicit type" normalize:"trim"`
}

type decodeOutput struct {
	MessageType   string             `json:"messageType,omitempty"`
	Decoded       json.RawMessage    `json:"decoded,omitempty"`
	Error         string             `json:"error,omitempty"`
	ValidBytes    int                `json:"validBytes,omitempty" jsonschema:"set when decoding failed but the first validBytes decoded into decoded"`
	UnknownFields []unknownFieldView `json:"unknownFields,omitempty" jsonschema:"fields the schema does not declare; the producer may use a newer schema"`
	Wire          []wireFieldView    `json:"wire,omitempty" jsonschema:"schemaless dump of the payload, when neither type nor subject is given"`
}

type detectInput struct {
	Base64   string `json:"base64" jsonschema:"payload bytes, base64-encoded" normalize:"trim"`
	SourceID string `json:"sourceId,omitempty" jsonschema:"proto source id to search; every enabled source when omitted" normalize:"trim"`
	Limit    int    `json:"limit,omitempty" jsonschema:"candidates to return, 1 to 50; 5 when omitted"`
}

type detectOutput struct {
	Candidates []candidateView `json:"candidates"`
}

type candidateView struct {
	SourceID       string          `json:"sourceId"`
	SourceRevision string          `json:"sourceRevision,omitempty"`
	MessageType    string          `json:"messageType"`
	Score          int             `json:"score" jsonschema:"0 to 100: mostly the share of bytes that decode as declared fields"`
	UnknownBytes   int             `json:"unknownBytes,omitempty" jsonschema:"bytes of fields the type does not declare"`
	Decoded        json.RawMessage `json:"decoded"`
}

type unknownFieldView struct {
	Path     string `json:"path,omitempty" jsonschema:"message holding the field, empty for the top level"`
	Number   int32  `json:"number"`
	WireType string `json:"wireType"`
	Size     int    `json:"size"`
}

type wireFieldView struct {
	Number   int32           `json:"number"`
	WireType string          `json:"wireType"`
	Varint   uint64          `json:"varint,omitempty"`
	Fixed    uint64          `json:"fixed,omitempty" jsonschema:"raw fixed32 or fixed64 bits"`
	Text     string          `json:"text,omitempty"`
	Bytes    []byte          `json:"bytes,omitempty" jsonschema:"base64, only when the bytes are neither text nor a message"`
	Message  json.RawMessage `json:"message,omitempty" jsonschema:"fields of bytes that parse as a message, shaped like wire"`
}

type validateInput struct {
	Type     string         `json:"type" jsonschema:"fully-qualified message type" normalize:"trim"`
	SourceID string         `json:"sourceId,omitempty" jsonschema:"proto source id; needed only when several sources define the type" normalize:"trim"`
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
	Conflicts    []conflictView `json:"conflicts,omitempty" jsonschema:"clashes between enabled proto sources"`
}

type conflictView struct {
	Kind     string        `json:"kind"`
	Severity string        `json:"severity"`
	Symbol   string        `json:"symbol"`
	First    schemaRefView `json:"first"`
	Second   schemaRefView `json:"second"`
	Reason   string        `json:"reason,omitempty"`
}

type schemaRefView struct {
	SourceID string `json:"sourceId"`
	Revision string `json:"revision,omitempty"`
	File     string `json:"file"`
}
