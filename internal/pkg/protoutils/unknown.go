// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package protoutils

import (
	"fmt"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var wireTypes = map[protowire.Type]entities.WireType{
	protowire.VarintType:     entities.WireVarint,
	protowire.Fixed64Type:    entities.WireFixed64,
	protowire.BytesType:      entities.WireBytes,
	protowire.StartGroupType: entities.WireGroup,
	protowire.Fixed32Type:    entities.WireFixed32,
}

// UnknownFields lists the fields of a decoded message its schema does not declare, nested messages included.
func UnknownFields(m protoreflect.Message) []entities.UnknownField {
	var out []entities.UnknownField
	collectUnknown(m, "", &out)
	return out
}

func collectUnknown(m protoreflect.Message, path string, out *[]entities.UnknownField) {
	for raw := m.GetUnknown(); len(raw) > 0; {
		num, typ, n := protowire.ConsumeField(raw)
		if n < 0 {
			break
		}
		*out = append(*out, entities.UnknownField{Path: path, Number: int32(num), WireType: wireTypes[typ], Size: n}) //nolint:gosec // field numbers fit int32
		raw = raw[n:]
	}
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		name := string(fd.Name())
		if path != "" {
			name = path + "." + name
		}
		switch {
		case fd.IsMap():
			if fd.MapValue().Message() != nil {
				v.Map().Range(func(k protoreflect.MapKey, mv protoreflect.Value) bool {
					collectUnknown(mv.Message(), fmt.Sprintf("%s[%s]", name, k.String()), out)
					return true
				})
			}
		case fd.IsList():
			if fd.Message() != nil {
				for i := range v.List().Len() {
					collectUnknown(v.List().Get(i).Message(), fmt.Sprintf("%s[%d]", name, i), out)
				}
			}
		case fd.Message() != nil:
			collectUnknown(v.Message(), name, out)
		}
		return true
	})
}
