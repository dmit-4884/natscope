// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	registrypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/registry"
)

// --- Mocks ---

// mockProtoSvc embeds proto.Registry; only registry methods are overridden.
type mockProtoSvc struct {
	protosvc.Registry
	listResult    []entities.ProtoMessageInfo
	getResult     *entities.ProtoMessageInfo
	getErr        error
	exampleResult map[string]interface{}
	exampleErr    error
	statsResult   *entities.ProtoStats
}

func (m *mockProtoSvc) ListMessages(_ context.Context) []entities.ProtoMessageInfo {
	return m.listResult
}

func (m *mockProtoSvc) GetMessage(_ context.Context, _, _ string) (*entities.ProtoMessageInfo, error) {
	return m.getResult, m.getErr
}

func (m *mockProtoSvc) GenerateExample(_ context.Context, _, _ string) (map[string]interface{}, error) {
	return m.exampleResult, m.exampleErr
}

func (m *mockProtoSvc) Stats(_ context.Context) *entities.ProtoStats {
	return m.statsResult
}

// --- Tests ---

func TestHandler_ListProtoMessages(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{listResult: []entities.ProtoMessageInfo{
			{FullName: "test.Msg", SourceID: "src-1"},
		}}
		handler := New(svc)

		resp, err := handler.ListProtoMessages(t.Context(), connect.NewRequest(&registrypb.ListProtoMessagesRequest{}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.Messages, 1)
		assert.Equal(t, "test.Msg", resp.Msg.Messages[0].GetFullName())
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		handler := New(&mockProtoSvc{})

		resp, err := handler.ListProtoMessages(t.Context(), connect.NewRequest(&registrypb.ListProtoMessagesRequest{}))
		require.NoError(t, err)
		assert.Empty(t, resp.Msg.Messages)
	})
}

func TestHandler_GetProtoMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{getResult: &entities.ProtoMessageInfo{FullName: "test.Msg", SourceID: "src-1"}}
		handler := New(svc)

		resp, err := handler.GetProtoMessage(t.Context(), connect.NewRequest(&registrypb.GetProtoMessageRequest{
			SourceId: "src-1",
			FullName: "test.Msg",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Message)
		assert.Equal(t, "test.Msg", resp.Msg.Message.GetFullName())
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{getErr: errs.ErrProtoMessageNotFound}
		handler := New(svc)

		_, err := handler.GetProtoMessage(t.Context(), connect.NewRequest(&registrypb.GetProtoMessageRequest{
			SourceId: "src-1",
			FullName: "missing.Msg",
		}))
		assert.ErrorIs(t, err, errs.ErrProtoMessageNotFound)
	})
}

func TestHandler_GenerateExample(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{exampleResult: map[string]interface{}{"field": "value"}}
		handler := New(svc)

		resp, err := handler.GenerateExample(t.Context(), connect.NewRequest(&registrypb.GenerateExampleRequest{
			SourceId: "src-1",
			FullName: "test.Msg",
		}))
		require.NoError(t, err)
		assert.Contains(t, resp.Msg.Json, "field")
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{exampleErr: errs.ErrProtoMessageNotFound}
		handler := New(svc)

		_, err := handler.GenerateExample(t.Context(), connect.NewRequest(&registrypb.GenerateExampleRequest{
			SourceId: "src-1",
			FullName: "missing.Msg",
		}))
		assert.ErrorIs(t, err, errs.ErrProtoMessageNotFound)
	})
}

func TestHandler_GetProtoStatus(t *testing.T) {
	t.Parallel()

	t.Run("Loaded", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{statsResult: &entities.ProtoStats{MessagesCount: 3}}
		handler := New(svc)

		resp, err := handler.GetProtoStatus(t.Context(), connect.NewRequest(&registrypb.GetProtoStatusRequest{}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.Loaded)
		assert.Equal(t, int32(3), resp.Msg.MessageCount)
	})

	t.Run("NilStats", func(t *testing.T) {
		t.Parallel()
		handler := New(&mockProtoSvc{})

		resp, err := handler.GetProtoStatus(t.Context(), connect.NewRequest(&registrypb.GetProtoStatusRequest{}))
		require.NoError(t, err)
		assert.False(t, resp.Msg.Loaded)
	})
}
