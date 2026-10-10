// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package proto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"
	"github.com/dmit-4884/natscope/internal/services/proto/registry"

	"google.golang.org/protobuf/encoding/protowire"
)

const decodeProto = `syntax = "proto3";
package shop;
message Order { string id = 1; int64 total = 2; }
`

func TestDecodeWithDescriptor(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"shop.proto": decodeProto})
	md, _ := schema.Message("shop.Order")

	var payload []byte
	payload = protowire.AppendTag(payload, 1, protowire.BytesType)
	payload = protowire.AppendString(payload, "o1")
	payload = protowire.AppendTag(payload, 2, protowire.VarintType)
	payload = protowire.AppendVarint(payload, 99)

	t.Run("unknown fields", func(t *testing.T) {
		t.Parallel()
		data := protowire.AppendTag(append([]byte{}, payload...), 9, protowire.VarintType)
		data = protowire.AppendVarint(data, 1)

		got := decodeWithDescriptor(schema, md, data, "shop.Order", true)
		require.True(t, got.Success)
		assert.Equal(t, []entities.UnknownField{{Number: 9, WireType: entities.WireVarint, Size: 2}}, got.UnknownFields)
	})

	t.Run("partial", func(t *testing.T) {
		t.Parallel()
		data := append(append([]byte{}, payload...), 0x1a, 0x05, 'a')

		got := decodeWithDescriptor(schema, md, data, "shop.Order", true)
		require.False(t, got.Success)
		assert.NotEmpty(t, got.Error)
		assert.Equal(t, len(payload), got.ValidBytes)
		var decoded map[string]any
		require.NoError(t, json.Unmarshal(got.Decoded, &decoded))
		assert.Equal(t, "o1", decoded["id"])
		assert.Equal(t, "99", decoded["total"])
	})

	t.Run("nothing decodable", func(t *testing.T) {
		t.Parallel()
		got := decodeWithDescriptor(schema, md, []byte{0x0a, 0x09, 'x'}, "shop.Order", true)
		require.False(t, got.Success)
		assert.Zero(t, got.ValidBytes)
		assert.Nil(t, got.Decoded)
	})
}

func TestDecodeWire_Service(t *testing.T) {
	t.Parallel()
	dump := newBareService().DecodeWire([]byte{0x08, 0x96, 0x01, 0x12})
	require.Len(t, dump.Fields, 1)
	assert.Equal(t, uint64(150), dump.Fields[0].Varint)
	assert.Equal(t, 3, dump.ValidBytes)
	assert.NotEmpty(t, dump.Error)
}

func TestDecodeWithSnapshot_Framing(t *testing.T) {
	t.Parallel()
	snap := &registry.Snapshot{SourceID: "src", Schema: prototest.Schema(t, map[string]string{"shop.proto": decodeProto})}
	order := []byte{0x0a, 0x02, 'o', '1'}
	grpc := entities.Framing{Kind: entities.FramingGRPC}

	got := decodeWithSnapshot(snap, append([]byte{0, 0, 0, 0, 4}, order...), "shop.Order", grpc)
	require.True(t, got.Success, got.Error)
	assert.JSONEq(t, `{"id":"o1","total":"0"}`, string(got.Decoded))

	wrong := decodeWithSnapshot(snap, order, "shop.Order", grpc)
	require.False(t, wrong.Success)
	assert.Contains(t, wrong.Error, "Cannot unwrap the grpc framing")

	unframed := decodeWithSnapshot(snap, append([]byte{0, 0, 0, 0, 4}, order...), "shop.Order", entities.Framing{})
	require.False(t, unframed.Success)
	assert.Contains(t, unframed.Error, "set the mapping's framing to gRPC")

	custom := entities.Framing{Kind: entities.FramingCustom, Prefix: []byte{0xca, 0xfe}}
	partial := decodeWithSnapshot(snap, append(append([]byte{0xca, 0xfe}, order...), 0x12, 0x09), "shop.Order", custom)
	require.False(t, partial.Success)
	assert.Equal(t, 2+len(order), partial.ValidBytes, "valid bytes count from the start of the payload")
}
