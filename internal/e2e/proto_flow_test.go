// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	codecpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/codec"
	registrypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/registry"
	sourcespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources"
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

const testProtoContent = `syntax = "proto3";

package e2eflow;

// A message flowing through the e2e pipeline.
message FlowMessage {
  string name = 1; // Display name.
  int32 count = 2;
  repeated string tags = 3;
}
`

func createLocalSource(t *testing.T, env *e2eEnv, name string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
	}
	createResp, err := env.sources.CreateSource(t.Context(), connect.NewRequest(&sourcespb.CreateSourceRequest{
		Name: name, SourceType: protopb.SourceType_SOURCE_TYPE_LOCAL, LocalPath: &dir, WatcherEnabled: new(false),
	}))
	require.NoError(t, err)
	sourceID := createResp.Msg.GetSource().GetId()
	refreshResp, err := env.sources.RefreshSource(t.Context(), connect.NewRequest(&sourcespb.RefreshSourceRequest{SourceId: sourceID}))
	require.NoError(t, err)
	outcome := refreshResp.Msg.GetOutcome()
	require.True(t, outcome.GetValid(), "compile diagnostics: %v", outcome.GetDiagnostics())
	require.NotEmpty(t, refreshResp.Msg.GetSource().GetActiveSchema().GetFingerprint())
	return sourceID
}

// TestProtoFlow drives the full proto pipeline end-to-end: create, compile,
// mapping, and server-side decode — the path TestE2E's registry_codec_sources
// subtest explicitly skips for lack of a real proto source.
func TestProtoFlow(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	sourceID := createLocalSource(t, env, "flow-source", map[string]string{"flow.proto": testProtoContent})

	const fullName = "e2eflow.FlowMessage"

	t.Run("registry sees the compiled source", func(t *testing.T) {
		statusResp, err := env.registry.GetProtoStatus(ctx, connect.NewRequest(&registrypb.GetProtoStatusRequest{}))
		require.NoError(t, err)
		assert.True(t, statusResp.Msg.GetLoaded())
		assert.Greater(t, statusResp.Msg.GetMessageCount(), int32(0))

		typesResp, err := env.registry.ListTypes(ctx, connect.NewRequest(&registrypb.ListTypesRequest{SourceId: &sourceID}))
		require.NoError(t, err)
		require.Len(t, typesResp.Msg.GetTypes(), 1)
		assert.Equal(t, fullName, typesResp.Msg.GetTypes()[0].GetFullName())
		assert.Equal(t, "A message flowing through the e2e pipeline.", typesResp.Msg.GetTypes()[0].GetComment())
		assert.False(t, typesResp.Msg.GetTypes()[0].GetDependency())

		descResp, err := env.registry.DescribeType(ctx, connect.NewRequest(&registrypb.DescribeTypeRequest{
			FullName: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)
		require.Len(t, descResp.Msg.GetMessages(), 1)
		fields := descResp.Msg.GetMessages()[0].GetFields()
		require.Len(t, fields, 3)
		assert.Equal(t, "Display name.", fields[0].GetComment())
		assert.True(t, fields[2].GetRepeated())

		exResp, err := env.registry.GenerateExample(ctx, connect.NewRequest(&registrypb.GenerateExampleRequest{
			FullName: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)
		assert.NotEmpty(t, exResp.Msg.GetJson())
	})

	var wireBytes []byte

	t.Run("codec encode/validate/decode round-trip", func(t *testing.T) {
		encResp, err := env.codec.EncodeMessage(ctx, connect.NewRequest(&codecpb.EncodeMessageRequest{
			MessageType: fullName, SourceId: sourceID,
			Data: `{"name":"hello","count":42,"tags":["a","b"]}`,
		}))
		require.NoError(t, err)
		wireBytes = encResp.Msg.GetResult().GetData()
		require.NotEmpty(t, wireBytes)

		valResp, err := env.codec.ValidateMessage(ctx, connect.NewRequest(&codecpb.ValidateMessageRequest{
			Data: wireBytes, MessageType: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)
		assert.True(t, valResp.Msg.GetResult().GetValid())

		decResp, err := env.codec.DecodeMessage(ctx, connect.NewRequest(&codecpb.DecodeMessageRequest{
			Data: wireBytes, MessageType: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)
		assert.Contains(t, decResp.Msg.GetResult().GetData(), "hello")
	})

	// Publish a raw (proto-encoded) message on a subject mapped to FlowMessage
	// and verify it comes back decoded through ListMessages/GetMessage.
	connResp, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: "proto-flow-conn", Urls: []string{env.natsURL},
	}))
	require.NoError(t, err)
	connID := connResp.Msg.GetConnection().GetId()

	const stream = "PROTO_FLOW"
	const subject = "flow.msg"
	_, err = env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Subjects: []string{subject},
	}))
	require.NoError(t, err)

	mappingResp, err := env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
		Pattern: subject, MessageType: fullName, SourceId: sourceID,
	}))
	require.NoError(t, err)

	t.Run("mapping health follows the pinned version", func(t *testing.T) {
		pinnedResp, err := env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
			Pattern: "flow.pinned", MessageType: fullName, SourceId: sourceID, PinnedFingerprint: new("no-such-fingerprint"),
		}))
		require.NoError(t, err)

		healthResp, err := env.mappings.BatchCheckMappingHealth(ctx, connect.NewRequest(&mappingspb.BatchCheckMappingHealthRequest{
			Ids: []string{mappingResp.Msg.GetMapping().GetId(), pinnedResp.Msg.GetMapping().GetId(), unknownUUID},
		}))
		require.NoError(t, err)
		items := healthResp.Msg.GetItems()
		require.Len(t, items, 3)
		assert.Equal(t, "ok", items[0].GetHealth())
		assert.Equal(t, "descriptor_missing", items[1].GetHealth())
		assert.Equal(t, "mapping_missing", items[2].GetHealth())

		_, err = env.mappings.DeleteMapping(ctx, connect.NewRequest(&mappingspb.DeleteMappingRequest{Id: pinnedResp.Msg.GetMapping().GetId()}))
		require.NoError(t, err)
	})

	t.Run("publish + server-side proto decode", func(t *testing.T) {
		msgType := fullName
		pubResp, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: subject, MessageType: &msgType, SourceId: &sourceID,
			Data: `{"name":"decoded","count":7,"tags":["x"]}`,
		}))
		require.NoError(t, err)
		require.Empty(t, pubResp.Msg.GetError())
		seq := pubResp.Msg.GetSequence()
		require.Greater(t, seq, uint64(0))

		var got *messagespb.GetMessageResponse
		require.Eventually(t, func() bool {
			resp, err := env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
				ConnectionId: connID, StreamName: stream, Sequence: seq,
			}))
			if err != nil {
				return false
			}
			got = resp.Msg
			return got.GetMessage() != nil
		}, 5*time.Second, 100*time.Millisecond)

		require.NotNil(t, got)
		decoded := got.GetMessage().GetDecoded()
		require.NotEmpty(t, decoded, "message on a mapped subject must be server-side proto-decoded")
		assert.Contains(t, decoded, "decoded")
		assert.Equal(t, fullName, got.GetMessage().GetDecodedType())
	})
}
