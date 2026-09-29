// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// --- Mocks ---

// mockNATSSvc embeds nats.StreamReader and overrides only the methods the
// streams handler calls.
type mockNATSSvc struct {
	natssvc.StreamReader
	listResult  []entities.StreamInfo
	listErr     error
	namesResult []string
	getResult   *entities.StreamInfo
	getErr      error
	relations   *entities.StreamRelations
}

func (m *mockNATSSvc) ListStreams(_ context.Context, _ string) ([]entities.StreamInfo, error) {
	return m.listResult, m.listErr
}

func (m *mockNATSSvc) ListStreamNames(_ context.Context, _ string) ([]string, error) {
	return m.namesResult, m.listErr
}

func (m *mockNATSSvc) GetStreamRelations(_ context.Context, _ string) (*entities.StreamRelations, error) {
	return m.relations, m.listErr
}

func (m *mockNATSSvc) GetStreamInfo(_ context.Context, _ string, _ string) (*entities.StreamInfo, error) {
	return m.getResult, m.getErr
}

// --- Tests ---

func TestHandler_ListStreams(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		h := New(&mockNATSSvc{listResult: []entities.StreamInfo{
			{Config: entities.StreamConfig{Name: "ORDERS"}},
			{Config: entities.StreamConfig{Name: "EVENTS"}},
		}})

		resp, err := h.ListStreams(t.Context(), connect.NewRequest(&streamspb.ListStreamsRequest{
			ConnectionId: "conn1",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Msg.Streams, 2)
		assert.Equal(t, "ORDERS", resp.Msg.Streams[0].Config.Name)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		h := New(&mockNATSSvc{listErr: errs.ErrNATSConnectionFailed})

		_, err := h.ListStreams(t.Context(), connect.NewRequest(&streamspb.ListStreamsRequest{
			ConnectionId: "conn1",
		}))
		assert.ErrorIs(t, err, errs.ErrNATSConnectionFailed)
	})
}

func TestHandler_ListStreamNames(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		h := New(&mockNATSSvc{namesResult: []string{"EVENTS", "ORDERS"}})

		resp, err := h.ListStreamNames(t.Context(), connect.NewRequest(&streamspb.ListStreamNamesRequest{
			ConnectionId: "conn1",
		}))
		require.NoError(t, err)
		assert.Equal(t, []string{"EVENTS", "ORDERS"}, resp.Msg.GetNames())
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		h := New(&mockNATSSvc{listErr: errs.ErrNATSConnectionFailed})

		_, err := h.ListStreamNames(t.Context(), connect.NewRequest(&streamspb.ListStreamNamesRequest{
			ConnectionId: "conn1",
		}))
		assert.ErrorIs(t, err, errs.ErrNATSConnectionFailed)
	})
}

func TestHandler_GetStream(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		h := New(&mockNATSSvc{getResult: &entities.StreamInfo{
			Config: entities.StreamConfig{Name: "ORDERS"},
		}})

		resp, err := h.GetStream(t.Context(), connect.NewRequest(&streamspb.GetStreamRequest{
			ConnectionId: "conn1",
			StreamName:   "ORDERS",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Msg.Stream)
		assert.Equal(t, "ORDERS", resp.Msg.Stream.Config.Name)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		h := New(&mockNATSSvc{getErr: errs.ErrStreamNotFound})

		_, err := h.GetStream(t.Context(), connect.NewRequest(&streamspb.GetStreamRequest{
			ConnectionId: "conn1",
			StreamName:   "MISSING",
		}))
		assert.ErrorIs(t, err, errs.ErrStreamNotFound)
	})
}

func TestHandler_GetStreamRelations(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		h := New(&mockNATSSvc{relations: &entities.StreamRelations{
			Nodes: []entities.StreamRelationNode{
				{ID: "AGG", Name: "AGG", Kind: entities.StreamNodeStream, Info: &entities.StreamInfo{
					Config: entities.StreamConfig{Name: "AGG", Replicas: 3},
					State:  &entities.StreamState{Msgs: 42},
				}},
				{ID: "external $JS.hub.API ORDERS", Name: "ORDERS", Kind: entities.StreamNodeExternal,
					External: &entities.ExternalStreamRef{ApiPrefix: "$JS.hub.API"}},
			},
			Edges: []entities.StreamRelationEdge{{
				Kind: entities.StreamRelationSource,
				From: "external $JS.hub.API ORDERS",
				To:   "AGG",
				Source: &entities.StreamSourceRef{
					Name:              "ORDERS",
					OptStartTime:      &start,
					SubjectTransforms: []entities.SubjectTransformConfig{{Source: "orders.>", Destination: "hub.orders.>"}},
				},
				State: &entities.StreamSourceInfo{Name: "ORDERS", Lag: 7, Active: -time.Nanosecond, Error: "stream not found"},
			}, {
				Kind:      entities.StreamRelationRepublish,
				From:      "AGG",
				To:        "subject audit.>",
				Republish: &entities.StreamRePublish{Src: ">", Dest: "audit.>", HeadersOnly: true},
			}},
		}})

		resp, err := h.GetStreamRelations(t.Context(), connect.NewRequest(&streamspb.GetStreamRelationsRequest{ConnectionId: "conn1"}))
		require.NoError(t, err)

		nodes, edges := resp.Msg.Nodes, resp.Msg.Edges
		require.Len(t, nodes, 2)
		assert.Equal(t, natspb.StreamNodeKind_STREAM_NODE_KIND_STREAM, nodes[0].Kind)
		assert.Equal(t, int32(3), nodes[0].Info.Config.Replicas)
		assert.Equal(t, uint64(42), nodes[0].Info.State.Msgs)
		assert.Equal(t, natspb.StreamNodeKind_STREAM_NODE_KIND_EXTERNAL, nodes[1].Kind)
		assert.Equal(t, "$JS.hub.API", nodes[1].External.ApiPrefix)

		require.Len(t, edges, 2)
		assert.Equal(t, natspb.StreamRelationKind_STREAM_RELATION_KIND_SOURCE, edges[0].Kind)
		assert.Equal(t, "AGG", edges[0].To)
		assert.Equal(t, start, edges[0].Source.OptStartTime.AsTime())
		assert.Equal(t, "hub.orders.>", edges[0].Source.SubjectTransforms[0].Destination)
		assert.Equal(t, uint64(7), edges[0].State.Lag)
		assert.Equal(t, -time.Nanosecond, edges[0].State.Active.AsDuration())
		assert.Equal(t, "stream not found", edges[0].State.Error)
		assert.Equal(t, natspb.StreamRelationKind_STREAM_RELATION_KIND_REPUBLISH, edges[1].Kind)
		assert.True(t, edges[1].Republish.HeadersOnly)
		assert.Nil(t, edges[1].Source)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		h := New(&mockNATSSvc{listErr: errs.ErrNATSConnectionFailed})

		_, err := h.GetStreamRelations(t.Context(), connect.NewRequest(&streamspb.GetStreamRelationsRequest{ConnectionId: "conn1"}))
		assert.ErrorIs(t, err, errs.ErrNATSConnectionFailed)
	})
}
