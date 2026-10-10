// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package e2e

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	livepb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/live"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// TestWatchKV checks that a bucket watch streams the changes of matching keys.
func TestWatchKV(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-kvwatch")
	_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: connID, Config: &natstypes.KVBucketConfig{Bucket: "watched"},
	}))
	require.NoError(t, err)

	wctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	events := make(chan *livepb.WatchKVEvent, 64)
	go func() {
		defer close(events)
		stream, err := env.live.WatchKV(wctx, connect.NewRequest(&livepb.WatchKVRequest{
			ConnectionId: connID, Bucket: "watched", Filter: "orders.*",
		}))
		if err != nil {
			return
		}
		defer stream.Close()
		for stream.Receive() {
			select {
			case events <- stream.Msg():
			case <-wctx.Done():
				return
			}
		}
	}()

	put := func(key string) {
		_, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
			ConnectionId: connID, Bucket: "watched", Key: key,
			Payload: &managementpb.PutKVKeyRequest_Value{Value: base64.StdEncoding.EncodeToString([]byte("v"))},
		}))
		require.NoError(t, err)
	}

	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case ev, ok := <-events:
			require.True(t, ok, "the watch ended early")
			for _, ch := range ev.GetChanges() {
				require.Equal(t, "orders.1", ch.GetKey(), "keys outside the filter are not watched")
				assert.Equal(t, "put", ch.GetOperation())
				assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("v")), ch.GetValue())
				assert.Equal(t, int32(1), ch.GetSize())
				assert.NotZero(t, ch.GetRevision())
				assert.WithinDuration(t, time.Now(), ch.GetCreated().AsTime(), time.Minute)
			}
			if len(ev.GetChanges()) > 0 {
				return
			}
		case <-tick.C:
			put("users.1")
			put("orders.1")
		case <-deadline:
			t.Fatal("no change arrived")
		}
	}
}
