// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
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
	templatespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// TestManagementOps exercises the JetStream admin operations that TestE2E's
// happy path misses: pause/resume, seal, KV purge, secure delete, batch ops.
func TestManagementOps(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	createResp, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: "mgmt-ops", Urls: []string{env.natsURL},
	}))
	require.NoError(t, err)
	connID := createResp.Msg.GetConnection().GetId()

	const stream = "MGMT_OPS"
	_, err = env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Subjects: []string{"mgmt.ops.>"},
	}))
	require.NoError(t, err)

	t.Run("consumer pause/resume", func(t *testing.T) {
		_, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
			ConnectionId: connID, StreamName: stream, Name: "pausable",
		}))
		require.NoError(t, err)

		pauseUntil := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
		pauseResp, err := env.management.PauseConsumer(ctx, connect.NewRequest(&managementpb.PauseConsumerRequest{
			ConnectionId: connID, StreamName: stream, ConsumerName: "pausable", PauseUntil: pauseUntil,
		}))
		require.NoError(t, err)
		assert.True(t, pauseResp.Msg.GetPaused(), "consumer should report paused=true right after pausing")

		resumeResp, err := env.management.ResumeConsumer(ctx, connect.NewRequest(&managementpb.ResumeConsumerRequest{
			ConnectionId: connID, StreamName: stream, ConsumerName: "pausable",
		}))
		require.NoError(t, err)
		assert.False(t, resumeResp.Msg.GetPaused(), "consumer should report paused=false right after resuming")

		_, err = env.management.PauseConsumer(ctx, connect.NewRequest(&managementpb.PauseConsumerRequest{
			ConnectionId: connID, StreamName: stream, ConsumerName: "pausable", PauseUntil: "not-a-timestamp",
		}))
		require.Error(t, err, "malformed pause_until must be rejected")
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("delete message (plain and secure)", func(t *testing.T) {
		pub1, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "mgmt.ops.del.1", Data: `{"a":1}`,
		}))
		require.NoError(t, err)
		seq1 := pub1.Msg.GetSequence()

		_, err = env.management.DeleteMessage(ctx, connect.NewRequest(&managementpb.DeleteMessageRequest{
			ConnectionId: connID, StreamName: stream, Sequence: seq1,
		}))
		require.NoError(t, err)

		_, err = env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
			ConnectionId: connID, StreamName: stream, Sequence: seq1,
		}))
		require.Error(t, err, "deleted message must be gone")
		assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

		pub2, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "mgmt.ops.del.2", Data: `{"a":2}`,
		}))
		require.NoError(t, err)
		seq2 := pub2.Msg.GetSequence()
		secure := true
		_, err = env.management.DeleteMessage(ctx, connect.NewRequest(&managementpb.DeleteMessageRequest{
			ConnectionId: connID, StreamName: stream, Sequence: seq2, Secure: &secure,
		}))
		require.NoError(t, err, "secure delete should succeed")

		_, err = env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
			ConnectionId: connID, StreamName: stream, Sequence: seq2,
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})

	t.Run("KV purge key history", func(t *testing.T) {
		const bucket = "MGMT_KV"
		_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
			ConnectionId: connID, Config: &natstypes.KVBucketConfig{Bucket: bucket, History: 5},
		}))
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			_, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
				ConnectionId: connID, Bucket: bucket, Key: "k", Value: "v",
			}))
			require.NoError(t, err)
		}

		_, err = env.management.PurgeKVKey(ctx, connect.NewRequest(&managementpb.PurgeKVKeyRequest{
			ConnectionId: connID, Bucket: bucket, Key: "k",
		}))
		require.NoError(t, err)

		_, err = env.management.GetKVKey(ctx, connect.NewRequest(&managementpb.GetKVKeyRequest{
			ConnectionId: connID, Bucket: bucket, Key: "k",
		}))
		require.Error(t, err, "a purged key must read back as not found")
		assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})

	t.Run("seal stream and object bucket", func(t *testing.T) {
		const sealStream = "MGMT_SEAL"
		_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: sealStream, Subjects: []string{"mgmt.seal.>"},
		}))
		require.NoError(t, err)

		sealResp, err := env.management.SealStream(ctx, connect.NewRequest(&managementpb.SealStreamRequest{
			ConnectionId: connID, StreamName: sealStream,
		}))
		require.NoError(t, err)
		assert.True(t, sealResp.Msg.GetStream().GetConfig().GetSealed(), "stream must report sealed=true")

		_, err = env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "mgmt.seal.x", Data: `{}`,
		}))
		require.NoError(t, err, "PublishMessage call itself should not error transport-wise")
		// A sealed stream rejects the publish at the NATS layer; the publish
		// response surfaces it as a soft error rather than blowing up the RPC.

		const objBucket = "MGMT_SEAL_OBJ"
		_, err = env.management.CreateObjectBucket(ctx, connect.NewRequest(&managementpb.CreateObjectBucketRequest{
			ConnectionId: connID, Config: &natstypes.ObjectBucketConfig{Bucket: objBucket},
		}))
		require.NoError(t, err)
		_, err = env.management.SealObjectBucket(ctx, connect.NewRequest(&managementpb.SealObjectBucketRequest{
			ConnectionId: connID, Bucket: objBucket,
		}))
		require.NoError(t, err)

		_, err = env.management.PutObject(ctx, connect.NewRequest(&managementpb.PutObjectRequest{
			ConnectionId: connID, Bucket: objBucket, Name: "f.txt", Data: []byte("x"),
		}))
		require.Error(t, err, "a sealed object bucket must reject new objects")
	})

	t.Run("mappings batch save (created/updated/deleted) and health check", func(t *testing.T) {
		createResp, err := env.mappings.BatchSaveMappings(ctx, connect.NewRequest(&mappingspb.BatchSaveMappingsRequest{
			Mappings: []*mappingspb.MappingBulkItem{
				{Pattern: "batch.a", MessageType: "x.A", SourceId: "s"},
				{Pattern: "batch.b", MessageType: "x.B", SourceId: "s"},
			},
		}))
		require.NoError(t, err)
		assert.EqualValues(t, 2, createResp.Msg.GetCreated())
		assert.EqualValues(t, 0, createResp.Msg.GetUpdated())
		assert.EqualValues(t, 0, createResp.Msg.GetDeleted())

		// Second call replaces the whole set: "batch.a" updated (same pattern+source),
		// "batch.b" dropped (absent from new set), "batch.c" created.
		updateResp, err := env.mappings.BatchSaveMappings(ctx, connect.NewRequest(&mappingspb.BatchSaveMappingsRequest{
			Mappings: []*mappingspb.MappingBulkItem{
				{Pattern: "batch.a", MessageType: "x.A2", SourceId: "s"},
				{Pattern: "batch.c", MessageType: "x.C", SourceId: "s"},
			},
		}))
		require.NoError(t, err)
		assert.EqualValues(t, 1, updateResp.Msg.GetCreated(), "batch.c is new")
		assert.EqualValues(t, 1, updateResp.Msg.GetUpdated(), "batch.a already existed")
		assert.EqualValues(t, 1, updateResp.Msg.GetDeleted(), "batch.b dropped from the new set")

		listResp, err := env.mappings.ListMappings(ctx, connect.NewRequest(&mappingspb.ListMappingsRequest{PageSize: 500}))
		require.NoError(t, err)
		var ids []string
		for _, m := range listResp.Msg.GetMappings() {
			if m.GetPattern() == "batch.a" || m.GetPattern() == "batch.c" {
				ids = append(ids, m.GetId())
			}
			assert.NotEqual(t, "batch.b", m.GetPattern(), "batch.b must have been deleted by the replace")
		}
		require.Len(t, ids, 2)

		healthResp, err := env.mappings.BatchCheckMappingHealth(ctx, connect.NewRequest(&mappingspb.BatchCheckMappingHealthRequest{
			Ids: ids,
		}))
		require.NoError(t, err)
		assert.Len(t, healthResp.Msg.GetItems(), 2)
	})

	t.Run("templates batch create and delete all", func(t *testing.T) {
		batchResp, err := env.templates.BatchCreateTemplates(ctx, connect.NewRequest(&templatespb.BatchCreateTemplatesRequest{
			Templates: []*templatespb.TemplateBulkCreateItem{
				{Name: "bt-1", Subject: "s1", Data: "{}"},
				{Name: "bt-2", Subject: "s2", Data: "{}"},
			},
		}))
		require.NoError(t, err)
		assert.EqualValues(t, 2, batchResp.Msg.GetCreated())

		listBefore, err := env.templates.ListTemplates(ctx, connect.NewRequest(&templatespb.ListTemplatesRequest{PageSize: 500}))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(listBefore.Msg.GetTemplates()), 2)

		_, err = env.templates.DeleteAllTemplates(ctx, connect.NewRequest(&templatespb.DeleteAllTemplatesRequest{}))
		require.NoError(t, err)

		listAfter, err := env.templates.ListTemplates(ctx, connect.NewRequest(&templatespb.ListTemplatesRequest{PageSize: 500}))
		require.NoError(t, err)
		assert.Empty(t, listAfter.Msg.GetTemplates(), "DeleteAllTemplates must leave the store empty")
	})
}
