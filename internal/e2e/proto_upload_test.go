// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package e2e

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"

	codecpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/codec"
	registrypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/registry"
	sourcespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources"
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

func TestProtoUpload(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	created, err := env.sources.CreateSource(ctx, connect.NewRequest(&sourcespb.CreateSourceRequest{
		Name: "uploads", SourceType: protopb.SourceType_SOURCE_TYPE_UPLOAD,
	}))
	require.NoError(t, err)
	sourceID := created.Msg.GetSource().GetId()

	t.Run("files", func(t *testing.T) {
		resp, err := env.sources.UploadSchema(ctx, connect.NewRequest(&sourcespb.UploadSchemaRequest{
			SourceId: sourceID,
			Content: &sourcespb.UploadSchemaRequest_Files{Files: &sourcespb.UploadedFiles{Files: []*sourcespb.UploadedFile{
				{Path: "flow/flow.proto", Content: testProtoContent},
			}}},
		}))
		require.NoError(t, err)
		require.True(t, resp.Msg.GetOutcome().GetValid(), "diagnostics: %v", resp.Msg.GetOutcome().GetDiagnostics())
		assert.Equal(t, protopb.SourceType_SOURCE_TYPE_UPLOAD, resp.Msg.GetSource().GetSourceType())
		assert.Len(t, resp.Msg.GetSource().GetActiveSchema().GetRevision(), 12)

		enc, err := env.codec.EncodeMessage(ctx, connect.NewRequest(&codecpb.EncodeMessageRequest{
			MessageType: "e2eflow.FlowMessage", SourceId: sourceID, Data: `{"name":"up"}`,
		}))
		require.NoError(t, err)
		assert.NotEmpty(t, enc.Msg.GetResult().GetData())
	})

	t.Run("descriptor set", func(t *testing.T) {
		set := prototest.DescriptorSet(t, map[string]string{"parcel.proto": `syntax = "proto3";
package e2eupload;
import "google/protobuf/timestamp.proto";
// A parcel.
message Parcel { google.protobuf.Timestamp at = 1; }
`})
		resp, err := env.sources.UploadSchema(ctx, connect.NewRequest(&sourcespb.UploadSchemaRequest{
			SourceId: sourceID,
			Content:  &sourcespb.UploadSchemaRequest_DescriptorSet{DescriptorSet: set},
		}))
		require.NoError(t, err)
		require.True(t, resp.Msg.GetOutcome().GetValid())

		types, err := env.registry.ListTypes(ctx, connect.NewRequest(&registrypb.ListTypesRequest{SourceId: &sourceID}))
		require.NoError(t, err)
		byName := map[string]*protopb.SchemaType{}
		for _, ty := range types.Msg.GetTypes() {
			byName[ty.GetFullName()] = ty
		}
		require.Contains(t, byName, "e2eupload.Parcel")
		assert.Equal(t, "A parcel.", byName["e2eupload.Parcel"].GetComment())
		assert.True(t, byName["google.protobuf.Timestamp"].GetDependency())
		assert.NotContains(t, byName, "e2eflow.FlowMessage", "the new upload replaces the active schema")

		revisions, err := env.sources.ListRevisions(ctx, connect.NewRequest(&sourcespb.ListRevisionsRequest{SourceId: sourceID}))
		require.NoError(t, err)
		assert.Len(t, revisions.Msg.GetRevisions(), 2)
	})

	t.Run("rejected input", func(t *testing.T) {
		_, err := env.sources.UploadSchema(ctx, connect.NewRequest(&sourcespb.UploadSchemaRequest{SourceId: sourceID}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

		resp, err := env.sources.UploadSchema(ctx, connect.NewRequest(&sourcespb.UploadSchemaRequest{
			SourceId: sourceID,
			Content:  &sourcespb.UploadSchemaRequest_DescriptorSet{DescriptorSet: []byte("not protobuf")},
		}))
		require.NoError(t, err)
		assert.False(t, resp.Msg.GetOutcome().GetValid())
		assert.NotEmpty(t, resp.Msg.GetOutcome().GetDiagnostics())
	})
}
