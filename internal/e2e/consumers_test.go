// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	livepb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/live"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	statspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/stats"
	settingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/settings/v1/settings"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
	settingstypes "github.com/dmit-4884/natscope/proto/gen/types/settings"
)

func jetStreamFor(t *testing.T, url string, opts ...nats.Option) jetstream.JetStream {
	t.Helper()
	nc, err := nats.Connect(url, opts...)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	return js
}

func addStream(t *testing.T, js jetstream.JetStream, name string, subjects ...string) jetstream.Stream {
	t.Helper()
	stream, err := js.CreateStream(t.Context(), jetstream.StreamConfig{Name: name, Subjects: subjects})
	require.NoError(t, err)
	return stream
}

func publishAll(t *testing.T, js jetstream.JetStream, subjects ...string) {
	t.Helper()
	for _, subject := range subjects {
		_, err := js.Publish(t.Context(), subject, []byte(`{"subject":"`+subject+`"}`))
		require.NoError(t, err)
	}
}

func consumersOverview(t *testing.T, env *e2eEnv, connID string) *statspb.GetAllConsumersResponse {
	t.Helper()
	resp, err := env.stats.GetAllConsumers(t.Context(), connect.NewRequest(&statspb.GetAllConsumersRequest{ConnectionId: connID}))
	require.NoError(t, err)
	return resp.Msg
}

func findConsumer(t *testing.T, consumers []*natstypes.ConsumerInfo, stream, name string) *natstypes.ConsumerInfo {
	t.Helper()
	for _, c := range consumers {
		if c.GetStream() == stream && c.GetName() == name {
			return c
		}
	}
	t.Fatalf("consumer %s on %s not listed", name, stream)
	return nil
}

func TestConsumersOverview(t *testing.T) {
	env := setupE2E(t)
	connID := createTestConnection(t, env, "consumers", env.natsURL, nil)
	js := jetStreamFor(t, env.natsURL)
	ctx := t.Context()

	stream := addStream(t, js, "OVERVIEW", "overview.>")
	addStream(t, js, "IDLE", "idle.>")
	billing, err := stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
		Durable: "billing", FilterSubject: "overview.created", AckPolicy: jetstream.AckExplicitPolicy,
	})
	require.NoError(t, err)
	_, err = stream.CreateConsumer(ctx, jetstream.ConsumerConfig{Name: "natscope-browse-e2e", AckPolicy: jetstream.AckNonePolicy})
	require.NoError(t, err)
	_, err = stream.CreateConsumer(ctx, jetstream.ConsumerConfig{Durable: "sleeper", AckPolicy: jetstream.AckExplicitPolicy})
	require.NoError(t, err)
	pauseUntil := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	_, err = stream.PauseConsumer(ctx, "sleeper", pauseUntil)
	require.NoError(t, err)
	publishAll(t, js, "overview.created", "overview.paid", "overview.created", "overview.created")

	batch, err := billing.Fetch(2, jetstream.FetchMaxWait(2*time.Second))
	require.NoError(t, err)
	var fetched []jetstream.Msg
	for msg := range batch.Messages() {
		fetched = append(fetched, msg)
	}
	require.Len(t, fetched, 2)
	require.NoError(t, fetched[0].DoubleAck(ctx))

	got := consumersOverview(t, env, connID)

	c := findConsumer(t, got.GetConsumers(), "OVERVIEW", "billing")
	assert.Empty(t, c.GetRaw(), "the overview leaves the raw JSON out")
	assert.Equal(t, uint64(1), c.GetNumPending())
	assert.Equal(t, int32(1), c.GetNumAckPending())
	assert.Equal(t, uint64(3), c.GetDelivered().GetStream())
	assert.Equal(t, uint64(2), c.GetAckFloor().GetStream(), "the floor moves past messages outside the filter")
	recentTimestamp(t, c.GetDelivered().GetLastActive(), "delivered.last_active")
	recentTimestamp(t, c.GetAckFloor().GetLastActive(), "ack_floor.last_active")
	assert.Equal(t, "overview.created", c.GetConfig().GetFilterSubject())
	findConsumer(t, got.GetConsumers(), "OVERVIEW", "natscope-browse-e2e")
	sleeper := findConsumer(t, got.GetConsumers(), "OVERVIEW", "sleeper")
	assert.True(t, sleeper.GetPaused())
	assert.Equal(t, pauseUntil, sleeper.GetPauseUntil().AsTime())
	assert.False(t, c.GetPaused())
	for _, s := range got.GetStreams() {
		assert.Empty(t, s.GetRaw(), "the overview leaves the raw JSON out")
	}

	names := make([]string, 0, len(got.GetStreams()))
	for _, s := range got.GetStreams() {
		names = append(names, s.GetConfig().GetName())
	}
	assert.Subset(t, names, []string{"OVERVIEW", "IDLE"}, "streams without consumers are listed too")
	assert.Empty(t, got.GetUnreadableStreams())
}

