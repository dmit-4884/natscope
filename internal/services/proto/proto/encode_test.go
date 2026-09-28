// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// compileOrderDescriptor compiles an inline message with a string, repeated
// string, and enum field, returning the "Order" message descriptor.
func compileOrderDescriptor(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	dir := t.TempDir()
	p := writeProto(t, dir, "order.proto", `
syntax = "proto3";
package qa5.v1;

enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
}

message Order {
  string id = 1;
  string amount = 2;
  repeated string tags = 3;
  Status status = 4;
}
`)
	entries, diags := protoutils.ReadFilesFromPaths([]string{p})
	require.Empty(t, diags)
	fds, compileDiags, err := compileFiles(t.Context(), entries, nil)
	require.NoError(t, err)
	require.Empty(t, compileDiags)
	require.Len(t, fds, 1)

	md := fds[0].Messages().ByName("Order")
	require.NotNil(t, md, "Order message must be present in compiled file")
	return md
}

// TestDecodeDynamic_UnknownEnumName_Errors is a regression test: an
// unrecognized enum name string must not be silently discarded.
func TestDecodeDynamic_UnknownEnumName_Errors(t *testing.T) {
	t.Parallel()
	md := compileOrderDescriptor(t)

	_, err := decodeDynamic(md, []byte(`{"status":"NOPE","id":"keepme"}`))
	require.Error(t, err, "unknown enum name must surface as an error, not be dropped")
}

// TestDecodeDynamic_UnknownField_Errors covers the PUB-4 part merged into
// a typo'd JSON field name must not be silently discarded either.
func TestDecodeDynamic_UnknownField_Errors(t *testing.T) {
	t.Parallel()
	md := compileOrderDescriptor(t)

	_, err := decodeDynamic(md, []byte(`{"amout":"5","id":"x"}`))
	require.Error(t, err, "unknown JSON field name must surface as an error, not be dropped")
}

// TestDecodeDynamic_UnknownEnumNumber_StillAccepted preserves the existing,
// correct proto3 open-enum behavior: an out-of-range enum number is valid and
// must still round-trip, unlike an unknown enum name.
func TestDecodeDynamic_UnknownEnumNumber_StillAccepted(t *testing.T) {
	t.Parallel()
	md := compileOrderDescriptor(t)

	msg, err := decodeDynamic(md, []byte(`{"status":99}`))
	require.NoError(t, err)
	data, err := encodeDynamic(msg)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

// TestDecodeDynamic_KnownFieldsAndEnum_StillEncodes is a sanity check that
// legitimate input still encodes successfully after tightening DiscardUnknown.
func TestDecodeDynamic_KnownFieldsAndEnum_StillEncodes(t *testing.T) {
	t.Parallel()
	md := compileOrderDescriptor(t)

	msg, err := decodeDynamic(md, []byte(`{"id":"a","amount":"1","tags":["t"],"status":"STATUS_ACTIVE"}`))
	require.NoError(t, err)
	data, err := encodeDynamic(msg)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

// TestEncodeDynamic_Deterministic is a regression test: encoding the same
// JSON repeatedly must always produce identical bytes.
func TestEncodeDynamic_Deterministic(t *testing.T) {
	t.Parallel()
	md := compileOrderDescriptor(t)

	var first string
	for i := range 20 {
		msg, err := decodeDynamic(md, []byte(`{"id":"a","amount":"1","tags":["t"]}`))
		require.NoError(t, err)
		data, err := encodeDynamic(msg)
		require.NoError(t, err)

		encoded := base64.StdEncoding.EncodeToString(data)
		if i == 0 {
			first = encoded
			continue
		}
		assert.Equal(t, first, encoded, "iteration %d produced different bytes for the same JSON input", i)
	}
}
