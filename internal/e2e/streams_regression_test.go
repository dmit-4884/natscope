// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/durationpb"

	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	statspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/stats"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// TestStreamRepublishClearDoesNotLoop checks that an empty republish update clears the config.
func TestStreamRepublishClearDoesNotLoop(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "republish-clear", env.natsURL, nil)

	const stream = "RPL"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"rpl.>"}, MaxMsgs: 50,
	}))
	require.NoError(t, err)

	for range 5 {
		_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "rpl.a", Data: "{}",
		}))
		require.NoError(t, err)
	}

	update, err := env.management.UpdateStream(ctx, connect.NewRequest(&managementpb.UpdateStreamRequest{
		ConnectionId: connID, StreamName: stream, Republish: &natstypes.RePublish{},
	}))
	require.NoError(t, err)
	assert.Nil(t, update.Msg.GetStream().GetConfig().GetRepublish(),
		"an empty republish message must clear the config, not become a '>'->'>' passthrough")

	_, err = env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
		ConnectionId: connID, Subject: "rpl.b", Data: "{}",
	}))
	require.NoError(t, err)

	stats, err := env.stats.GetStreamStats(ctx, connect.NewRequest(&statspb.GetStreamStatsRequest{
		ConnectionId: connID, StreamName: stream,
	}))
	require.NoError(t, err)
	assert.EqualValues(t, 6, stats.Msg.GetStream().GetMessages(),
		"one publish after clearing republish must add exactly one message, not trigger a self-republish loop")
}

// TestStreamConsumerLimitsInactiveThresholdPersists checks that inactive_threshold survives create and update.
func TestStreamConsumerLimitsInactiveThresholdPersists(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "consumer-limits", env.natsURL, nil)

	const stream = "CL"
	limits := &natstypes.ConsumerLimits{InactiveThreshold: durationpb.New(30 * time.Second), MaxAckPending: 100}
	create, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"cl.>"}, ConsumerLimits: limits,
	}))
	require.NoError(t, err)
	require.NotNil(t, create.Msg.GetStream().GetConfig().GetConsumerLimits())
	assert.Equal(t, 30*time.Second, create.Msg.GetStream().GetConfig().GetConsumerLimits().GetInactiveThreshold().AsDuration())

	update, err := env.management.UpdateStream(ctx, connect.NewRequest(&managementpb.UpdateStreamRequest{
		ConnectionId: connID, StreamName: stream, Description: new("x"), ConsumerLimits: limits,
	}))
	require.NoError(t, err)
	require.NotNil(t, update.Msg.GetStream().GetConfig().GetConsumerLimits())
	assert.Equal(t, 30*time.Second, update.Msg.GetStream().GetConfig().GetConsumerLimits().GetInactiveThreshold().AsDuration(),
		"inactive_threshold must survive an update, not be silently dropped")
}

// TestStreamSubjectTransformRoundTrips checks that subject_transform is returned and survives Update.
func TestStreamSubjectTransformRoundTrips(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "subject-transform", env.natsURL, nil)

	const stream = "TR"
	create, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"tr.>"},
		SubjectTransform: &managementpb.SubjectTransform{Source: "tr.>", Destination: "trx.>"},
	}))
	require.NoError(t, err)
	created := create.Msg.GetStream().GetConfig().GetSubjectTransform()
	require.NotNil(t, created, "subject_transform must be present in the create response")
	assert.Equal(t, "tr.>", created.GetSource())
	assert.Equal(t, "trx.>", created.GetDestination())

	got, err := env.streams.GetStream(ctx, connect.NewRequest(&streamspb.GetStreamRequest{
		ConnectionId: connID, StreamName: stream,
	}))
	require.NoError(t, err)
	require.NotNil(t, got.Msg.GetStream().GetConfig().GetSubjectTransform())

	update, err := env.management.UpdateStream(ctx, connect.NewRequest(&managementpb.UpdateStreamRequest{
		ConnectionId: connID, StreamName: stream, Description: new("edited"),
		SubjectTransform: got.Msg.GetStream().GetConfig().GetSubjectTransform(),
	}))
	require.NoError(t, err)
	updated := update.Msg.GetStream().GetConfig().GetSubjectTransform()
	require.NotNil(t, updated, "round-tripping the current subject_transform back through Update must not drop it")
	assert.Equal(t, "tr.>", updated.GetSource())
	assert.Equal(t, "trx.>", updated.GetDestination())
}

