// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"

	"google.golang.org/protobuf/types/dynamicpb"
)

const everythingProto = `
syntax = "proto3";
package tpl;

import "google/protobuf/any.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/field_mask.proto";
import "google/protobuf/struct.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/wrappers.proto";

enum Kind {
  KIND_UNSPECIFIED = 0;
  KIND_A = 1;
}

message Node {
  string name = 1;
  Node parent = 2;
  repeated Node children = 3;
}

message Everything {
  google.protobuf.Timestamp at = 1;
  google.protobuf.Duration ttl = 2;
  google.protobuf.Struct meta = 3;
  google.protobuf.Value value = 4;
  google.protobuf.ListValue list = 5;
  google.protobuf.Any any = 6;
  google.protobuf.Empty empty = 7;
  google.protobuf.FieldMask mask = 8;
  google.protobuf.Int64Value big = 9;
  google.protobuf.StringValue label = 10;
  google.protobuf.BoolValue flag = 11;
  repeated google.protobuf.Timestamp history = 12;
  map<string, Node> nodes = 13;
  map<int64, Kind> kinds = 14;
  Kind kind = 15;
  int64 count = 16;
  uint64 total = 17;
  bytes blob = 18;
  double ratio = 19;
  oneof choice {
    string text = 20;
    int32 number = 21;
  }
  optional string note = 22;
  Node root = 23;
}
`

func TestTemplate_IsAcceptedByProtojson(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"everything.proto": everythingProto})

	for _, name := range []string{"tpl.Everything", "tpl.Node"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			md, ok := schema.Message(name)
			require.True(t, ok)

			example, err := json.Marshal(protoutils.Template(md))
			require.NoError(t, err)
			require.NoError(t, schema.ParseJSON(example, dynamicpb.NewMessage(md)), "example %s must encode", example)
		})
	}
}

func TestTemplate_ShapesWellKnownTypes(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"everything.proto": everythingProto})
	md, ok := schema.Message("tpl.Everything")
	require.True(t, ok)

	example, ok := protoutils.Template(md).(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "1970-01-01T00:00:00Z", example["at"])
	assert.Equal(t, "0s", example["ttl"])
	assert.Equal(t, map[string]any{}, example["meta"])
	assert.Equal(t, "0", example["big"])
	assert.Equal(t, []any{"1970-01-01T00:00:00Z"}, example["history"])
	assert.Equal(t, "KIND_UNSPECIFIED", example["kind"])
	assert.Equal(t, "0", example["count"])
	assert.Contains(t, example, "text", "first member of a oneof is shown")
	assert.NotContains(t, example, "number", "only one member of a oneof can be set")
	assert.Contains(t, example, "note", "proto3 optional fields are not a real oneof")
}

func TestTemplate_RootWellKnownType(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"everything.proto": everythingProto})
	md, ok := schema.Message("google.protobuf.Timestamp")
	require.True(t, ok)
	assert.Equal(t, "1970-01-01T00:00:00Z", protoutils.Template(md))
}
