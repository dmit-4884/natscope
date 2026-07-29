// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package protoutils provides utility functions for working with protocol buffer descriptors.
package protoutils

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// Template generates a template/stub with default values for a message.
func Template(md protoreflect.MessageDescriptor, visited map[string]bool) map[string]any {
	fullName := string(md.FullName())

	// Guard against circular references.
	if visited[fullName] {
		return map[string]any{}
	}
	visited[fullName] = true
	defer delete(visited, fullName)

	result := make(map[string]any)

	fields := md.Fields()
	for i := range fields.Len() {
		fd := fields.Get(i)
		fieldName := string(fd.Name())
		value := fieldTemplate(fd, visited)

		// Repeated fields (but not maps).
		if fd.IsList() && !fd.IsMap() {
			result[fieldName] = []any{value}
		} else {
			result[fieldName] = value
		}
	}

	return result
}

// fieldTemplate generates a template value for a single field.
func fieldTemplate(fd protoreflect.FieldDescriptor, visited map[string]bool) any {
	// Map fields.
	if fd.IsMap() {
		keyVal := fieldTemplate(fd.MapKey(), visited)
		valueVal := fieldTemplate(fd.MapValue(), visited)
		return map[string]any{
			fmt.Sprintf("%v", keyVal): valueVal,
		}
	}

	// Message types.
	if fd.Kind() == protoreflect.MessageKind {
		return Template(fd.Message(), visited)
	}

	// Enum types.
	if fd.Kind() == protoreflect.EnumKind {
		values := fd.Enum().Values()
		if values.Len() > 0 {
			return string(values.Get(0).Name())
		}
		return "UNKNOWN"
	}

	// Handle scalar types
	switch fd.Kind() {
	case protoreflect.StringKind:
		return ""
	case protoreflect.BytesKind:
		return ""
	case protoreflect.BoolKind:
		return false
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return 0.0
	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return 0
	default:
		return nil
	}
}
