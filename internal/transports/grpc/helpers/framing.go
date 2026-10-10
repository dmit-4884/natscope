// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package grpchelpers

import (
	"github.com/dmit-4884/natscope/internal/entities"

	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

var framingKinds = map[entities.FramingKind]protopb.FramingKind{
	entities.FramingGRPC:      protopb.FramingKind_FRAMING_KIND_GRPC,
	entities.FramingConfluent: protopb.FramingKind_FRAMING_KIND_CONFLUENT,
	entities.FramingDelimited: protopb.FramingKind_FRAMING_KIND_VARINT_DELIMITED,
	entities.FramingCustom:    protopb.FramingKind_FRAMING_KIND_CUSTOM,
}

// FramingToProto converts a framing for the wire; no framing becomes nil.
func FramingToProto(f entities.Framing) *protopb.Framing {
	kind, ok := framingKinds[f.Kind]
	if !ok {
		return nil
	}
	return &protopb.Framing{Kind: kind, SchemaId: f.SchemaID, Prefix: f.Prefix, Suffix: f.Suffix}
}

// FramingFromProto converts a wire framing; nil or unspecified means no framing.
func FramingFromProto(f *protopb.Framing) entities.Framing {
	for kind, pb := range framingKinds {
		if pb == f.GetKind() {
			return entities.Framing{Kind: kind, SchemaID: f.GetSchemaId(), Prefix: f.GetPrefix(), Suffix: f.GetSuffix()}
		}
	}
	return entities.Framing{}
}
