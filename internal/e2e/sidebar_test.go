// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package e2e

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
)

func TestListStreamNames(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "stream-names", env.natsURL, nil)

	for _, name := range []string{"ZETA", "ALPHA", "MIDDLE"} {
		_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: name, Subjects: []string{"names." + name},
		}))
		require.NoError(t, err)
	}

	resp, err := env.streams.ListStreamNames(ctx, connect.NewRequest(&streamspb.ListStreamNamesRequest{ConnectionId: connID}))
	require.NoError(t, err)
	assert.Equal(t, []string{"ALPHA", "MIDDLE", "ZETA"}, resp.Msg.GetNames())
}

func TestSidebarLayout(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "sidebar-layout", env.natsURL, nil)

	get := func(t *testing.T) *connectionspb.SidebarLayout {
		t.Helper()
		resp, err := env.connections.GetSidebarLayout(ctx, connect.NewRequest(&connectionspb.GetSidebarLayoutRequest{ConnectionId: connID}))
		require.NoError(t, err)
		return resp.Msg.GetLayout()
	}

	t.Run("empty before anything is arranged", func(t *testing.T) {
		layout := get(t)
		assert.Empty(t, layout.GetStreams().GetPinned())
		assert.Empty(t, layout.GetStreams().GetOrder())
	})

	t.Run("sections update independently", func(t *testing.T) {
		_, err := env.connections.UpdateSidebarLayout(ctx, connect.NewRequest(&connectionspb.UpdateSidebarLayoutRequest{
			ConnectionId: connID,
			Streams:      &connectionspb.SectionLayout{Pinned: []string{"ORDERS"}, Order: []string{"EVENTS", "ORDERS", "AUDIT"}},
		}))
		require.NoError(t, err)
		_, err = env.connections.UpdateSidebarLayout(ctx, connect.NewRequest(&connectionspb.UpdateSidebarLayoutRequest{
			ConnectionId: connID,
			Kv:           &connectionspb.SectionLayout{Pinned: []string{"config"}},
		}))
		require.NoError(t, err)

		layout := get(t)
		assert.Equal(t, []string{"ORDERS"}, layout.GetStreams().GetPinned())
		assert.Equal(t, []string{"EVENTS", "AUDIT"}, layout.GetStreams().GetOrder(), "a pinned name leaves the manual order")
		assert.Equal(t, []string{"config"}, layout.GetKv().GetPinned())
	})

	t.Run("unknown connection", func(t *testing.T) {
		_, err := env.connections.UpdateSidebarLayout(ctx, connect.NewRequest(&connectionspb.UpdateSidebarLayoutRequest{
			ConnectionId: unknownUUID,
			Streams:      &connectionspb.SectionLayout{Pinned: []string{"ORDERS"}},
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "%v", err)
		assert.Equal(t, "CONNECTION_NOT_FOUND", errorReason(t, err))
	})

	t.Run("deleting the connection drops its layout", func(t *testing.T) {
		_, err := env.connections.DeleteConnection(ctx, connect.NewRequest(&connectionspb.DeleteConnectionRequest{Id: connID}))
		require.NoError(t, err)

		_, err = env.connections.GetSidebarLayout(ctx, connect.NewRequest(&connectionspb.GetSidebarLayoutRequest{ConnectionId: connID}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "%v", err)
	})
}
