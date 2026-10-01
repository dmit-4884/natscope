// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// FramingKind is how a payload wraps its protobuf message.
type FramingKind string

const (
	FramingNone      FramingKind = ""
	FramingGRPC      FramingKind = "grpc"
	FramingConfluent FramingKind = "confluent"
	FramingDelimited FramingKind = "varint_delimited"
	FramingCustom    FramingKind = "custom"
)

// Framing wraps the protobuf message of a mapped subject; SchemaID is written by Confluent encoding,
// Prefix and Suffix are the bytes of custom framing.
type Framing struct {
	Kind     FramingKind
	SchemaID int32
	Prefix   []byte
	Suffix   []byte
}

// Valid reports whether Kind is a known framing.
func (f Framing) Valid() bool {
	switch f.Kind {
	case FramingNone, FramingGRPC, FramingConfluent, FramingDelimited, FramingCustom:
		return true
	default:
		return false
	}
}
