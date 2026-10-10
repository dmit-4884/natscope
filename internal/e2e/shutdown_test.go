// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

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
)

// TestGracefulShutdown_ClosesOpenLiveStream checks that Stop ends an open Live stream promptly.
func TestGracefulShutdown_ClosesOpenLiveStream(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	createResp, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: "shutdown-conn", Urls: []string{env.natsURL},
	}))
	require.NoError(t, err)
	connID := createResp.Msg.GetConnection().GetId()

	subCtx, subCancel := context.WithTimeout(ctx, 20*time.Second)
	defer subCancel()

	stream, err := env.live.Subscribe(subCtx, connect.NewRequest(&livepb.SubscribeRequest{
		ConnectionId:  connID,
		Subscriptions: []*livepb.LiveSubscription{{Subject: "shutdown.>"}},
	}))
	require.NoError(t, err)

	streamEnded := make(chan struct{})
	go func() {
		defer close(streamEnded)
		for stream.Receive() { //nolint:revive // drain only
		}
	}()

	// Wait for the first server flush so Stop hits an open stream.
	require.Eventually(t, func() bool {
		select {
		case <-streamEnded:
			return false
		default:
			return true
		}
	}, 2*time.Second, 50*time.Millisecond)

	stopCtx, stopCancel := context.WithTimeout(ctx, 5*time.Second)
	defer stopCancel()

	start := time.Now()
	err = env.app.Stop(stopCtx)
	elapsed := time.Since(start)

	require.NoError(t, err, "Stop must succeed with an open Live stream")
	assert.Less(t, elapsed, 3*time.Second, "Stop must cancel the stream's context instead of waiting out the shutdown timeout")

	select {
	case <-streamEnded:
	case <-time.After(3 * time.Second):
		t.Fatal("Live stream never observed shutdown")
	}
}
