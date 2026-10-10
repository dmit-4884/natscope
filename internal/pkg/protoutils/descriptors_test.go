// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package protoutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func setOf(t *testing.T, files ...*descriptorpb.FileDescriptorProto) []byte {
	t.Helper()
	data, err := proto.Marshal(&descriptorpb.FileDescriptorSet{File: files})
	require.NoError(t, err)
	return data
}

func fileWith(name, pkg string, messages ...*descriptorpb.DescriptorProto) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name: new(name), Package: new(pkg), MessageType: messages, Syntax: new("proto3"),
	}
}

func message(name string, fields ...string) *descriptorpb.DescriptorProto {
	m := &descriptorpb.DescriptorProto{Name: new(name)}
	for i, f := range fields {
		m.Field = append(m.Field, &descriptorpb.FieldDescriptorProto{
			Name: new(f), Number: new(int32(i + 1)), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		})
	}
	return m
}

func TestFindConflicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		inputs func(t *testing.T) []SchemaInput
		want   []entities.SchemaConflict
	}{
		{
			name: "same type defined differently in two files",
			inputs: func(t *testing.T) []SchemaInput {
				return []SchemaInput{
					{SourceID: "a", Revision: "v1", Bytes: setOf(t, fileWith("agent.proto", "ag", message("Stats", "id")))},
					{SourceID: "b", Revision: "v3", Bytes: setOf(t, fileWith("agents.proto", "ag", message("Stats", "id", "count")))},
				}
			},
			want: []entities.SchemaConflict{{
				Kind: entities.ConflictDifferentShape, Severity: entities.SeverityError, Symbol: "ag.Stats",
				First:  entities.SchemaRef{SourceID: "a", Revision: "v1", File: "agent.proto"},
				Second: entities.SchemaRef{SourceID: "b", Revision: "v3", File: "agents.proto"},
				Reason: "two sources define the type differently",
			}},
		},
		{
			name: "same type defined identically in two files",
			inputs: func(t *testing.T) []SchemaInput {
				return []SchemaInput{
					{SourceID: "a", Bytes: setOf(t, fileWith("money.proto", "fin", message("Money", "units")))},
					{SourceID: "b", Bytes: setOf(t, fileWith("vendor/money.proto", "fin", message("Money", "units")))},
				}
			},
			want: []entities.SchemaConflict{{
				Kind: entities.ConflictSameShape, Severity: entities.SeverityInfo, Symbol: "fin.Money",
				First:  entities.SchemaRef{SourceID: "a", File: "money.proto"},
				Second: entities.SchemaRef{SourceID: "b", File: "vendor/money.proto"},
				Reason: "two sources define the type identically",
			}},
		},
		{
			name: "one file path with different content",
			inputs: func(t *testing.T) []SchemaInput {
				return []SchemaInput{
					{SourceID: "a", Bytes: setOf(t, fileWith("common.proto", "c", message("Common")))},
					{SourceID: "b", Bytes: setOf(t, fileWith("common.proto", "c", message("CommonV2")))},
				}
			},
			want: []entities.SchemaConflict{{
				Kind: entities.ConflictFileContent, Severity: entities.SeverityError, Symbol: "common.proto",
				First:  entities.SchemaRef{SourceID: "a", File: "common.proto"},
				Second: entities.SchemaRef{SourceID: "b", File: "common.proto"},
				Reason: "the file has different content in two sources",
			}},
		},
		{
			name: "different types",
			inputs: func(t *testing.T) []SchemaInput {
				return []SchemaInput{
					{SourceID: "a", Bytes: setOf(t, fileWith("foo.proto", "x", message("Foo")))},
					{SourceID: "b", Bytes: setOf(t, fileWith("bar.proto", "x", message("Bar")))},
				}
			},
		},
		{
			name: "one file shipped twice",
			inputs: func(t *testing.T) []SchemaInput {
				set := setOf(t, fileWith("common.proto", "c", message("Common")))
				return []SchemaInput{{SourceID: "a", Bytes: set}, {SourceID: "b", Bytes: set}}
			},
		},
		{
			name: "comments differ",
			inputs: func(t *testing.T) []SchemaInput {
				commented := func(text string) *descriptorpb.FileDescriptorProto {
					f := fileWith("common.proto", "c", message("Common"))
					f.SourceCodeInfo = &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{
						{Path: []int32{4, 0}, Span: []int32{1, 0, 20}, LeadingComments: new(text)},
					}}
					return f
				}
				return []SchemaInput{
					{SourceID: "a", Bytes: setOf(t, commented(" first"))},
					{SourceID: "b", Bytes: setOf(t, commented(" second"))},
				}
			},
		},
		{
			name: "well-known files differ",
			inputs: func(t *testing.T) []SchemaInput {
				return []SchemaInput{
					{SourceID: "a", Bytes: setOf(t, fileWith("google/protobuf/any.proto", "google.protobuf", message("Any", "type_url")))},
					{SourceID: "b", Bytes: setOf(t, fileWith("google/protobuf/any.proto", "google.protobuf", message("Any")))},
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FindConflicts(tt.inputs(t))
			require.Len(t, got, len(tt.want))
			for i, c := range got {
				c.BaseEntity = entities.BaseEntity{}
				assert.Equal(t, tt.want[i], *c)
			}
		})
	}
}
