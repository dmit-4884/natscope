// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package grpchelpers

import (
	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

var wireTypes = map[entities.WireType]protopb.WireType{
	entities.WireVarint:  protopb.WireType_WIRE_TYPE_VARINT,
	entities.WireFixed64: protopb.WireType_WIRE_TYPE_FIXED64,
	entities.WireBytes:   protopb.WireType_WIRE_TYPE_BYTES,
	entities.WireGroup:   protopb.WireType_WIRE_TYPE_GROUP,
	entities.WireFixed32: protopb.WireType_WIRE_TYPE_FIXED32,
}

// WireTypeToProto converts a protobuf wire type for the wire.
func WireTypeToProto(t entities.WireType) protopb.WireType {
	return wireTypes[t]
}

// DecodeResultToProto converts a decode result for the wire; nil stays nil.
func DecodeResultToProto(r *entities.DecodeResult) *protopb.DecodeResult {
	if r == nil {
		return nil
	}
	pb := &protopb.DecodeResult{
		Data:        string(r.Decoded),
		MessageType: r.MessageType,
		ValidBytes:  int32(r.ValidBytes), //nolint:gosec // bounded by the payload size
		Auto:        r.Auto,
		SourceId:    r.SourceID,
		UnknownFields: slices.To(r.UnknownFields, func(f entities.UnknownField) *protopb.UnknownField {
			pb := converter.Convert(f, &protopb.UnknownField{}, converter.WithIgnoreFields("WireType"))
			pb.WireType = wireTypes[f.WireType]
			return pb
		}),
	}
	if r.Error != "" {
		pb.Error = &r.Error
	}
	return pb
}
