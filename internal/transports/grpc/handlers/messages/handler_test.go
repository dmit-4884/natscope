// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
)

// --- Mocks ---

// mockMsgService implements the (small) messages Service interface.
type mockMsgService struct {
	listResult *entities.MessagesResponse
	listErr    error
	getResult  *entities.Message
	getErr     error
	nextIn     *entities.MessageNextRequest
	nextResult *entities.Message
}

func (m *mockMsgService) List(_ context.Context, _ *entities.MessageListRequest) (*entities.MessagesResponse, error) {
	return m.listResult, m.listErr
}

func (m *mockMsgService) Get(_ context.Context, _ *entities.MessageGetRequest) (*entities.Message, error) {
	return m.getResult, m.getErr
}

func (m *mockMsgService) Next(_ context.Context, in *entities.MessageNextRequest) (*entities.Message, error) {
	m.nextIn = in
	return m.nextResult, nil
}

// --- Tests ---

func TestHandler_ListMessages(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockMsgService{listResult: &entities.MessagesResponse{
			Messages: []*entities.Message{{Sequence: 1, Subject: "orders.new"}},
			HasMore:  true,
			NextSeq:  42,
		}}
		handler := New(svc)

		resp, err := handler.ListMessages(t.Context(), connect.NewRequest(&messagespb.ListMessagesRequest{
			ConnectionId: "conn1",
			StreamName:   "ORDERS",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Msg.Messages, 1)
		assert.True(t, resp.Msg.HasMore)
		assert.Equal(t, uint64(42), resp.Msg.NextSeq)
		assert.Equal(t, "orders.new", resp.Msg.Messages[0].Subject)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockMsgService{listErr: errs.ErrStreamNotFound}
		handler := New(svc)

		_, err := handler.ListMessages(t.Context(), connect.NewRequest(&messagespb.ListMessagesRequest{
			ConnectionId: "conn1",
			StreamName:   "MISSING",
		}))
		assert.ErrorIs(t, err, errs.ErrStreamNotFound)
	})
}

func TestHandler_GetMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockMsgService{getResult: &entities.Message{Sequence: 7, Subject: "orders.paid"}}
		handler := New(svc)

		resp, err := handler.GetMessage(t.Context(), connect.NewRequest(&messagespb.GetMessageRequest{
			ConnectionId: "conn1",
			StreamName:   "ORDERS",
			Sequence:     7,
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Msg.Message)
		assert.Equal(t, uint64(7), resp.Msg.Message.Sequence)
		assert.Equal(t, "orders.paid", resp.Msg.Message.Subject)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockMsgService{getErr: errs.ErrMsgNotFound}
		handler := New(svc)

		_, err := handler.GetMessage(t.Context(), connect.NewRequest(&messagespb.GetMessageRequest{
			ConnectionId: "conn1",
			StreamName:   "ORDERS",
			Sequence:     999,
		}))
		assert.ErrorIs(t, err, errs.ErrMsgNotFound)
	})
}

func TestHandler_GetNextMessage(t *testing.T) {
	t.Parallel()

	t.Run("returns the match for the requested filters", func(t *testing.T) {
		t.Parallel()
		svc := &mockMsgService{nextResult: &entities.Message{Sequence: 7, Subject: "orders.paid"}}

		resp, err := New(svc).GetNextMessage(t.Context(), connect.NewRequest(&messagespb.GetNextMessageRequest{
			ConnectionId: "conn1", StreamName: "ORDERS", StartSeq: 5, Subjects: []string{"orders.paid", "orders.*"},
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Message)
		assert.Equal(t, uint64(7), resp.Msg.GetMessage().GetSequence())
		assert.Equal(t, &entities.MessageNextRequest{
			ConnectionID: "conn1", StreamName: "ORDERS", StartSeq: 5, Subjects: []string{"orders.paid", "orders.*"},
		}, svc.nextIn)
	})

	t.Run("no match leaves the message unset", func(t *testing.T) {
		t.Parallel()
		resp, err := New(&mockMsgService{}).GetNextMessage(t.Context(), connect.NewRequest(&messagespb.GetNextMessageRequest{
			ConnectionId: "conn1", StreamName: "ORDERS", StartSeq: 99,
		}))
		require.NoError(t, err)
		assert.Nil(t, resp.Msg.Message)
	})
}
