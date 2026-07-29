// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
)

// --- Mocks ---

// mockNATSSvc embeds nats.StreamReader and overrides only the two methods the
// streams handler calls.
type mockNATSSvc struct {
	natssvc.StreamReader
	listResult []entities.StreamInfo
	listErr    error
	getResult  *entities.StreamInfo
	getErr     error
}

func (m *mockNATSSvc) ListStreams(_ context.Context, _ string) ([]entities.StreamInfo, error) {
	return m.listResult, m.listErr
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
