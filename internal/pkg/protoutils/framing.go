// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"

	coreio "github.com/altessa-s/go-atlas/core/io"
)

const (
	gRPCFramePrefixSize            = 5
	gRPCFrameUncompressedFlag byte = 0x00
	gRPCFrameCompressedFlag   byte = 0x01
	confluentMagic            byte = 0x00
	confluentHeaderSize            = 5
	maxInflatedBytes               = 16 << 20

	protobufWireTypeMask byte = 0x07
	wireTypeStartGroup   byte = 3
	wireTypeEndGroup     byte = 4
	wireTypeReserved6    byte = 6
	wireTypeReserved7    byte = 7
)

// Unframe strips the framing around a protobuf message; offset is where the message starts in data,
// -1 when the message was decompressed and its bytes don't map onto data.
func Unframe(data []byte, f entities.Framing) (msg []byte, offset int, err error) {
	switch f.Kind {
	case entities.FramingNone:
		return data, 0, nil
	case entities.FramingGRPC:
		return unframeGRPC(data)
	case entities.FramingConfluent:
		return unframeConfluent(data)
	case entities.FramingDelimited:
		size, n := protowire.ConsumeVarint(data)
		if n < 0 {
			return nil, 0, errors.New("missing varint length prefix")
		}
		if size != uint64(len(data)-n) {
			return nil, 0, fmt.Errorf("length prefix says %d bytes, %d follow", size, len(data)-n)
		}
		return data[n:], n, nil
	case entities.FramingCustom:
		if !bytes.HasPrefix(data, f.Prefix) {
			return nil, 0, fmt.Errorf("payload does not start with %s", hex.EncodeToString(f.Prefix))
		}
		if !bytes.HasSuffix(data[len(f.Prefix):], f.Suffix) {
			return nil, 0, fmt.Errorf("payload does not end with %s", hex.EncodeToString(f.Suffix))
		}
		return data[len(f.Prefix) : len(data)-len(f.Suffix)], len(f.Prefix), nil
	default:
		return nil, 0, fmt.Errorf("unknown framing %q", f.Kind)
	}
}

func unframeGRPC(data []byte) ([]byte, int, error) {
	if len(data) < gRPCFramePrefixSize {
		return nil, 0, errors.New("shorter than the 5-byte gRPC frame header")
	}
	size := binary.BigEndian.Uint32(data[1:gRPCFramePrefixSize])
	if int64(size) != int64(len(data)-gRPCFramePrefixSize) {
		return nil, 0, fmt.Errorf("gRPC frame says %d bytes, %d follow", size, len(data)-gRPCFramePrefixSize)
	}
	body := data[gRPCFramePrefixSize:]
	switch data[0] {
	case gRPCFrameUncompressedFlag:
		return body, gRPCFramePrefixSize, nil
	case gRPCFrameCompressedFlag:
		zr, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, 0, fmt.Errorf("compressed gRPC frame is not gzip: %w", err)
		}
		inflated, err := io.ReadAll(coreio.NewLimitedReadCloser(zr, maxInflatedBytes))
		if errors.Is(err, coreio.ErrReadLimitExceeded) {
			return nil, 0, fmt.Errorf("compressed gRPC frame inflates past %d bytes", maxInflatedBytes)
		}
		if err != nil {
			return nil, 0, fmt.Errorf("inflate gRPC frame: %w", err)
		}
		return inflated, -1, nil
	default:
		return nil, 0, fmt.Errorf("gRPC compression flag must be 0 or 1, got %d", data[0])
	}
}

func unframeConfluent(data []byte) ([]byte, int, error) {
	if len(data) < confluentHeaderSize || data[0] != confluentMagic {
		return nil, 0, errors.New("missing the Confluent magic byte and 4-byte schema id")
	}
	offset := confluentHeaderSize
	count, n := protowire.ConsumeVarint(data[offset:])
	if n < 0 {
		return nil, 0, errors.New("missing Confluent message indexes")
	}
	offset += n
	for range protowire.DecodeZigZag(count) {
		_, n = protowire.ConsumeVarint(data[offset:])
		if n < 0 {
			return nil, 0, errors.New("truncated Confluent message indexes")
		}
		offset += n
	}
	return data[offset:], offset, nil
}

// Frame wraps an encoded message; md picks the Confluent message indexes.
func Frame(msg []byte, f entities.Framing, md protoreflect.MessageDescriptor) ([]byte, error) {
	switch f.Kind {
	case entities.FramingNone:
		return msg, nil
	case entities.FramingGRPC:
		out := make([]byte, gRPCFramePrefixSize, gRPCFramePrefixSize+len(msg))
		binary.BigEndian.PutUint32(out[1:], uint32(len(msg))) //nolint:gosec // bounded by the request size
		return append(out, msg...), nil
	case entities.FramingConfluent:
		out := make([]byte, confluentHeaderSize, confluentHeaderSize+len(msg))
		binary.BigEndian.PutUint32(out[1:], uint32(f.SchemaID)) //nolint:gosec // schema ids are positive
		return append(appendMessageIndexes(out, md), msg...), nil
	case entities.FramingDelimited:
		return append(protowire.AppendVarint(nil, uint64(len(msg))), msg...), nil
	case entities.FramingCustom:
		out := append(append([]byte{}, f.Prefix...), msg...)
		return append(out, f.Suffix...), nil
	default:
		return nil, fmt.Errorf("unknown framing %q", f.Kind)
	}
}

func appendMessageIndexes(b []byte, md protoreflect.MessageDescriptor) []byte {
	var indexes []int
	for d := protoreflect.Descriptor(md); d != nil; d = d.Parent() {
		if _, ok := d.(protoreflect.MessageDescriptor); ok {
			indexes = append([]int{d.Index()}, indexes...)
		}
	}
	if len(indexes) == 1 && indexes[0] == 0 {
		return append(b, 0)
	}
	b = protowire.AppendVarint(b, protowire.EncodeZigZag(int64(len(indexes))))
	for _, i := range indexes {
		b = protowire.AppendVarint(b, protowire.EncodeZigZag(int64(i)))
	}
	return b
}

// FramingHint suggests a framing for a payload that does not decode unframed. Never strips anything.
func FramingHint(data []byte) string {
	if len(data) < 1 {
		return ""
	}

	if len(data) >= gRPCFramePrefixSize {
		if data[0] == gRPCFrameUncompressedFlag || data[0] == gRPCFrameCompressedFlag {
			if int64(binary.BigEndian.Uint32(data[1:gRPCFramePrefixSize])) == int64(len(data)-gRPCFramePrefixSize) {
				return " (note: payload looks like a gRPC frame; set the mapping's framing to gRPC)"
			}
		}
		if data[0] == confluentMagic {
			return " (note: payload may be Confluent Schema Registry framed; set the mapping's framing to Confluent)"
		}
	}

	if size, n := protowire.ConsumeVarint(data); n > 0 && n < len(data) && size == uint64(len(data)-n) {
		return " (note: payload looks length-delimited; set the mapping's framing to varint-delimited)"
	}

	first := data[0]
	wireType := first & protobufWireTypeMask
	if wireType == wireTypeStartGroup || wireType == wireTypeEndGroup ||
		wireType == wireTypeReserved6 || wireType == wireTypeReserved7 {
		return fmt.Sprintf(
			" (note: first byte 0x%02x is not a valid protobuf tag; the payload likely has a prefix, set a custom framing)",
			first,
		)
	}

	return ""
}