func TestConsumersOverviewAccess(t *testing.T) {
	env := setupE2E(t)
	srv := startNATSWithOptions(t, func(o *server.Options) {
		o.JetStream = true
		o.StoreDir = t.TempDir()
		o.Users = []*server.User{
			{Username: "admin", Password: "pw"},
			{Username: "u", Password: "pw", Permissions: &server.Permissions{
				Publish: &server.SubjectPermission{Deny: []string{"$JS.API.CONSUMER.LIST.SECRET"}},
			}},
			{Username: "blind", Password: "pw", Permissions: &server.Permissions{
				Publish: &server.SubjectPermission{Deny: []string{"$JS.API.STREAM.LIST"}},
			}},
		}
	})
	admin := jetStreamFor(t, srv.ClientURL(), nats.UserInfo("admin", "pw"))
	for _, name := range []string{"OPEN", "SECRET"} {
		stream := addStream(t, admin, name, name+".>")
		_, err := stream.CreateConsumer(t.Context(), jetstream.ConsumerConfig{Durable: "worker", AckPolicy: jetstream.AckExplicitPolicy})
		require.NoError(t, err)
	}
	login := func(user string) *natstypes.AuthConfig {
		return &natstypes.AuthConfig{Method: natstypes.AuthMethod_AUTH_METHOD_USER_PASSWORD, Username: new(user), Password: new("pw")}
	}

	t.Run("a stream with unreadable consumers is reported, the rest are listed", func(t *testing.T) {
		connID := createTestConnection(t, env, "consumers-partial", srv.ClientURL(), login("u"))
		start := time.Now()
		got := consumersOverview(t, env, connID)
		assert.Less(t, time.Since(start), 3*time.Second, "a denied listing must fail fast")

		findConsumer(t, got.GetConsumers(), "OPEN", "worker")
		for _, c := range got.GetConsumers() {
			assert.NotEqual(t, "SECRET", c.GetStream())
		}
		require.Len(t, got.GetUnreadableStreams(), 1)
		unreadable := got.GetUnreadableStreams()[0]
		assert.Equal(t, "SECRET", unreadable.GetStream())
		assert.Equal(t, natstypes.AccessStatus_ACCESS_STATUS_DENIED, unreadable.GetAccess().GetStatus())
		assert.Equal(t, "publish", unreadable.GetAccess().GetOperation())
		assert.Equal(t, "$JS.API.CONSUMER.LIST.SECRET", unreadable.GetAccess().GetSubject())
	})

	t.Run("no permission to list streams is no access", func(t *testing.T) {
		connID := createTestConnection(t, env, "consumers-blind", srv.ClientURL(), login("blind"))
		start := time.Now()
		_, err := env.stats.GetAllConsumers(t.Context(), connect.NewRequest(&statspb.GetAllConsumersRequest{ConnectionId: connID}))
		require.Error(t, err)
		assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
		assert.Less(t, time.Since(start), 3*time.Second, "a denied listing must fail fast")
		assert.Equal(t, "$JS.API.STREAM.LIST", errorMetadata(t, err)["subject"])
	})
}

