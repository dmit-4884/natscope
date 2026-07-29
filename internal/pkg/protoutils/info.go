// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// Info extracts message info from a descriptor.
func Info(md protoreflect.MessageDescriptor) entities.ProtoMessageInfo {
	fields := make([]*entities.ProtoField, 0, md.Fields().Len())

	fds := md.Fields()
	for i := range fds.Len() {
		fd := fds.Get(i)
		field := &entities.ProtoField{
			Name:   string(fd.Name()),
			Number: int32(fd.Number()),
			Type:   fd.Kind().String(),
			Label:  cardinalityToLabel(fd.Cardinality()),
		}

		if fd.Kind() == protoreflect.MessageKind {
			field.IsMessage = true
			field.Type = string(fd.Message().FullName())
		}

		fields = append(fields, field)
	}

	return entities.ProtoMessageInfo{
		FullName:  string(md.FullName()),
		ProtoFile: md.ParentFile().Path(),
		Package:   string(md.ParentFile().Package()),
		Fields:    fields,
	}
}

// cardinalityToLabel maps a Cardinality to its label string.
func cardinalityToLabel(c protoreflect.Cardinality) string {
	switch c {
	case protoreflect.Optional:
		return "LABEL_OPTIONAL"
	case protoreflect.Required:
		return "LABEL_REQUIRED"
	case protoreflect.Repeated:
		return "LABEL_REPEATED"
	default:
		return "LABEL_OPTIONAL"
	}
}
