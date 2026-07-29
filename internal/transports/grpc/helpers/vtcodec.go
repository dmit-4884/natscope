// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package grpchelpers

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

// VTCodec is a Connect codec preferring vtprotobuf MarshalVT/UnmarshalVT/SizeVT,
// falling back to reflection-based proto for codegen-less types.
type VTCodec struct{}

// Name returns "proto", supplanting the default binary codec.
func (VTCodec) Name() string { return "proto" }

type vtMarshaler interface{ MarshalVT() ([]byte, error) }
type vtUnmarshaler interface{ UnmarshalVT([]byte) error }
type vtSizer interface{ SizeVT() int }

// Marshal encodes m via MarshalVT when available, else proto.Marshal.
func (VTCodec) Marshal(m any) ([]byte, error) {
	if vt, ok := m.(vtMarshaler); ok {
		return vt.MarshalVT()
	}
	pm, ok := m.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("vtcodec: %T is not proto.Message", m)
	}
	return proto.Marshal(pm)
}

// Unmarshal decodes data into m via UnmarshalVT when available, else
// proto.Unmarshal.
func (VTCodec) Unmarshal(data []byte, m any) error {
	if vt, ok := m.(vtUnmarshaler); ok {
		return vt.UnmarshalVT(data)
	}
	pm, ok := m.(proto.Message)
	if !ok {
		return fmt.Errorf("vtcodec: %T is not proto.Message", m)
	}
	return proto.Unmarshal(data, pm)
}

// Size reports m's wire size via SizeVT when available, else proto.Size.
func (VTCodec) Size(m any) int {
	if vt, ok := m.(vtSizer); ok {
		return vt.SizeVT()
	}
	pm, ok := m.(proto.Message)
	if !ok {
		return 0
	}
	return proto.Size(pm)
}
