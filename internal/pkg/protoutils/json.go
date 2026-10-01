// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// UnmarshalBinary parses wire bytes into m.
func (s *Schema) UnmarshalBinary(data []byte, m proto.Message) error {
	return proto.UnmarshalOptions{Resolver: s.Types}.Unmarshal(data, m)
}

// MarshalJSON renders m with proto field names and zero values included.
func (s *Schema) MarshalJSON(m proto.Message) ([]byte, error) {
	return protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: true,
		Resolver:        s.DecodeResolver(),
	}.Marshal(m)
}

// UnmarshalJSON parses JSON into m; unknown fields and enum names are errors.
func (s *Schema) UnmarshalJSON(data []byte, m proto.Message) error {
	return protojson.UnmarshalOptions{Resolver: s.Types}.Unmarshal(data, m)
}

// MarshalBinary encodes m deterministically.
func MarshalBinary(m proto.Message) ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(m)
}