// TestStreamSourcesReplaceNotAppend checks that Update replaces the sources list.
func TestStreamSourcesReplaceNotAppend(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "sources-replace", env.natsURL, nil)

	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: "ORIG", Storage: 1, Subjects: []string{"orig.>"},
	}))
	require.NoError(t, err)

	_, err = env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: "AGG", Storage: 1,
		Sources: []*managementpb.StreamSourceConfig{{Name: "ORIG"}},
	}))
	require.NoError(t, err)

	update, err := env.management.UpdateStream(ctx, connect.NewRequest(&managementpb.UpdateStreamRequest{
		ConnectionId: connID, StreamName: "AGG", Description: new("edited"),
		Sources: []*managementpb.StreamSourceConfig{{Name: "ORIG"}},
	}))
	require.NoError(t, err, "resending the current sources list must replace, not append and duplicate")
	assert.Len(t, update.Msg.GetStream().GetConfig().GetSources(), 1)
}

// TestConsumerUpdateKeepsBackoffWhenOmitted checks that an update without backoff keeps the existing one.
func TestConsumerUpdateKeepsBackoffWhenOmitted(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "consumer-backoff", env.natsURL, nil)

	const stream = "CB"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"cb.>"},
	}))
	require.NoError(t, err)

	create, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
		ConnectionId: connID, StreamName: stream, Name: "c", MaxDeliver: 5,
		BackOff: []*durationpb.Duration{durationpb.New(time.Second), durationpb.New(2 * time.Second)},
	}))
	require.NoError(t, err)
	require.Len(t, create.Msg.GetConsumer().GetConfig().GetBackOff(), 2)

	update, err := env.management.UpdateConsumer(ctx, connect.NewRequest(&managementpb.UpdateConsumerRequest{
		ConnectionId: connID, StreamName: stream, ConsumerName: "c", Description: new("only description"),
	}))
	require.NoError(t, err)
	backoff := update.Msg.GetConsumer().GetConfig().GetBackOff()
	require.Len(t, backoff, 2, "an update that doesn't mention back_off must not wipe the existing backoff")
	assert.Equal(t, time.Second, backoff[0].AsDuration())
	assert.Equal(t, 2*time.Second, backoff[1].AsDuration())
}

// TestStreamDollarPrefixRejected checks that a "$"-prefixed stream name is rejected.
func TestStreamDollarPrefixRejected(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "dollar-prefix", env.natsURL, nil)

	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: "$hidden", Subjects: []string{"hid.>"},
	}))
	require.Error(t, err, "a $-prefixed stream would be invisible to ListStreams/GetAllStreamsStats")
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// TestConsumerBlankNameRejected checks that a whitespace-only durable name is rejected.
func TestConsumerBlankNameRejected(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "blank-consumer-name", env.natsURL, nil)

	const stream = "BN"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"bn.>"},
	}))
	require.NoError(t, err)

	_, err = env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
		ConnectionId: connID, StreamName: stream, Name: "   ",
	}))
	require.Error(t, err, "a whitespace-only name must be rejected, not silently create an ephemeral consumer")
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// TestMessageMultiValueHeadersPreserved checks that repeated headers keep every value on list and get.
func TestMessageMultiValueHeadersPreserved(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "multi-headers", env.natsURL, nil)

	const stream = "HD"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"hd.>"},
	}))
	require.NoError(t, err)

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)

	msg := &nats.Msg{Subject: "hd.a", Data: []byte("hdr-msg"), Header: nats.Header{}}
	msg.Header.Add("X-Multi", "one")
	msg.Header.Add("X-Multi", "two")
	msg.Header.Set("Other", "v")
	_, err = js.PublishMsg(ctx, msg)
	require.NoError(t, err)

	listResp, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
		ConnectionId: connID, StreamName: stream,
		Direction: messagespb.Direction_DIRECTION_FORWARD, StartSeq: new(uint64(1)),
	}))
	require.NoError(t, err)
	require.Len(t, listResp.Msg.GetMessages(), 1)
	assert.Equal(t, "one, two", listResp.Msg.GetMessages()[0].GetHeaders()["X-Multi"],
		"ListMessages must keep every value of a repeated header")

	getResp, err := env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
		ConnectionId: connID, StreamName: stream, Sequence: 1,
	}))
	require.NoError(t, err)
	assert.Equal(t, "one, two", getResp.Msg.GetMessage().GetHeaders()["X-Multi"],
		"GetMessage must keep every value of a repeated header too")
}

