// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

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

	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
)

// newFeatureConnection creates a saved connection to the harness NATS server.
func newFeatureConnection(t *testing.T, env *e2eEnv, name string) string {
	t.Helper()
	resp, err := env.connections.CreateConnection(t.Context(), connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: name, Urls: []string{env.natsURL},
	}))
	require.NoError(t, err)
	return resp.Msg.GetConnection().GetId()
}

// TestStreamFeatureFlags creates and updates streams with the NATS 2.11–2.14
// flags against the embedded server.
func TestStreamFeatureFlags(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := newFeatureConnection(t, env, "stream-flags")

	t.Run("counter stream", func(t *testing.T) {
		resp, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: "FLAGS_COUNTER", Subjects: []string{"flags.counter.>"}, AllowMsgCounter: true,
		}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.GetStream().GetConfig().GetAllowMsgCounter())

		_, err = env.management.UpdateStream(ctx, connect.NewRequest(&managementpb.UpdateStreamRequest{
			ConnectionId: connID, StreamName: "FLAGS_COUNTER", Description: new("still a counter"),
		}))
		require.NoError(t, err, "an unrelated update must keep the counter flag the server refuses to change")
	})

	t.Run("schedules, delete markers, async persist and fast batch", func(t *testing.T) {
		resp, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId:           connID,
			Name:                   "FLAGS_SCHED",
			Subjects:               []string{"flags.sched.>"},
			AllowMsgSchedules:      true,
			SubjectDeleteMarkerTtl: durationpb.New(time.Minute),
			PersistMode:            1,
			AllowBatchPublish:      true,
		}))
		require.NoError(t, err)
		cfg := resp.Msg.GetStream().GetConfig()
		assert.True(t, cfg.GetAllowMsgSchedules())
		assert.Equal(t, time.Minute, cfg.GetSubjectDeleteMarkerTtl().AsDuration())
		assert.True(t, cfg.GetAllowMsgTtl(), "delete markers switch on per-message TTL")
		assert.Equal(t, int32(1), cfg.GetPersistMode())
		assert.True(t, cfg.GetAllowBatchPublish())

		upd, err := env.management.UpdateStream(ctx, connect.NewRequest(&managementpb.UpdateStreamRequest{
			ConnectionId: connID, StreamName: "FLAGS_SCHED",
			SubjectDeleteMarkerTtl: durationpb.New(2 * time.Minute), AllowBatchPublish: new(false),
		}))
		require.NoError(t, err)
		ucfg := upd.Msg.GetStream().GetConfig()
		assert.Equal(t, 2*time.Minute, ucfg.GetSubjectDeleteMarkerTtl().AsDuration())
		assert.False(t, ucfg.GetAllowBatchPublish())
		assert.True(t, ucfg.GetAllowMsgSchedules())
		assert.Equal(t, int32(1), ucfg.GetPersistMode())

		_, err = env.management.UpdateStream(ctx, connect.NewRequest(&managementpb.UpdateStreamRequest{
			ConnectionId: connID, StreamName: "FLAGS_SCHED", AllowMsgSchedules: new(false),
		}))
		require.Error(t, err, "the server refuses to disable schedules")
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("incompatible combination surfaces the server reason", func(t *testing.T) {
		_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: "FLAGS_BAD", Subjects: []string{"flags.bad.>"},
			AllowMsgCounter: true, AllowMsgTtl: true,
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		assert.Contains(t, err.Error(), "counter stream cannot use message TTLs")
	})
}

