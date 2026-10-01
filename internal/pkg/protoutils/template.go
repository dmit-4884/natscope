// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package protoutils provides utility functions for working with protocol buffer descriptors.
package protoutils

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

const floatZero = 0.0

var wellKnownExamples = map[protoreflect.FullName]func() any{
	"google.protobuf.Timestamp":   func() any { return "1970-01-01T00:00:00Z" },
	"google.protobuf.Duration":    func() any { return "0s" },
	"google.protobuf.FieldMask":   func() any { return "" },
	"google.protobuf.Struct":      func() any { return map[string]any{} },
	"google.protobuf.Value":       func() any { return nil },
	"google.protobuf.ListValue":   func() any { return []any{} },
	"google.protobuf.Any":         func() any { return map[string]any{} },
	"google.protobuf.Empty":       func() any { return map[string]any{} },
	"google.protobuf.BoolValue":   func() any { return false },
	"google.protobuf.StringValue": func() any { return "" },
	"google.protobuf.BytesValue":  func() any { return "" },
	"google.protobuf.DoubleValue": func() any { return floatZero },
	"google.protobuf.FloatValue":  func() any { return floatZero },
	"google.protobuf.Int32Value":  func() any { return 0 },
	"google.protobuf.UInt32Value": func() any { return 0 },
	"google.protobuf.Int64Value":  func() any { return "0" },
	"google.protobuf.UInt64Value": func() any { return "0" },
}

// Template builds an example JSON value for md that protojson accepts.
func Template(md protoreflect.MessageDescriptor) any {
	return messageTemplate(md, make(map[protoreflect.FullName]bool))
}

func messageTemplate(md protoreflect.MessageDescriptor, visited map[protoreflect.FullName]bool) any {
	if wk, ok := wellKnownExamples[md.FullName()]; ok {
		return wk()
	}
	if visited[md.FullName()] {
		return map[string]any{}
	}
	visited[md.FullName()] = true
	defer delete(visited, md.FullName())

	result := make(map[string]any)
	fields := md.Fields()
	for i := range fields.Len() {
		fd := fields.Get(i)
		if oneof := fd.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() && oneof.Fields().Get(0) != fd {
			continue
		}
		value := fieldTemplate(fd, visited)
		if fd.IsList() {
			value = []any{value}
		}
		result[string(fd.Name())] = value
	}
	return result
}

func fieldTemplate(fd protoreflect.FieldDescriptor, visited map[protoreflect.FullName]bool) any {
	if fd.IsMap() {
		return map[string]any{fmt.Sprint(scalarTemplate(fd.MapKey())): fieldTemplate(fd.MapValue(), visited)}
	}
	switch fd.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return messageTemplate(fd.Message(), visited)
	case protoreflect.EnumKind:
		if values := fd.Enum().Values(); values.Len() > 0 {
			return string(values.Get(0).Name())
		}
		return 0
	default:
		return scalarTemplate(fd)
	}
}

func scalarTemplate(fd protoreflect.FieldDescriptor) any {
	switch fd.Kind() {
	case protoreflect.StringKind, protoreflect.BytesKind:
		return ""
	case protoreflect.BoolKind:
		return false
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return floatZero
	case protoreflect.Int64Kind, protoreflect.Uint64Kind, protoreflect.Sint64Kind,
		protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind:
		return "0"
	default:
		return 0
	}
}
