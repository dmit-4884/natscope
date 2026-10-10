// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package protoutils

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ParseBinary parses wire bytes into m.
func (s *Schema) ParseBinary(data []byte, m proto.Message) error {
	return proto.UnmarshalOptions{Resolver: s.Types}.Unmarshal(data, m)
}

// RenderJSON renders m with proto field names and zero values included.
func (s *Schema) RenderJSON(m proto.Message) ([]byte, error) {
	return protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: true,
		Resolver:        s.DecodeResolver(),
	}.Marshal(m)
}

// ParseJSON parses JSON into m; unknown fields and enum names are errors.
func (s *Schema) ParseJSON(data []byte, m proto.Message) error {
	return protojson.UnmarshalOptions{Resolver: s.Types}.Unmarshal(data, m)
}

// EncodeBinary encodes m deterministically.
func EncodeBinary(m proto.Message) ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(m)
}
