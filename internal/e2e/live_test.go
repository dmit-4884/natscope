// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
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
