// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/encoding/protowire"
)

const maxWireDepth = 32

// DecodeWire reads a protobuf payload without a schema; valid counts the bytes that form whole fields.
func DecodeWire(data []byte) (fields []*entities.WireField, valid int, err error) {
	return decodeWire(data, 0, 0)
}

func decodeWire(data []byte, base, depth int) ([]*entities.WireField, int, error) {
	var fields []*entities.WireField
	offset := 0
	for offset < len(data) {
		f, err := decodeWireField(data[offset:], base+offset, depth)
		if err != nil {
			return fields, offset, err
		}
		fields = append(fields, f)
		offset += f.Length
	}
	return fields, offset, nil
}

func decodeWireField(data []byte, offset, depth int) (*entities.WireField, error) {
	num, typ, tagLen := protowire.ConsumeTag(data)
	if tagLen < 0 {
		return nil, fmt.Errorf("bad field tag at byte %d: %w", offset, protowire.ParseError(tagLen))
	}
	f := &entities.WireField{Number: int32(num), Offset: offset} //nolint:gosec // ConsumeTag caps numbers below 2^29
	rest := data[tagLen:]
	var n int
	switch typ {
	case protowire.VarintType:
		f.WireType = entities.WireVarint
		f.Varint, n = protowire.ConsumeVarint(rest)
	case protowire.Fixed32Type:
		var v uint32
		v, n = protowire.ConsumeFixed32(rest)
		f.WireType, f.Fixed = entities.WireFixed32, uint64(v)
	case protowire.Fixed64Type:
		f.WireType = entities.WireFixed64
		f.Fixed, n = protowire.ConsumeFixed64(rest)
	case protowire.BytesType:
		var v []byte
		v, n = protowire.ConsumeBytes(rest)
		f.WireType = entities.WireBytes
		if n >= 0 {
			describeBytes(f, v, offset+n-len(v)+tagLen, depth)
		}
	case protowire.StartGroupType:
		var v []byte
		v, n = protowire.ConsumeGroup(num, rest)
		f.WireType = entities.WireGroup
		if n >= 0 && depth < maxWireDepth {
			f.Message, _, _ = decodeWire(v, offset+tagLen, depth+1) //nolint:errcheck // ConsumeGroup already checked the group
		}
	default:
		return nil, fmt.Errorf("unsupported wire type %d at byte %d", typ, offset)
	}
	if n < 0 {
		return nil, fmt.Errorf("field %d at byte %d: %w", num, offset, protowire.ParseError(n))
	}
	if num < protowire.MinValidNumber {
		return nil, fmt.Errorf("invalid field number %d at byte %d", num, offset)
	}
	f.Length = tagLen + n
	return f, nil
}

func describeBytes(f *entities.WireField, v []byte, offset, depth int) {
	f.Bytes = v
	if utf8.Valid(v) && printable(string(v)) {
		f.Text = string(v)
	}
	if len(v) == 0 || depth >= maxWireDepth {
		return
	}
	if nested, valid, err := decodeWire(v, offset, depth+1); err == nil && valid == len(v) {
		f.Message = nested
	}
}

func printable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
