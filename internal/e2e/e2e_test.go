// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"encoding/base64"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/durationpb"

	historypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/history/v1/history"
	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	statspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/stats"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
	registrypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/registry"
	sourcespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources"
	settingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/settings/v1/settings"
	templatespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
	settingstypes "github.com/dmit-4884/natscope/proto/gen/types/settings"
)

const (
	streamName  = "E2E"
	testSubject = "e2e.test"
)

// recentTimestamp asserts a google.protobuf.Timestamp is present and points at
// roughly "now" (within a generous window to tolerate slow CI).
func recentTimestamp(t *testing.T, ts interface{ AsTime() time.Time }, label string) {
	t.Helper()
	require.NotNil(t, ts, "%s must be non-nil", label)
	got := ts.AsTime()
	assert.WithinDuration(t, time.Now(), got, 10*time.Minute, "%s should be recent", label)
}

// TestE2E exercises every major flow through the real Connect transport
// against an embedded NATS server; subtests share state and run in order,
// reusing the connection the first subtest creates.
func TestE2E(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	// connectionID is set by the connections subtest and consumed by the rest.
	var connectionID string

	t.Run("connections", func(t *testing.T) {
		// Create
		desc := "e2e connection"
		createResp, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
			Name:        "e2e",
			Description: &desc,
			Urls:        []string{env.natsURL},
		}))
		require.NoError(t, err)
		conn := createResp.Msg.GetConnection()
		require.NotNil(t, conn)
		require.NotEmpty(t, conn.GetId())
		assert.Equal(t, "e2e", conn.GetName())
		assert.Equal(t, []string{env.natsURL}, conn.GetUrls())
		recentTimestamp(t, conn.GetCreatedAt(), "connection.created_at")
		connectionID = conn.GetId()

		// List
		listResp, err := env.connections.ListConnections(ctx, connect.NewRequest(&connectionspb.ListConnectionsRequest{}))
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(listResp.Msg.GetConnections()), 1)
		assert.True(t, containsConnID(listResp.Msg.GetConnections(), connectionID))

		// TestConnection — should succeed and report server info (JetStream enabled).
		testResp, err := env.connections.TestConnection(ctx, connect.NewRequest(&connectionspb.TestConnectionRequest{
			Urls:           []string{env.natsURL},
			ConnectTimeout: durationpb.New(10 * time.Second),
		}))
		require.NoError(t, err)
		assert.True(t, testResp.Msg.GetSuccess(), "test connection should succeed: %s", testResp.Msg.GetError())
		assert.NotEmpty(t, testResp.Msg.GetServerVersion(), "server version should be reported")
		assert.True(t, testResp.Msg.GetJetstreamEnabled(), "embedded server has JetStream enabled")

		// Update (change description)
		newDesc := "updated e2e connection"
		updResp, err := env.connections.UpdateConnection(ctx, connect.NewRequest(&connectionspb.UpdateConnectionRequest{
			Id:          connectionID,
			Description: &newDesc,
		}))
		require.NoError(t, err)
		assert.Equal(t, newDesc, updResp.Msg.GetConnection().GetDescription())

		// Duplicate, then delete the duplicate.
		dupResp, err := env.connections.DuplicateConnection(ctx, connect.NewRequest(&connectionspb.DuplicateConnectionRequest{
			Id:   connectionID,
			Name: "e2e-copy",
		}))
		require.NoError(t, err)
		dupID := dupResp.Msg.GetConnection().GetId()
		require.NotEmpty(t, dupID)
		assert.NotEqual(t, connectionID, dupID)

		_, err = env.connections.DeleteConnection(ctx, connect.NewRequest(&connectionspb.DeleteConnectionRequest{Id: dupID}))
		require.NoError(t, err)

		// The original must remain.
		listResp2, err := env.connections.ListConnections(ctx, connect.NewRequest(&connectionspb.ListConnectionsRequest{}))
		require.NoError(t, err)
		assert.True(t, containsConnID(listResp2.Msg.GetConnections(), connectionID))
		assert.False(t, containsConnID(listResp2.Msg.GetConnections(), dupID))
	})

	require.NotEmpty(t, connectionID, "connections subtest must establish a connection id")

	t.Run("streams_and_management", func(t *testing.T) {
		// Create stream capturing e2e.>
		createResp, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connectionID,
			Name:         streamName,
			Subjects:     []string{"e2e.>"},
			MaxMsgs:      1000,
		}))
		require.NoError(t, err)
		require.NotNil(t, createResp.Msg.GetStream())
		assert.Equal(t, streamName, createResp.Msg.GetStream().GetConfig().GetName())

		// ListStreams contains E2E
		listResp, err := env.streams.ListStreams(ctx, connect.NewRequest(&streamspb.ListStreamsRequest{
			ConnectionId: connectionID,
		}))
		require.NoError(t, err)
		assert.True(t, containsStream(listResp.Msg.GetStreams(), streamName))

		// GetStream
		getResp, err := env.streams.GetStream(ctx, connect.NewRequest(&streamspb.GetStreamRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
		}))
		require.NoError(t, err)
		require.NotNil(t, getResp.Msg.GetStream())
		assert.Equal(t, streamName, getResp.Msg.GetStream().GetConfig().GetName())

		// UpdateStream — bump max_msgs
		newMax := int64(5000)
		updResp, err := env.management.UpdateStream(ctx, connect.NewRequest(&managementpb.UpdateStreamRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
			Subjects:     []string{"e2e.>"},
			MaxMsgs:      &newMax,
		}))
		require.NoError(t, err)
		assert.Equal(t, newMax, updResp.Msg.GetStream().GetConfig().GetMaxMsgs())

		// CreateConsumer (durable, explicit ack)
		consResp, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
			Name:         "e2e-consumer",
			AckPolicy:    1, // explicit
			AckWait:      durationpb.New(30 * time.Second),
		}))
		require.NoError(t, err)
		require.NotNil(t, consResp.Msg.GetConsumer())
		assert.Equal(t, "e2e-consumer", consResp.Msg.GetConsumer().GetName())
		assert.Equal(t, "e2e-consumer", consResp.Msg.GetConsumer().GetConfig().GetDurable(),
			"created consumer must be durable, otherwise the server reaps it after its ephemeral inactivity default")

		// ListConsumers contains it, still durable when read back from the server.
		listCons, err := env.management.ListConsumers(ctx, connect.NewRequest(&managementpb.ListConsumersRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
		}))
		require.NoError(t, err)
		var listed *natstypes.ConsumerInfo
		for _, c := range listCons.Msg.GetConsumers() {
			if c.GetName() == "e2e-consumer" {
				listed = c
			}
		}
		require.NotNil(t, listed, "created consumer should be listed")
		assert.Equal(t, "e2e-consumer", listed.GetConfig().GetDurable(), "durable_name must round-trip through the server")
		assert.Zero(t, listed.GetConfig().GetInactiveThreshold().AsDuration(),
			"a durable consumer must not inherit the server's 5s ephemeral inactivity default")

		// UpdateConsumer must not drop the durable name it was created with.
		newDesc := "e2e consumer updated"
		updCons, err := env.management.UpdateConsumer(ctx, connect.NewRequest(&managementpb.UpdateConsumerRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
			ConsumerName: "e2e-consumer",
			Description:  &newDesc,
		}))
		require.NoError(t, err)
		assert.Equal(t, newDesc, updCons.Msg.GetConsumer().GetConfig().GetDescription())
		assert.Equal(t, "e2e-consumer", updCons.Msg.GetConsumer().GetConfig().GetDurable(), "update must preserve durable_name")
		assert.Zero(t, updCons.Msg.GetConsumer().GetConfig().GetInactiveThreshold().AsDuration(),
			"update must not downgrade the consumer to ephemeral")

		_, err = env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
			ConnectionId:   connectionID,
			StreamName:     streamName,
			Name:           "e2e-push-consumer",
			AckPolicy:      1,
			AckWait:        durationpb.New(30 * time.Second),
			DeliverSubject: "push.e2e.deliver",
		}))
		require.NoError(t, err)

		listPush, err := env.management.ListConsumers(ctx, connect.NewRequest(&managementpb.ListConsumersRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
		}))
		require.NoError(t, err)
		var listedPush *natstypes.ConsumerInfo
		for _, c := range listPush.Msg.GetConsumers() {
			if c.GetName() == "e2e-push-consumer" {
				listedPush = c
			}
		}
		require.NotNil(t, listedPush, "push consumer must be listed alongside pull consumers")
		assert.Equal(t, "push.e2e.deliver", listedPush.GetConfig().GetDeliverSubject())
		assert.Equal(t, "e2e-push-consumer", listedPush.GetConfig().GetDurable())

		_, err = env.management.DeleteConsumer(ctx, connect.NewRequest(&managementpb.DeleteConsumerRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
			ConsumerName: "e2e-push-consumer",
		}))
		require.NoError(t, err)

		// Stats: stream stats, server info, health.
		statsResp, err := env.stats.GetStreamStats(ctx, connect.NewRequest(&statspb.GetStreamStatsRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
		}))
		require.NoError(t, err)
		require.NotNil(t, statsResp.Msg.GetStream())

		srvResp, err := env.stats.GetServerInfo(ctx, connect.NewRequest(&statspb.GetServerInfoRequest{
			ConnectionId: connectionID,
		}))
		require.NoError(t, err)
		require.NotNil(t, srvResp.Msg.GetServerInfo())
		assert.NotEmpty(t, srvResp.Msg.GetServerInfo().GetVersion())

		healthResp, err := env.stats.GetHealth(ctx, connect.NewRequest(&statspb.GetHealthRequest{
			ConnectionId: connectionID,
		}))
		require.NoError(t, err)
		require.NotNil(t, healthResp.Msg.GetHealth())

		// PurgeStream (empty stream → 0 purged is fine).
		purgeResp, err := env.management.PurgeStream(ctx, connect.NewRequest(&managementpb.PurgeStreamRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
		}))
		require.NoError(t, err)
		_ = purgeResp.Msg.GetPurged()

		// Clean up the consumer; keep the stream for the message tests.
		_, err = env.management.DeleteConsumer(ctx, connect.NewRequest(&managementpb.DeleteConsumerRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
			ConsumerName: "e2e-consumer",
		}))
		require.NoError(t, err)
	})

	t.Run("messages_and_publish", func(t *testing.T) {
		payload := `{"hello":"e2e","n":42}`

		// Publish raw JSON (no proto) → returns stream + sequence.
		pubResp, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connectionID,
			Subject:      testSubject,
			Data:         payload,
		}))
		require.NoError(t, err)
		require.Empty(t, pubResp.Msg.GetError(), "publish should not soft-fail")
		assert.Equal(t, streamName, pubResp.Msg.GetStream())
		seq := pubResp.Msg.GetSequence()
		require.Greater(t, seq, uint64(0), "publish must return a sequence")

		// ListMessages → at least 1, the published payload present.
		var listResp *connect.Response[messagespb.ListMessagesResponse]
		require.Eventually(t, func() bool {
			listResp, err = env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
				ConnectionId: connectionID,
				StreamName:   streamName,
			}))
			return err == nil && len(listResp.Msg.GetMessages()) >= 1
		}, 5*time.Second, 100*time.Millisecond, "published message should appear in the stream")
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(listResp.Msg.GetMessages()), 1)

		var found *natstypes.NatsMessage
		for _, m := range listResp.Msg.GetMessages() {
			if m.GetSubject() == testSubject {
				found = m
			}
		}
		require.NotNil(t, found, "the published subject must be present")
		recentTimestamp(t, found.GetTimestamp(), "message.timestamp")
		assert.Equal(t, payload, decodeMessagePayload(t, found))

		// GetMessage by sequence.
		getResp, err := env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
			Sequence:     seq,
		}))
		require.NoError(t, err)
		require.NotNil(t, getResp.Msg.GetMessage())
		assert.Equal(t, testSubject, getResp.Msg.GetMessage().GetSubject())
		assert.Equal(t, payload, decodeMessagePayload(t, getResp.Msg.GetMessage()))
	})

	t.Run("kv", func(t *testing.T) {
		const bucket = "E2EKV"

		_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
			ConnectionId: connectionID,
			Config: &natstypes.KVBucketConfig{
				Bucket:  bucket,
				History: 5,
			},
		}))
		require.NoError(t, err)

		putResp, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
			ConnectionId: connectionID,
			Bucket:       bucket,
			Key:          "greeting",
			Value:        "hello-kv",
		}))
		require.NoError(t, err)
		assert.Greater(t, putResp.Msg.GetRevision(), uint64(0))

		// Regression: PutKVKey used to ignore the CAS `revision` field (always
		// kv.Put). A wrong expected revision must now fail; the current one succeeds.
		casPut, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
			ConnectionId: connectionID, Bucket: bucket, Key: "cas-key", Value: "v1",
		}))
		require.NoError(t, err)
		casRev := casPut.Msg.GetRevision()
		_, casErr := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
			ConnectionId: connectionID, Bucket: bucket, Key: "cas-key", Value: "stale", Revision: casRev + 99,
		}))
		require.Error(t, casErr, "CAS put with a wrong expected revision must be rejected")
		casOK, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
			ConnectionId: connectionID, Bucket: bucket, Key: "cas-key", Value: "v2", Revision: casRev,
		}))
		require.NoError(t, err)
		assert.Equal(t, casRev+1, casOK.Msg.GetRevision())

		getResp, err := env.management.GetKVKey(ctx, connect.NewRequest(&managementpb.GetKVKeyRequest{
			ConnectionId: connectionID,
			Bucket:       bucket,
			Key:          "greeting",
		}))
		require.NoError(t, err)
		require.NotNil(t, getResp.Msg.GetEntry())
		// KV entry values come back base64-encoded on the wire; the put side
		// stored plaintext, but reads always base64-encode the stored bytes.
		decoded, err := base64.StdEncoding.DecodeString(getResp.Msg.GetEntry().GetValue())
		require.NoError(t, err)
		assert.Equal(t, "hello-kv", string(decoded))

		keysResp, err := env.management.ListKVKeys(ctx, connect.NewRequest(&managementpb.ListKVKeysRequest{
			ConnectionId: connectionID,
			Bucket:       bucket,
		}))
		require.NoError(t, err)
		assert.Contains(t, keysResp.Msg.GetKeys(), "greeting")

		bucketsResp, err := env.management.ListKVBuckets(ctx, connect.NewRequest(&managementpb.ListKVBucketsRequest{
			ConnectionId: connectionID,
		}))
		require.NoError(t, err)
		foundBucket := false
		for _, b := range bucketsResp.Msg.GetBuckets() {
			if b.GetBucket() == bucket {
				foundBucket = true
			}
		}
		assert.True(t, foundBucket, "created KV bucket should be listed")

		_, err = env.management.DeleteKVKey(ctx, connect.NewRequest(&managementpb.DeleteKVKeyRequest{
			ConnectionId: connectionID,
			Bucket:       bucket,
			Key:          "greeting",
		}))
		require.NoError(t, err)

		_, err = env.management.DeleteKVBucket(ctx, connect.NewRequest(&managementpb.DeleteKVBucketRequest{
			ConnectionId: connectionID,
			Bucket:       bucket,
		}))
		require.NoError(t, err)
	})

	t.Run("objects", func(t *testing.T) {
		const bucket = "E2EOBJ"

		_, err := env.management.CreateObjectBucket(ctx, connect.NewRequest(&managementpb.CreateObjectBucketRequest{
			ConnectionId: connectionID,
			Config: &natstypes.ObjectBucketConfig{
				Bucket: bucket,
			},
		}))
		require.NoError(t, err)

		putResp, err := env.management.PutObject(ctx, connect.NewRequest(&managementpb.PutObjectRequest{
			ConnectionId: connectionID,
			Bucket:       bucket,
			Name:         "file.txt",
			Data:         []byte("object-payload-e2e"),
		}))
		require.NoError(t, err)
		require.NotNil(t, putResp.Msg.GetInfo())
		assert.Equal(t, "file.txt", putResp.Msg.GetInfo().GetName())

		getResp, err := env.management.GetObject(ctx, connect.NewRequest(&managementpb.GetObjectRequest{
			ConnectionId: connectionID,
			Bucket:       bucket,
			Name:         "file.txt",
		}))
		require.NoError(t, err)
		assert.Equal(t, []byte("object-payload-e2e"), getResp.Msg.GetData())

		listResp, err := env.management.ListObjects(ctx, connect.NewRequest(&managementpb.ListObjectsRequest{
			ConnectionId: connectionID,
			Bucket:       bucket,
		}))
		require.NoError(t, err)
		foundObj := false
		for _, o := range listResp.Msg.GetObjects() {
			if o.GetName() == "file.txt" {
				foundObj = true
			}
		}
		assert.True(t, foundObj, "stored object should be listed")

		_, err = env.management.DeleteObject(ctx, connect.NewRequest(&managementpb.DeleteObjectRequest{
			ConnectionId: connectionID,
			Bucket:       bucket,
			Name:         "file.txt",
		}))
		require.NoError(t, err)

		_, err = env.management.DeleteObjectBucket(ctx, connect.NewRequest(&managementpb.DeleteObjectBucketRequest{
			ConnectionId: connectionID,
			Bucket:       bucket,
		}))
		require.NoError(t, err)
	})

	t.Run("mappings", func(t *testing.T) {
		createResp, err := env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
			Pattern:     "e2e.>",
			MessageType: "x.Y",
			SourceId:    "s",
		}))
		require.NoError(t, err)
		m := createResp.Msg.GetMapping()
		require.NotNil(t, m)
		require.NotEmpty(t, m.GetId())
		assert.Equal(t, "e2e.>", m.GetPattern())
		assert.Equal(t, "x.Y", m.GetMessageType())
		recentTimestamp(t, m.GetCreatedAt(), "mapping.created_at")
		mappingID := m.GetId()

		listResp, err := env.mappings.ListMappings(ctx, connect.NewRequest(&mappingspb.ListMappingsRequest{}))
		require.NoError(t, err)
		found := false
		for _, mm := range listResp.Msg.GetMappings() {
			if mm.GetId() == mappingID {
				found = true
			}
		}
		assert.True(t, found, "created mapping should be listed")

		_, err = env.mappings.DeleteMapping(ctx, connect.NewRequest(&mappingspb.DeleteMappingRequest{Id: mappingID}))
		require.NoError(t, err)
	})

	t.Run("templates", func(t *testing.T) {
		createResp, err := env.templates.CreateTemplate(ctx, connect.NewRequest(&templatespb.CreateTemplateRequest{
			Name:    "e2e-template",
			Subject: "e2e.tmpl",
			Data:    `{"k":"v"}`,
		}))
		require.NoError(t, err)
		tmpl := createResp.Msg.GetTemplate()
		require.NotNil(t, tmpl)
		require.NotEmpty(t, tmpl.GetId())
		assert.Equal(t, "e2e-template", tmpl.GetName())
		recentTimestamp(t, tmpl.GetCreatedAt(), "template.created_at")
		tmplID := tmpl.GetId()

		listResp, err := env.templates.ListTemplates(ctx, connect.NewRequest(&templatespb.ListTemplatesRequest{}))
		require.NoError(t, err)
		found := false
		for _, tt := range listResp.Msg.GetTemplates() {
			if tt.GetId() == tmplID {
				found = true
			}
		}
		assert.True(t, found, "created template should be listed")

		getResp, err := env.templates.GetTemplate(ctx, connect.NewRequest(&templatespb.GetTemplateRequest{Id: tmplID}))
		require.NoError(t, err)
		assert.Equal(t, "e2e-template", getResp.Msg.GetTemplate().GetName())

		newName := "e2e-template-renamed"
		updResp, err := env.templates.UpdateTemplate(ctx, connect.NewRequest(&templatespb.UpdateTemplateRequest{
			Id:   tmplID,
			Name: &newName,
		}))
		require.NoError(t, err)
		assert.Equal(t, newName, updResp.Msg.GetTemplate().GetName())

		_, err = env.templates.DeleteTemplate(ctx, connect.NewRequest(&templatespb.DeleteTemplateRequest{Id: tmplID}))
		require.NoError(t, err)
	})

	t.Run("settings", func(t *testing.T) {
		// GetSettings (defaults / current).
		_, err := env.settings.GetSettings(ctx, connect.NewRequest(&settingspb.GetSettingsRequest{}))
		require.NoError(t, err)

		// UpdateSettings — change a display field.
		density := "compact"
		updResp, err := env.settings.UpdateSettings(ctx, connect.NewRequest(&settingspb.UpdateSettingsRequest{
			Display: &settingstypes.DisplaySettings{
				Density: &density,
			},
		}))
		require.NoError(t, err)
		require.NotNil(t, updResp.Msg.GetSettings())
		assert.Equal(t, density, updResp.Msg.GetSettings().GetDisplay().GetDensity())

		// GetSettings reflects the change.
		getResp, err := env.settings.GetSettings(ctx, connect.NewRequest(&settingspb.GetSettingsRequest{}))
		require.NoError(t, err)
		assert.Equal(t, density, getResp.Msg.GetSettings().GetDisplay().GetDensity())

		// ResetSettings.
		_, err = env.settings.ResetSettings(ctx, connect.NewRequest(&settingspb.ResetSettingsRequest{}))
		require.NoError(t, err)
	})

	t.Run("history", func(t *testing.T) {
		// The publish in messages_and_publish recorded a history entry.
		listResp, err := env.history.ListPublishHistory(ctx, connect.NewRequest(&historypb.ListPublishHistoryRequest{}))
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(listResp.Msg.GetEntries()), 1, "publish must have recorded history")

		matched := false
		for _, e := range listResp.Msg.GetEntries() {
			if e.GetSubject() == testSubject && e.GetSuccess() {
				matched = true
				assert.Equal(t, streamName, e.GetStream())
				recentTimestamp(t, e.GetCreatedAt(), "history.created_at")
			}
		}
		assert.True(t, matched, "history should contain the successful publish to %s", testSubject)
	})

	t.Run("registry_codec_sources", func(t *testing.T) {
		// No proto sources configured → registry is empty; assert the RPCs
		// respond cleanly rather than driving a full decode here.
		statusResp, err := env.registry.GetProtoStatus(ctx, connect.NewRequest(&registrypb.GetProtoStatusRequest{}))
		require.NoError(t, err)
		assert.False(t, statusResp.Msg.GetLoaded(), "no proto sources → not loaded")
		assert.Zero(t, statusResp.Msg.GetMessageCount())

		listResp, err := env.registry.ListProtoMessages(ctx, connect.NewRequest(&registrypb.ListProtoMessagesRequest{}))
		require.NoError(t, err)
		assert.Empty(t, listResp.Msg.GetMessages(), "empty registry returns no messages")

		srcResp, err := env.sources.ListSources(ctx, connect.NewRequest(&sourcespb.ListSourcesRequest{}))
		require.NoError(t, err)
		assert.Empty(t, srcResp.Msg.GetSources(), "no proto sources configured")

		t.Log("SKIPPED full proto decode/encode: requires an external git repo or local proto source on disk; " +
			"verified registry/sources RPCs respond with an empty registry instead.")
	})

	// Final cleanup: delete the stream we kept for the message tests.
	t.Run("cleanup_stream", func(t *testing.T) {
		_, err := env.management.DeleteStream(ctx, connect.NewRequest(&managementpb.DeleteStreamRequest{
			ConnectionId: connectionID,
			StreamName:   streamName,
		}))
		require.NoError(t, err)
	})
}

func containsConnID(conns []*natstypes.SavedConnection, id string) bool {
	for _, c := range conns {
		if c.GetId() == id {
			return true
		}
	}
	return false
}

func containsStream(streams []*natstypes.StreamInfo, name string) bool {
	for _, s := range streams {
		if s.GetConfig().GetName() == name {
			return true
		}
	}
	return false
}

// decodeMessagePayload returns the raw payload of a NatsMessage. The wire form
// carries the body base64-encoded in data_base64.
func decodeMessagePayload(t *testing.T, m *natstypes.NatsMessage) string {
	t.Helper()
	if m.GetDataBase64() == "" {
		return ""
	}
	dec, err := base64.StdEncoding.DecodeString(m.GetDataBase64())
	require.NoError(t, err, "message payload must be valid base64")
	return string(dec)
}