// TestStreamNameNotSilentlyTrimmed checks that a padded stream name is rejected, not trimmed.
func TestStreamNameNotSilentlyTrimmed(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "padded-name", env.natsURL, nil)

	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: " padded ", Subjects: []string{"pad.>"},
	}))
	require.Error(t, err, "a padded name must be rejected, not silently trimmed")
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// TestConsumerOptStartTimeReflectedInResponses checks that CreateConsumer and ListConsumers return opt_start_time.
func TestConsumerOptStartTimeReflectedInResponses(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "opt-start-time", env.natsURL, nil)

	const stream = "OT"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"ot.>"},
	}))
	require.NoError(t, err)

	const optStartTime = "2026-01-01T00:00:00Z"
	create, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
		ConnectionId: connID, StreamName: stream, Name: "t", DeliverPolicy: 4, OptStartTime: optStartTime,
	}))
	require.NoError(t, err)
	assert.Equal(t, optStartTime, create.Msg.GetConsumer().GetConfig().GetOptStartTime(),
		"CreateConsumer's own response must echo opt_start_time back")

	list, err := env.management.ListConsumers(ctx, connect.NewRequest(&managementpb.ListConsumersRequest{
		ConnectionId: connID, StreamName: stream,
	}))
	require.NoError(t, err)
	require.Len(t, list.Msg.GetConsumers(), 1)
	assert.Equal(t, optStartTime, list.Msg.GetConsumers()[0].GetConfig().GetOptStartTime())
}

// TestConsumerPauseValidationAndPrecision checks pause_until format errors and fractional-second precision.
func TestConsumerPauseValidationAndPrecision(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "pause-validation", env.natsURL, nil)

	const stream = "PZ"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"pz.>"},
	}))
	require.NoError(t, err)
	_, err = env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
		ConnectionId: connID, StreamName: stream, Name: "u",
	}))
	require.NoError(t, err)

	_, err = env.management.PauseConsumer(ctx, connect.NewRequest(&managementpb.PauseConsumerRequest{
		ConnectionId: connID, StreamName: stream, ConsumerName: "u", PauseUntil: "2030-01-01",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Contains(t, err.Error(), "RFC3339", "the rejection must name the expected format")

	resp, err := env.management.PauseConsumer(ctx, connect.NewRequest(&managementpb.PauseConsumerRequest{
		ConnectionId: connID, StreamName: stream, ConsumerName: "u", PauseUntil: "2030-01-01T00:00:00.5Z",
	}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.PauseUntil)
	assert.Equal(t, "2030-01-01T00:00:00.5Z", *resp.Msg.PauseUntil, "fractional seconds must not be truncated")
}

// TestStreamStatsSubjectsPopulated checks that GetStreamStats reports per-subject message counts.
func TestStreamStatsSubjectsPopulated(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "stream-subjects-stats", env.natsURL, nil)

	const stream = "SF"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"sf.>"},
	}))
	require.NoError(t, err)

	for _, subj := range []string{"sf.a", "sf.b", "sf.c"} {
		_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: subj, Data: "{}",
		}))
		require.NoError(t, err)
	}

	stats, err := env.stats.GetStreamStats(ctx, connect.NewRequest(&statspb.GetStreamStatsRequest{
		ConnectionId: connID, StreamName: stream,
	}))
	require.NoError(t, err)
	assert.Len(t, stats.Msg.GetStream().GetSubjects(), 3, "per-subject message counts must be populated")
	assert.EqualValues(t, 1, stats.Msg.GetStream().GetSubjects()["sf.a"])
}
