// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// SchemaTypeKind is the kind of a named schema type.
type SchemaTypeKind string

const (
	SchemaTypeMessage SchemaTypeKind = "message"
	SchemaTypeEnum    SchemaTypeKind = "enum"
	SchemaTypeService SchemaTypeKind = "service"
)

// SchemaType summarizes a message, enum or service; MemberCount counts its fields, values or methods.
type SchemaType struct {
	FullName       string
	Kind           SchemaTypeKind
	File           string
	Package        string
	Comment        string
	MemberCount    int32
	Dependency     bool
	SourceID       string
	SourceRevision string
}

// TypeDescription holds a described type and, optionally, the types it reaches; the requested type comes first.
type TypeDescription struct {
	Messages []*SchemaMessage
	Enums    []*SchemaEnum
	Services []*SchemaService
}

// SchemaMessage describes a message type.
type SchemaMessage struct {
	FullName   string
	File       string
	Comment    string
	Deprecated bool
	Fields     []*SchemaField
}

// SchemaField describes a field; Kind and TypeName describe the element, or the value of a map.
type SchemaField struct {
	Name       string
	JSONName   string
	Number     int32
	Kind       string
	TypeName   string
	Repeated   bool
	MapKey     string
	Optional   bool
	Required   bool
	Oneof      string
	Deprecated bool
	Comment    string
}

// SchemaEnum describes an enum type.
type SchemaEnum struct {
	FullName   string
	File       string
	Comment    string
	Deprecated bool
	Values     []*SchemaEnumValue
}

// SchemaEnumValue describes an enum value.
type SchemaEnumValue struct {
	Name       string
	Number     int32
	Comment    string
	Deprecated bool
}

// SchemaService describes a service.
type SchemaService struct {
	FullName   string
	File       string
	Comment    string
	Deprecated bool
	Methods    []*SchemaMethod
}

// SchemaMethod describes a service method.
type SchemaMethod struct {
	Name            string
	InputType       string
	OutputType      string
	ClientStreaming bool
	ServerStreaming bool
	Comment         string
	Deprecated      bool
}
