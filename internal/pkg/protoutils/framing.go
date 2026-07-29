// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import "fmt"

// Binary framing detection constants for FramingHint.
const (
	// gRPC frame prefix: 1B compression flag + 4B length.
	gRPCFramePrefixSize = 5

	// Only valid values for the leading byte of a gRPC frame.
	gRPCFrameUncompressedFlag byte = 0x00
	gRPCFrameCompressedFlag   byte = 0x01

	// Isolates the 3-bit wire-type from a protobuf tag byte.
	protobufWireTypeMask byte = 0x07

	// Reserved/invalid wire types — strong "not protobuf" signal as first byte.
	wireTypeStartGroup byte = 3
	wireTypeEndGroup   byte = 4
	wireTypeReserved6  byte = 6
	wireTypeReserved7  byte = 7
)

// FramingHint returns an advisory suffix about likely wrappers (gRPC frame,
// Confluent SR, bad tag) for an undecodable payload. Never auto-strips.
func FramingHint(data []byte) string {
	if len(data) < 1 {
		return ""
	}

	if len(data) >= gRPCFramePrefixSize {
		// gRPC frame: byte0 = flag, byte1..4 = BE uint32 length.
		if data[0] == gRPCFrameUncompressedFlag || data[0] == gRPCFrameCompressedFlag {
			length := uint32(data[1])<<24 | uint32(data[2])<<16 | uint32(data[3])<<8 | uint32(data[4])
			if length == uint32(len(data)-gRPCFramePrefixSize) {
				return " (note: payload looks like a gRPC frame; strip the 5-byte prefix before decoding)"
			}
		}
		// Confluent SR frame: 0x00 magic + 4-byte BE schema id.
		if data[0] == gRPCFrameUncompressedFlag {
			return " (note: payload may be Confluent SR framed; strip the 5-byte schema-id prefix)"
		}
	}

	first := data[0]
	wireType := first & protobufWireTypeMask
	// Valid wire types: 0,1,2,5; 3,4 deprecated groups, 6,7 invalid.
	if wireType == wireTypeStartGroup || wireType == wireTypeEndGroup ||
		wireType == wireTypeReserved6 || wireType == wireTypeReserved7 {
		return fmt.Sprintf(
			" (note: first byte 0x%02x is not a valid protobuf tag — payload likely has a non-protobuf prefix)",
			first,
		)
	}

	return ""
}
