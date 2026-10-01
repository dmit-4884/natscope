// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

const envelopeProto = `
syntax = "proto3";
package demo;

import "google/protobuf/any.proto";

message Inner {
  int32 x = 1;
}

message Envelope {
  google.protobuf.Any payload = 1;
  repeated google.protobuf.Any items = 2;
}
`

func TestParseSchema_CollectsMessagesAndEnumsWithoutMapEntries(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"shop.proto": `
syntax = "proto3";
package shop;

message Order {
  enum Status {
    STATUS_UNSPECIFIED = 0;
    STATUS_PAID = 1;
  }
  message Line {
    string sku = 1;
  }
  map<string, Line> lines = 1;
  Status status = 2;
}
`})

	assert.Contains(t, schema.Messages, "shop.Order")
	assert.Contains(t, schema.Messages, "shop.Order.Line")
	assert.NotContains(t, schema.Messages, "shop.Order.LinesEntry", "map entries are synthetic, never a payload type")
	assert.Contains(t, schema.Enums, "shop.Order.Status")
}

func TestParseSchema_RejectsEmptyAndGarbage(t *testing.T) {
	t.Parallel()
	_, err := protoutils.ParseSchema(nil)
	require.Error(t, err)
	_, err = protoutils.ParseSchema([]byte{0xff, 0xff, 0xff})
	require.Error(t, err)
}

func TestSchema_AnyRoundTripsThroughJSONAndBinary(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"envelope.proto": envelopeProto})
	md, ok := schema.Message("demo.Envelope")
	require.True(t, ok)

	in := dynamicpb.NewMessage(md)
	require.NoError(t, schema.ParseJSON([]byte(
		`{"payload":{"@type":"type.googleapis.com/demo.Inner","x":42},"items":[{"@type":"type.googleapis.com/demo.Inner","x":7}]}`,
	), in))
	wire, err := protoutils.EncodeBinary(in)
	require.NoError(t, err)

	out := dynamicpb.NewMessage(md)
	require.NoError(t, schema.ParseBinary(wire, out))
	rendered, err := schema.RenderJSON(out)
	require.NoError(t, err)
	assert.JSONEq(t,
		`{"payload":{"@type":"type.googleapis.com/demo.Inner","x":42},"items":[{"@type":"type.googleapis.com/demo.Inner","x":7}]}`,
		string(rendered))
}

func TestSchema_AnyWithTypeOutsideSchemaKeepsTypeURL(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"envelope.proto": envelopeProto})
	md, ok := schema.Message("demo.Envelope")
	require.True(t, ok)

	payload, err := proto.Marshal(&anypb.Any{TypeUrl: "type.googleapis.com/elsewhere.Missing", Value: []byte{0x08, 0x01}})
	require.NoError(t, err)
	wire := append([]byte{0x0a, byte(len(payload))}, payload...)

	msg := dynamicpb.NewMessage(md)
	require.NoError(t, schema.ParseBinary(wire, msg))
	rendered, err := schema.RenderJSON(msg)
	require.NoError(t, err, "an Any of an unknown type must not fail the whole message")
	assert.JSONEq(t, `{"payload":{"@type":"type.googleapis.com/elsewhere.Missing"},"items":[]}`, string(rendered))
}

func TestSchema_UnmarshalJSONIsStrict(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"order.proto": `
syntax = "proto3";
package qa;

import "google/protobuf/any.proto";

enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
}

message Order {
  string id = 1;
  repeated string tags = 2;
  Status status = 3;
  google.protobuf.Any extra = 4;
}
`})
	md, ok := schema.Message("qa.Order")
	require.True(t, ok)

	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{name: "known fields and enum name", json: `{"id":"a","tags":["t"],"status":"STATUS_ACTIVE"}`},
		{name: "unknown enum number is kept", json: `{"status":99}`},
		{name: "unknown enum name", json: `{"status":"NOPE"}`, wantErr: true},
		{name: "unknown field", json: `{"idd":"x"}`, wantErr: true},
		{name: "any of a type outside the schema", json: `{"extra":{"@type":"type.googleapis.com/x.Missing"}}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := schema.ParseJSON([]byte(tt.json), dynamicpb.NewMessage(md))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestEncodeBinary_IsDeterministic(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"m.proto": `
syntax = "proto3";
package det;
message M {
  map<string, int32> counts = 1;
  repeated string tags = 2;
}
`})
	md, ok := schema.Message("det.M")
	require.True(t, ok)

	var first []byte
	for i := range 20 {
		msg := dynamicpb.NewMessage(md)
		require.NoError(t, schema.ParseJSON([]byte(`{"counts":{"a":1,"b":2,"c":3,"d":4},"tags":["t"]}`), msg))
		data, err := protoutils.EncodeBinary(msg)
		require.NoError(t, err)
		if i == 0 {
			first = data
			continue
		}
		assert.Equal(t, first, data, "iteration %d produced different bytes", i)
	}
}
