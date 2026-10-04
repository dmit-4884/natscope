// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	livepb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/live"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

func openLive(t *testing.T, env *e2eEnv, connID string, subjects ...string) <-chan *livepb.LiveEvent {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	subscriptions := make([]*livepb.LiveSubscription, 0, len(subjects))
	for _, s := range subjects {
		subscriptions = append(subscriptions, &livepb.LiveSubscription{Subject: s})
	}

	events := make(chan *livepb.LiveEvent, 64)
	go func() {
		defer close(events)
		stream, err := env.live.Subscribe(ctx, connect.NewRequest(&livepb.SubscribeRequest{
			ConnectionId:  connID,
			Subscriptions: subscriptions,
		}))
		if err != nil {
			return
		}
		defer stream.Close()
		for stream.Receive() {
			select {
			case events <- stream.Msg():
			case <-ctx.Done():
				return
			}
		}
	}()
	return events
}

func waitLive(t *testing.T, events <-chan *livepb.LiveEvent, timeout time.Duration, match func(*livepb.LiveEvent) bool) *livepb.LiveEvent {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-events:
			require.True(t, ok, "live session ended before a matching event")
			if match(ev) {
				return ev
			}
		case <-deadline:
			t.Fatal("no matching live event in time")
			return nil
		}
	}
}

func batchMessage(subject string, out **natstypes.NatsMessage) func(*livepb.LiveEvent) bool {
	return func(ev *livepb.LiveEvent) bool {
		for _, m := range ev.GetBatch().GetMessages() {
			if m.GetSubject() == subject {
				*out = m
				return true
			}
		}
		return false
	}
}

func publishUntil(t *testing.T, stop <-chan struct{}, publish func()) {
	t.Helper()
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				publish()
			}
		}
	}()
}

func TestSubscribeCoreSubjects(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "subscribe-core", env.natsURL, nil)

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	defer nc.Close()

	t.Run("delivers core messages with their reply subject", func(t *testing.T) {
		events := openLive(t, env, connID, "core.>")
		stop := make(chan struct{})
		defer close(stop)
		publishUntil(t, stop, func() { _ = nc.PublishRequest("core.ask", "my.reply.inbox", []byte(`{"q":1}`)) })

		var msg *natstypes.NatsMessage
		waitLive(t, events, 10*time.Second, batchMessage("core.ask", &msg))
		assert.Equal(t, "my.reply.inbox", msg.GetReply())
		assert.Empty(t, msg.GetStream())
	})

	t.Run("core publish reaches a plain subscriber", func(t *testing.T) {
		sub, err := nc.SubscribeSync("core.out")
		require.NoError(t, err)
		require.NoError(t, nc.Flush())

		resp, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID,
			Subject:      "core.out",
			Data:         `{"resent":true}`,
			Headers:      map[string]string{"X-Trace": "t-1"},
			Core:         true,
		}))
		require.NoError(t, err)
		require.Nil(t, resp.Msg.Error)
		assert.Empty(t, resp.Msg.GetStream(), "a core publish has no stream")

		got, err := sub.NextMsg(5 * time.Second)
		require.NoError(t, err)
		assert.JSONEq(t, `{"resent":true}`, string(got.Data))
		assert.Equal(t, "t-1", got.Header.Get("X-Trace"))
	})

	t.Run("answers a waiting request through its reply subject", func(t *testing.T) {
		events := openLive(t, env, connID, "svc.question")

		replies := make(chan *nats.Msg, 1)
		stop := make(chan struct{})
		defer close(stop)
		go func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				if reply, err := nc.Request("svc.question", []byte("ping"), 500*time.Millisecond); err == nil {
					replies <- reply
					return
				}
			}
		}()

		var msg *natstypes.NatsMessage
		waitLive(t, events, 10*time.Second, batchMessage("svc.question", &msg))
		require.NotEmpty(t, msg.GetReply())

		resp, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: msg.GetReply(), Data: "pong", Core: true,
		}))
		require.NoError(t, err)
		require.Nil(t, resp.Msg.Error)

		select {
		case reply := <-replies:
			assert.Equal(t, "pong", string(reply.Data))
		case <-time.After(5 * time.Second):
			t.Fatal("the requester never got the reply")
		}
	})

	t.Run("core publish may carry no body", func(t *testing.T) {
		resp, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "core.empty", Core: true,
		}))
		require.NoError(t, err)
		assert.Nil(t, resp.Msg.Error)
	})
}

func TestSubscribePermissionDenied(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	srv := startNATSWithOptions(t, func(o *server.Options) {
		o.Users = []*server.User{
			{
				Username: "viewer",
				Password: "pw",
				Permissions: &server.Permissions{
					Publish:   &server.SubjectPermission{Deny: []string{"secret.>"}},
					Subscribe: &server.SubjectPermission{Deny: []string{"secret.>"}},
				},
			},
			{Username: "admin", Password: "pw"},
		}
	})
	connID := createTestConnection(t, env, "subscribe-viewer", srv.ClientURL(), &natstypes.AuthConfig{
		Method:   natstypes.AuthMethod_AUTH_METHOD_USER_PASSWORD,
		Username: new("viewer"),
		Password: new("pw"),
	})

	admin, err := nats.Connect(srv.ClientURL(), nats.UserInfo("admin", "pw"))
	require.NoError(t, err)
	defer admin.Close()

	t.Run("a denied subject is reported and the others keep flowing", func(t *testing.T) {
		events := openLive(t, env, connID, "open.>", "secret.>")

		denied := waitLive(t, events, 10*time.Second, func(ev *livepb.LiveEvent) bool { return ev.GetError() != nil })
		assert.Equal(t, "SUBSCRIBE_PERMISSION_DENIED", denied.GetError().GetCode())
		access := denied.GetError().GetAccess()
		assert.Equal(t, natstypes.AccessStatus_ACCESS_STATUS_DENIED, access.GetStatus())
		assert.Equal(t, "subscribe", access.GetOperation())
		assert.Equal(t, "secret.>", access.GetSubject())

		stop := make(chan struct{})
		defer close(stop)
		publishUntil(t, stop, func() { _ = admin.Publish("open.news", []byte("hello")) })
		var msg *natstypes.NatsMessage
		waitLive(t, events, 10*time.Second, batchMessage("open.news", &msg))
		assert.Equal(t, "open.news", msg.GetSubject())
	})

	t.Run("a denied core publish fails fast with the reason", func(t *testing.T) {
		start := time.Now()
		resp, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "secret.op", Data: "x", Core: true,
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Error)
		assert.Equal(t, `Failed to publish message: no permission to publish to "secret.op"`, resp.Msg.GetError())
		assert.Less(t, time.Since(start), 5*time.Second)
	})
}