func TestGetNextMessage(t *testing.T) {
	env := setupE2E(t)
	connID := createTestConnection(t, env, "next-message", env.natsURL, nil)
	js := jetStreamFor(t, env.natsURL)
	addStream(t, js, "NEXT", "next.>")
	publishAll(t, js, "next.created", "next.paid", "next.created", "next.shipped")

	next := func(t *testing.T, start uint64, subjects ...string) *natstypes.NatsMessage {
		t.Helper()
		resp, err := env.messages.GetNextMessage(t.Context(), connect.NewRequest(&messagespb.GetNextMessageRequest{
			ConnectionId: connID, StreamName: "NEXT", StartSeq: start, Subjects: subjects,
		}))
		require.NoError(t, err)
		return resp.Msg.Message
	}

	tests := []struct {
		name     string
		start    uint64
		subjects []string
		wantSeq  uint64
	}{
		{name: "the start sequence itself counts", start: 2, subjects: []string{"next.paid"}, wantSeq: 2},
		{name: "skips other subjects", start: 2, subjects: []string{"next.created"}, wantSeq: 3},
		{name: "the earliest match of several filters", start: 2, subjects: []string{"next.shipped", "next.created"}, wantSeq: 3},
		{name: "wildcards match", start: 4, subjects: []string{"next.*"}, wantSeq: 4},
		{name: "no filters match every subject", start: 2, wantSeq: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := next(t, tt.start, tt.subjects...)
			require.NotNil(t, msg)
			assert.Equal(t, tt.wantSeq, msg.GetSequence())
		})
	}

	t.Run("nothing after the start sequence is an empty answer", func(t *testing.T) {
		assert.Nil(t, next(t, 3, "next.paid"))
		assert.Nil(t, next(t, 99))
	})

	t.Run("the message comes with its payload", func(t *testing.T) {
		msg := next(t, 1, "next.paid")
		require.NotNil(t, msg)
		assert.Equal(t, "next.paid", msg.GetSubject())
		assert.JSONEq(t, `{"subject":"next.paid"}`, decodeMessagePayload(t, msg))
	})

	t.Run("a malformed subject is rejected, not answered with nothing", func(t *testing.T) {
		for _, subject := range []string{"next. bad", "next..x", ">.foo"} {
			_, err := env.messages.GetNextMessage(t.Context(), connect.NewRequest(&messagespb.GetNextMessageRequest{
				ConnectionId: connID, StreamName: "NEXT", StartSeq: 1, Subjects: []string{subject},
			}))
			require.Error(t, err, subject)
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), subject)
		}
	})

	t.Run("an unknown stream is not found", func(t *testing.T) {
		_, err := env.messages.GetNextMessage(t.Context(), connect.NewRequest(&messagespb.GetNextMessageRequest{
			ConnectionId: connID, StreamName: "MISSING", StartSeq: 1,
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
}

func TestConsumersOverviewHidesOnlyItsOwnLiveConsumer(t *testing.T) {
	env := setupE2E(t)
	connID := createTestConnection(t, env, "consumers-own", env.natsURL, nil)
	js := jetStreamFor(t, env.natsURL)
	ctx := t.Context()
	_, err := env.settings.UpdateSettings(ctx, connect.NewRequest(&settingspb.UpdateSettingsRequest{
		Live: &settingstypes.LiveSettings{SubscriptionMode: new("jetstream_ordered")},
	}))
	require.NoError(t, err)
	stream := addStream(t, js, "OWN", "own.>")
	_, err = stream.CreateConsumer(ctx, jetstream.ConsumerConfig{Name: "natscope-live-impostor", AckPolicy: jetstream.AckNonePolicy})
	require.NoError(t, err)

	subCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	stopPublishing := make(chan struct{})
	defer close(stopPublishing)
	go func() {
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopPublishing:
				return
			case <-subCtx.Done():
				return
			case <-ticker.C:
				_, _ = js.Publish(subCtx, "own.tick", []byte(`{}`))
			}
		}
	}()
	sub, err := env.live.Subscribe(subCtx, connect.NewRequest(&livepb.SubscribeRequest{
		ConnectionId:  connID,
		Subscriptions: []*livepb.LiveSubscription{{Subject: "own.>", StreamName: new("OWN")}},
	}))
	require.NoError(t, err)
	defer sub.Close()
	require.True(t, sub.Receive(), "the live session starts")

	var live string
	require.Eventually(t, func() bool {
		for name := range stream.ConsumerNames(ctx).Name() {
			if strings.HasPrefix(name, "natscope-live-") && name != "natscope-live-impostor" {
				live = name
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "the live session reads the stream through its own consumer")

	got := consumersOverview(t, env, connID)
	findConsumer(t, got.GetConsumers(), "OWN", "natscope-live-impostor")
	for _, c := range got.GetConsumers() {
		assert.NotEqual(t, live, c.GetName(), "natscope's own live consumer is not the user's")
	}
}
