// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	livepb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/live"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
)

// TestLiveSubscribe opens a server-streaming live subscription, publishes on
// the subject, and asserts the message arrives as a batch event.
func TestLiveSubscribe(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	createResp, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: "live-conn", Urls: []string{env.natsURL},
	}))
	require.NoError(t, err)
	connID := createResp.Msg.GetConnection().GetId()

	const stream = "LIVE_FLOW"
	const subject = "live.msg"
	_, err = env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Subjects: []string{subject},
	}))
	require.NoError(t, err)

	subCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Subscribe() doesn't return until the server flushes its first event
	// (the live loop's first tick is a deliberate no-op), so publishing must run concurrently, not after, or it deadlocks.
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
				_, _ = env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
					ConnectionId: connID, Subject: subject, Data: `{"live":true}`,
				}))
			}
		}
	}()

	sub, err := env.live.Subscribe(subCtx, connect.NewRequest(&livepb.SubscribeRequest{
		ConnectionId:  connID,
		Subscriptions: []*livepb.LiveSubscription{{Subject: subject}},
	}))
	require.NoError(t, err)
	defer sub.Close()

	found := false
	for !found && sub.Receive() {
		evt := sub.Msg()
		if batch := evt.GetBatch(); batch != nil {
			for _, m := range batch.GetMessages() {
				if m.GetSubject() == subject {
					found = true
					break
				}
			}
		}
	}
	if !found {
		require.NoError(t, sub.Err())
	}
	assert.True(t, found, "the published message must arrive on the live subscription")
}

// TestLiveEndsWhenConnectionReplaced checks that a session ends with a retryable error when its connection drops.
func TestLiveEndsWhenConnectionReplaced(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "live-replaced", env.natsURL, nil)

	const subject = "live.replaced"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: "LIVE_REPLACED", Subjects: []string{subject},
	}))
	require.NoError(t, err)

	subCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
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
				_, _ = env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
					ConnectionId: connID, Subject: subject, Data: `{"live":true}`,
				}))
			}
		}
	}()

	sub, err := env.live.Subscribe(subCtx, connect.NewRequest(&livepb.SubscribeRequest{
		ConnectionId:  connID,
		Subscriptions: []*livepb.LiveSubscription{{Subject: subject}},
	}))
	require.NoError(t, err)
	defer sub.Close()

	for sub.Receive() {
		if sub.Msg().GetBatch() != nil {
			break
		}
	}
	require.NoError(t, sub.Err())

	_, err = env.connections.UpdateConnection(ctx, connect.NewRequest(&connectionspb.UpdateConnectionRequest{
		Id: connID, Description: new("replaced"),
	}))
	require.NoError(t, err)

	for sub.Receive() {
	}
	require.Error(t, sub.Err())
	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(sub.Err()), "%v", sub.Err())
	assert.Equal(t, "LIVE_CONNECTION_LOST", errorReason(t, sub.Err()))
}

func TestLiveSubscribeStreamWithoutOwnSubjects(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "live-derived", env.natsURL, nil)

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	for _, cfg := range []jetstream.StreamConfig{
		{Name: "ORIGIN", Subjects: []string{"origin.>"}},
		{Name: "ORIGIN_MIRROR", Mirror: &jetstream.StreamSource{Name: "ORIGIN"}},
		{Name: "ORIGIN_AGG", Sources: []*jetstream.StreamSource{{Name: "ORIGIN"}}},
	} {
		_, err := js.CreateStream(ctx, cfg)
		require.NoError(t, err, cfg.Name)
	}

	for _, stream := range []string{"ORIGIN_MIRROR", "ORIGIN_AGG"} {
		t.Run(stream, func(t *testing.T) {
			subCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
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
						_, _ = js.Publish(subCtx, "origin.event", []byte(`{"derived":true}`))
					}
				}
			}()

			sub, err := env.live.Subscribe(subCtx, connect.NewRequest(&livepb.SubscribeRequest{
				ConnectionId:  connID,
				Subscriptions: []*livepb.LiveSubscription{{Subject: ">", StreamName: new(stream)}},
			}))
			require.NoError(t, err)
			defer sub.Close()

			found := false
			for !found && sub.Receive() {
				for _, m := range sub.Msg().GetBatch().GetMessages() {
					if m.GetSubject() == "origin.event" {
						found = true
						break
					}
				}
			}
			require.NoError(t, sub.Err())
			assert.True(t, found, "messages stored in %s must reach its live subscription", stream)
		})
	}
}
