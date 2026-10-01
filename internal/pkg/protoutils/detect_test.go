// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"

	"google.golang.org/protobuf/encoding/protowire"
)

const detectProto = `syntax = "proto3";
package det;
message User { string name = 1; int64 id = 2; string email = 3; }
message Note { string text = 1; }
message Point { double x = 1; double y = 2; }
message Series { int64 value = 1; repeated int64 history = 2; }
message Index { map<string, string> labels = 1; }
`

var detectTypes = []string{"det.Index", "det.Index.LabelsEntry", "det.Note", "det.Point", "det.Series", "det.User"}

func userPayload() []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, "ann")
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, 7)
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	return protowire.AppendString(b, "a@b")
}

func TestSchema_DetectTypes(t *testing.T) {
	t.Parallel()
	schema := prototest.Schema(t, map[string]string{"det.proto": detectProto})

	t.Run("best match first", func(t *testing.T) {
		t.Parallel()
		got := schema.DetectTypes(userPayload(), detectTypes, 10)
		require.Len(t, got, 2)
		assert.Equal(t, "det.User", got[0].MessageType)
		assert.Equal(t, 93, got[0].Score)
		assert.Zero(t, got[0].UnknownBytes)
		assert.Equal(t, "det.Note", got[1].MessageType)
		assert.Equal(t, 44, got[1].Score)
		assert.Equal(t, 7, got[1].UnknownBytes)
	})

	t.Run("map entries skipped", func(t *testing.T) {
		t.Parallel()
		var b []byte
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, "a")
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendString(b, "b")

		got := schema.DetectTypes(b, detectTypes, 10)
		require.Len(t, got, 1)
		assert.Equal(t, "det.Note", got[0].MessageType)
	})

	t.Run("limit", func(t *testing.T) {
		t.Parallel()
		got := schema.DetectTypes(userPayload(), detectTypes, 1)
		require.Len(t, got, 1)
		assert.Equal(t, "det.User", got[0].MessageType)
	})

	t.Run("packed repeated field", func(t *testing.T) {
		t.Parallel()
		var b []byte
		b = protowire.AppendTag(b, 1, protowire.VarintType)
		b = protowire.AppendVarint(b, 1)
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendBytes(b, []byte{1, 2, 3})

		got := schema.DetectTypes(b, detectTypes, 10)
		require.Len(t, got, 1)
		assert.Equal(t, "det.Series", got[0].MessageType)
		assert.Equal(t, 92, got[0].Score)
	})

	t.Run("not protobuf", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, schema.DetectTypes([]byte{0xff, 0xff, 0xff}, detectTypes, 10))
	})

	t.Run("no declared field", func(t *testing.T) {
		t.Parallel()
		b := protowire.AppendTag(nil, 9, protowire.VarintType)
		assert.Empty(t, schema.DetectTypes(protowire.AppendVarint(b, 1), detectTypes, 10))
	})

	t.Run("unknown type names", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, schema.DetectTypes(userPayload(), []string{"det.Missing"}, 10))
	})
}