// TestConsumerReset resets consumers against the embedded 2.14 server.
func TestConsumerReset(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := newFeatureConnection(t, env, "consumer-reset")

	const stream = "RESET_FLOW"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Subjects: []string{"reset.>"},
	}))
	require.NoError(t, err)
	for range 5 {
		_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "reset.msg", Data: `{"n":1}`,
		}))
		require.NoError(t, err)
	}

	_, err = env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
		ConnectionId: connID, StreamName: stream, Name: "replay",
	}))
	require.NoError(t, err)

	t.Run("without sequence keeps the ack floor and redelivers unacked messages", func(t *testing.T) {
		nc, err := nats.Connect(env.natsURL)
		require.NoError(t, err)
		t.Cleanup(nc.Close)
		js, err := jetstream.New(nc)
		require.NoError(t, err)
		cons, err := js.Consumer(ctx, stream, "replay")
		require.NoError(t, err)

		acked, err := cons.Fetch(2, jetstream.FetchMaxWait(2*time.Second))
		require.NoError(t, err)
		for msg := range acked.Messages() {
			require.NoError(t, msg.DoubleAck(ctx))
		}
		unacked, err := cons.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
		require.NoError(t, err)
		for range unacked.Messages() {
		}

		resp, err := env.management.ResetConsumer(ctx, connect.NewRequest(&managementpb.ResetConsumerRequest{
			ConnectionId: connID, StreamName: stream, ConsumerName: "replay",
		}))
		require.NoError(t, err)
		assert.Equal(t, "replay", resp.Msg.GetConsumer().GetName())
		assert.Equal(t, uint64(3), resp.Msg.GetResetSeq(), "delivery restarts right after the ack floor")
		assert.Zero(t, resp.Msg.GetConsumer().GetNumAckPending(), "the unacked message is no longer pending ack")
		assert.Equal(t, uint64(3), resp.Msg.GetConsumer().GetNumPending(), "messages 3-5 are delivered again")
	})

	t.Run("to a sequence skips earlier messages", func(t *testing.T) {
		resp, err := env.management.ResetConsumer(ctx, connect.NewRequest(&managementpb.ResetConsumerRequest{
			ConnectionId: connID, StreamName: stream, ConsumerName: "replay", Sequence: new(uint64(3)),
		}))
		require.NoError(t, err)
		assert.Equal(t, uint64(3), resp.Msg.GetResetSeq())
		assert.Equal(t, uint64(3), resp.Msg.GetConsumer().GetNumPending())
	})

	t.Run("sequence reset on deliver last is rejected by the server", func(t *testing.T) {
		_, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
			ConnectionId: connID, StreamName: stream, Name: "latest", DeliverPolicy: 1,
		}))
		require.NoError(t, err)

		_, err = env.management.ResetConsumer(ctx, connect.NewRequest(&managementpb.ResetConsumerRequest{
			ConnectionId: connID, StreamName: stream, ConsumerName: "latest", Sequence: new(uint64(2)),
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		assert.Contains(t, err.Error(), "deliver policy", "the user learns why the reset was refused")
	})

	t.Run("sequence below the start sequence is rejected with the reason", func(t *testing.T) {
		_, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
			ConnectionId: connID, StreamName: stream, Name: "from-four", DeliverPolicy: 3, OptStartSeq: 4,
		}))
		require.NoError(t, err)

		_, err = env.management.ResetConsumer(ctx, connect.NewRequest(&managementpb.ResetConsumerRequest{
			ConnectionId: connID, StreamName: stream, ConsumerName: "from-four", Sequence: new(uint64(2)),
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		assert.Contains(t, err.Error(), "start sequence")
	})

	t.Run("unknown consumer", func(t *testing.T) {
		_, err := env.management.ResetConsumer(ctx, connect.NewRequest(&managementpb.ResetConsumerRequest{
			ConnectionId: connID, StreamName: stream, ConsumerName: "ghost",
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
}

// TestConsumerPriorityGroups creates a pinned-client consumer, lets a client
// pin itself and unpins it through the API.
func TestConsumerPriorityGroups(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := newFeatureConnection(t, env, "priority-groups")

	const stream = "PRIO_FLOW"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Subjects: []string{"prio.>"},
	}))
	require.NoError(t, err)
	for range 3 {
		_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "prio.job", Data: `{"n":1}`,
		}))
		require.NoError(t, err)
	}

	created, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
		ConnectionId: connID, StreamName: stream, Name: "pinned",
		PriorityPolicy: 1, PriorityGroups: []string{"jobs"}, PinnedTtl: durationpb.New(time.Minute),
	}))
	require.NoError(t, err)
	cfg := created.Msg.GetConsumer().GetConfig()
	assert.Equal(t, int32(1), cfg.GetPriorityPolicy())
	assert.Equal(t, []string{"jobs"}, cfg.GetPriorityGroups())
	assert.Equal(t, time.Minute, cfg.GetPinnedTtl().AsDuration())

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	cons, err := js.Consumer(ctx, stream, "pinned")
	require.NoError(t, err)
	batch, err := cons.Fetch(1, jetstream.FetchPriorityGroup("jobs"), jetstream.FetchMaxWait(2*time.Second))
	require.NoError(t, err)
	for msg := range batch.Messages() {
		require.NoError(t, msg.Ack())
	}
	require.NoError(t, batch.Error())

	pinnedID := func() string {
		resp, err := env.management.ListConsumers(ctx, connect.NewRequest(&managementpb.ListConsumersRequest{
			ConnectionId: connID, StreamName: stream,
		}))
		require.NoError(t, err)
		for _, c := range resp.Msg.GetConsumers() {
			if c.GetName() == "pinned" && len(c.GetPriorityGroups()) > 0 {
				return c.GetPriorityGroups()[0].GetPinnedClientId()
			}
		}
		return ""
	}
	require.NotEmpty(t, pinnedID(), "the fetching client should be pinned")

	_, err = env.management.UnpinConsumer(ctx, connect.NewRequest(&managementpb.UnpinConsumerRequest{
		ConnectionId: connID, StreamName: stream, ConsumerName: "pinned", Group: "jobs",
	}))
	require.NoError(t, err)
	assert.Empty(t, pinnedID(), "unpin should release the pinned client")

	t.Run("switching the policy off clears the groups", func(t *testing.T) {
		upd, err := env.management.UpdateConsumer(ctx, connect.NewRequest(&managementpb.UpdateConsumerRequest{
			ConnectionId: connID, StreamName: stream, ConsumerName: "pinned", PriorityPolicy: new(int32(0)),
		}))
		require.NoError(t, err)
		assert.Zero(t, upd.Msg.GetConsumer().GetConfig().GetPriorityPolicy())
		assert.Empty(t, upd.Msg.GetConsumer().GetConfig().GetPriorityGroups())
	})

	t.Run("invalid group name is rejected before reaching the server", func(t *testing.T) {
		_, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
			ConnectionId: connID, StreamName: stream, Name: "bad",
			PriorityPolicy: 1, PriorityGroups: []string{"not a group!"},
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("unpin on an unknown consumer", func(t *testing.T) {
		_, err := env.management.UnpinConsumer(ctx, connect.NewRequest(&managementpb.UnpinConsumerRequest{
			ConnectionId: connID, StreamName: stream, ConsumerName: "ghost", Group: "jobs",
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
}

// TestPublishFeatureHeaders publishes counter increments, per-message TTLs and
// delayed schedules through the publish API against the embedded server.
func TestPublishFeatureHeaders(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := newFeatureConnection(t, env, "publish-features")

	publish := func(subject, data string, headers map[string]string) *publishpb.PublishMessageResponse {
		t.Helper()
		resp, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: subject, Data: data, Headers: headers,
		}))
		require.NoError(t, err)
		return resp.Msg
	}

	t.Run("counter increments return the running total", func(t *testing.T) {
		_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: "PUB_COUNTER", Subjects: []string{"hits.>"}, AllowMsgCounter: true,
		}))
		require.NoError(t, err)

		first := publish("hits.page", "", map[string]string{"Nats-Incr": "+5"})
		require.Nil(t, first.Error)
		assert.Equal(t, "5", first.GetCounterValue())

		second := publish("hits.page", "", map[string]string{"Nats-Incr": "-2"})
		require.Nil(t, second.Error)
		assert.Equal(t, "3", second.GetCounterValue())

		rejected := publish("hits.page", `{"x":1}`, nil)
		require.NotNil(t, rejected.Error, "a counter stream refuses messages without Nats-Incr")
	})

	t.Run("empty data needs Nats-Incr", func(t *testing.T) {
		_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "hits.page",
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("per-message ttl", func(t *testing.T) {
		_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: "PUB_TTL", Subjects: []string{"ttl.>"}, AllowMsgTtl: true,
		}))
		require.NoError(t, err)
		_, err = env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: "PUB_NO_TTL", Subjects: []string{"nottl.>"},
		}))
		require.NoError(t, err)

		ok := publish("ttl.one", `{"n":1}`, map[string]string{"Nats-TTL": "1h"})
		require.Nil(t, ok.Error)
		assert.NotZero(t, ok.GetSequence())

		refused := publish("nottl.one", `{"n":1}`, map[string]string{"Nats-TTL": "1h"})
		require.NotNil(t, refused.Error, "a stream without allow_msg_ttl refuses Nats-TTL")
	})

	t.Run("delayed schedule publishes to its target", func(t *testing.T) {
		_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: "PUB_SCHED", Subjects: []string{"sched.>"}, AllowMsgSchedules: true,
		}))
		require.NoError(t, err)

		at := time.Now().Add(time.Second).UTC().Format(time.RFC3339)
		resp := publish("sched.job.1", `{"run":true}`, map[string]string{
			"Nats-Schedule":        "@at " + at,
			"Nats-Schedule-Target": "sched.target",
		})
		require.Nil(t, resp.Error)

		nc, err := nats.Connect(env.natsURL)
		require.NoError(t, err)
		t.Cleanup(nc.Close)
		js, err := jetstream.New(nc)
		require.NoError(t, err)
		stream, err := js.Stream(ctx, "PUB_SCHED")
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			msg, err := stream.GetLastMsgForSubject(ctx, "sched.target")
			return err == nil && string(msg.Data) == `{"run":true}`
		}, 10*time.Second, 200*time.Millisecond, "the scheduled message should reach its target")
	})
}
