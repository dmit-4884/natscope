// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package protoutils_test

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"
)

const framingProto = `syntax = "proto3";
package shop;
message Order { string id = 1; message Line { string sku = 1; } }
message Refund { string id = 1; }
`

var framedMsg = []byte{0x0a, 0x02, 'o', '1'}

func withMsg(prefix ...byte) []byte {
	return append(prefix, framedMsg...)
}

func TestFrameUnframe(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"shop.proto": framingProto})
	order, _ := schema.Message("shop.Order")

	tests := []struct {
		name    string
		framing entities.Framing
		want    []byte
	}{
		{name: "none", framing: entities.Framing{}, want: framedMsg},
		{name: "grpc", framing: entities.Framing{Kind: entities.FramingGRPC}, want: withMsg(0, 0, 0, 0, 4)},
		{name: "varint delimited", framing: entities.Framing{Kind: entities.FramingDelimited}, want: withMsg(4)},
		{
			name:    "custom",
			framing: entities.Framing{Kind: entities.FramingCustom, Prefix: []byte{0xca, 0xfe}, Suffix: []byte{0x0d, 0x0a}},
			want:    append(withMsg(0xca, 0xfe), 0x0d, 0x0a),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			framed, err := protoutils.Frame(framedMsg, tt.framing, order)
			require.NoError(t, err)
			assert.Equal(t, tt.want, framed)
			got, offset, err := protoutils.Unframe(framed, tt.framing)
			require.NoError(t, err)
			assert.Equal(t, framedMsg, got)
			assert.Equal(t, len(framed)-len(framedMsg)-len(tt.framing.Suffix), offset)
		})
	}
}

func TestFrame_ConfluentMessageIndexes(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"shop.proto": framingProto})
	confluent := entities.Framing{Kind: entities.FramingConfluent, SchemaID: 258}

	tests := []struct {
		name string
		md   string
		want []byte
	}{
		{name: "first message is a single 0", md: "shop.Order", want: withMsg(0, 0, 0, 1, 2, 0)},
		{name: "second message is count 1 then index 1, zigzag", md: "shop.Refund", want: withMsg(0, 0, 0, 1, 2, 2, 2)},
		{name: "nested message", md: "shop.Order.Line", want: withMsg(0, 0, 0, 1, 2, 4, 0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			md, ok := schema.Message(tt.md)
			require.True(t, ok)
			framed, err := protoutils.Frame(framedMsg, confluent, md)
			require.NoError(t, err)
			assert.Equal(t, tt.want, framed)
			got, offset, err := protoutils.Unframe(framed, confluent)
			require.NoError(t, err)
			assert.Equal(t, framedMsg, got)
			assert.Equal(t, len(tt.want)-len(framedMsg), offset)
		})
	}
}

func TestUnframe_CompressedGRPC(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(framedMsg)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	framed := append([]byte{1, 0, 0, 0, byte(buf.Len())}, buf.Bytes()...)

	got, offset, err := protoutils.Unframe(framed, entities.Framing{Kind: entities.FramingGRPC})
	require.NoError(t, err)
	assert.Equal(t, framedMsg, got)
	assert.Equal(t, -1, offset, "inflated bytes don't map onto the payload")
}

func TestUnframe_RejectsGzipBomb(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(make([]byte, 17<<20))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	header := []byte{1, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(header[1:], uint32(buf.Len()))

	_, _, err = protoutils.Unframe(append(header, buf.Bytes()...), entities.Framing{Kind: entities.FramingGRPC})
	require.ErrorContains(t, err, "inflates past")
}

func TestUnframe_Rejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    []byte
		framing entities.Framing
	}{
		{name: "grpc too short", data: []byte{0, 0}, framing: entities.Framing{Kind: entities.FramingGRPC}},
		{name: "grpc wrong length", data: withMsg(0, 0, 0, 0, 9), framing: entities.Framing{Kind: entities.FramingGRPC}},
		{name: "grpc bad flag", data: withMsg(7, 0, 0, 0, 4), framing: entities.Framing{Kind: entities.FramingGRPC}},
		{name: "grpc compressed garbage", data: withMsg(1, 0, 0, 0, 4), framing: entities.Framing{Kind: entities.FramingGRPC}},
		{name: "confluent without magic", data: withMsg(1, 0, 0, 0, 1, 0), framing: entities.Framing{Kind: entities.FramingConfluent}},
		{name: "delimited wrong length", data: withMsg(9), framing: entities.Framing{Kind: entities.FramingDelimited}},
		{name: "custom prefix", data: framedMsg, framing: entities.Framing{Kind: entities.FramingCustom, Prefix: []byte{0xca}}},
		{
			name:    "custom suffix",
			data:    withMsg(0xca),
			framing: entities.Framing{Kind: entities.FramingCustom, Prefix: []byte{0xca}, Suffix: []byte{0xfe}},
		},
		{name: "unknown kind", data: framedMsg, framing: entities.Framing{Kind: "zstd"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := protoutils.Unframe(tt.data, tt.framing)
			require.Error(t, err)
		})
	}
}

func TestFramingHint(t *testing.T) {
	t.Parallel()
	assert.Contains(t, protoutils.FramingHint(withMsg(0, 0, 0, 0, 4)), "gRPC")
	assert.Contains(t, protoutils.FramingHint(withMsg(0, 0, 0, 1, 2, 0)), "Confluent")
	assert.Contains(t, protoutils.FramingHint(withMsg(4)), "varint-delimited")
	assert.Contains(t, protoutils.FramingHint([]byte{0xff, 0x01}), "custom framing")
	assert.Empty(t, protoutils.FramingHint(framedMsg))
}
