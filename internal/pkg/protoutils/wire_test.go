// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package protoutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/types/dynamicpb"
)

func wirePayload() []byte {
	var nested []byte
	nested = protowire.AppendTag(nested, 1, protowire.VarintType)
	nested = protowire.AppendVarint(nested, 7)

	var b []byte
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, 150)
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendString(b, "hello")
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	b = protowire.AppendBytes(b, nested)
	b = protowire.AppendTag(b, 4, protowire.Fixed32Type)
	b = protowire.AppendFixed32(b, 0x3f800000)
	b = protowire.AppendTag(b, 5, protowire.Fixed64Type)
	b = protowire.AppendFixed64(b, 42)
	b = protowire.AppendTag(b, 6, protowire.BytesType)
	b = protowire.AppendBytes(b, []byte{0xff, 0x00})
	return b
}

func TestDecodeWire(t *testing.T) {
	t.Parallel()
	data := wirePayload()

	fields, valid, err := protoutils.DecodeWire(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), valid)
	require.Len(t, fields, 6)

	assert.Equal(t, &entities.WireField{Number: 1, WireType: entities.WireVarint, Offset: 0, Length: 3, Varint: 150}, fields[0])

	assert.Equal(t, entities.WireBytes, fields[1].WireType)
	assert.Equal(t, "hello", fields[1].Text)
	assert.Nil(t, fields[1].Message, "printable text that is not a whole message stays text")

	require.Len(t, fields[2].Message, 1)
	assert.Equal(t, uint64(7), fields[2].Message[0].Varint)
	assert.Equal(t, fields[2].Offset+2, fields[2].Message[0].Offset, "nested offsets point into the payload")

	assert.Equal(t, entities.WireFixed32, fields[3].WireType)
	assert.Equal(t, uint64(0x3f800000), fields[3].Fixed)
	assert.Equal(t, uint64(42), fields[4].Fixed)

	assert.Empty(t, fields[5].Text, "non-printable bytes are not text")
	assert.Equal(t, []byte{0xff, 0x00}, fields[5].Bytes)
}

func TestDecodeWire_StopsAtBrokenField(t *testing.T) {
	t.Parallel()
	data := wirePayload()
	truncated := data[:len(data)-1]

	fields, valid, err := protoutils.DecodeWire(truncated)
	require.Error(t, err)
	assert.Len(t, fields, 5)
	assert.Equal(t, fields[4].Offset+fields[4].Length, valid)

	_, valid, err = protoutils.DecodeWire([]byte{0x00, 0x01})
	require.Error(t, err, "field number 0 is invalid")
	assert.Zero(t, valid)
}

func TestDecodeWire_Group(t *testing.T) {
	t.Parallel()
	var b []byte
	b = protowire.AppendTag(b, 3, protowire.StartGroupType)
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, 9)
	b = protowire.AppendTag(b, 3, protowire.EndGroupType)

	fields, valid, err := protoutils.DecodeWire(b)
	require.NoError(t, err)
	assert.Equal(t, len(b), valid)
	require.Len(t, fields, 1)
	assert.Equal(t, entities.WireGroup, fields[0].WireType)
	require.Len(t, fields[0].Message, 1)
	assert.Equal(t, uint64(9), fields[0].Message[0].Varint)
}

func TestUnknownFields(t *testing.T) {
	t.Parallel()
	newer := prototest.Schema(t, map[string]string{"shop.proto": `syntax = "proto3";
package shop;
message Item { string sku = 1; int64 price = 2; }
message Order { string id = 1; repeated Item items = 2; map<string, Item> by_sku = 3; string note = 4; }
`})
	older := prototest.Schema(t, map[string]string{"shop.proto": `syntax = "proto3";
package shop;
message Item { string sku = 1; }
message Order { string id = 1; repeated Item items = 2; map<string, Item> by_sku = 3; }
`})

	newMD, _ := newer.Message("shop.Order")
	payload := dynamicpb.NewMessage(newMD)
	require.NoError(t, newer.ParseJSON([]byte(`{"id":"o1","items":[{"sku":"a","price":"5"}],"bySku":{"a":{"sku":"a","price":"5"}},"note":"rush"}`), payload))
	data, err := protoutils.EncodeBinary(payload)
	require.NoError(t, err)

	oldMD, _ := older.Message("shop.Order")
	decoded := dynamicpb.NewMessage(oldMD)
	require.NoError(t, older.ParseBinary(data, decoded))

	unknown := protoutils.UnknownFields(decoded)
	assert.ElementsMatch(t, []entities.UnknownField{
		{Path: "", Number: 4, WireType: entities.WireBytes, Size: 6},
		{Path: "items[0]", Number: 2, WireType: entities.WireVarint, Size: 2},
		{Path: "by_sku[a]", Number: 2, WireType: entities.WireVarint, Size: 2},
	}, unknown)
}
