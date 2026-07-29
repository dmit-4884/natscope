// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	ptr "github.com/altessa-s/go-atlas/core/types/ptr"
	historypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/history/v1/history"
)

// --- Mocks ---

type mockHistoryService struct {
	listResult *entities.List[entities.PublishHistories]
	listErr    error
	listInput  *entities.PublishHistoryList
}

func (m *mockHistoryService) Record(_ context.Context, _ *entities.PublishHistoryCreate) (*entities.PublishHistory, error) {
	return nil, nil
}

func (m *mockHistoryService) List(_ context.Context, in *entities.PublishHistoryList) (*entities.List[entities.PublishHistories], error) {
	m.listInput = in
	return m.listResult, m.listErr
}

// --- Tests ---

func TestHandler_ListPublishHistory(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockHistoryService{
			listResult: &entities.List[entities.PublishHistories]{
				Items: entities.PublishHistories{
					entities.PublishHistoryNew(func(h *entities.PublishHistory) {
						h.Subject = "orders.created"
						h.Success = true
					}),
				},
				NextCursor: ptr.Wrap("cursor-2"),
			},
		}
		handler := New(svc)

		resp, err := handler.ListPublishHistory(t.Context(), connect.NewRequest(&historypb.ListPublishHistoryRequest{}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Msg.Entries, 1)
		require.NotNil(t, resp.Msg.NextPageToken)
		assert.Equal(t, "cursor-2", *resp.Msg.NextPageToken)
	})

	t.Run("WithPagination", func(t *testing.T) {
		t.Parallel()
		svc := &mockHistoryService{
			listResult: &entities.List[entities.PublishHistories]{},
		}
		handler := New(svc)

		resp, err := handler.ListPublishHistory(t.Context(), connect.NewRequest(&historypb.ListPublishHistoryRequest{
			PageToken: "prev-cursor",
			PageSize:  25,
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "prev-cursor", svc.listInput.Cursor)
		require.NotNil(t, svc.listInput.Limit)
		assert.Equal(t, int64(25), *svc.listInput.Limit)
	})

	t.Run("WithFilters", func(t *testing.T) {
		t.Parallel()
		svc := &mockHistoryService{
			listResult: &entities.List[entities.PublishHistories]{},
		}
		handler := New(svc)

		connURL := "nats://localhost:4222"
		stream := "ORDERS"
		resp, err := handler.ListPublishHistory(t.Context(), connect.NewRequest(&historypb.ListPublishHistoryRequest{
			ConnectionUrl: &connURL,
			Stream:        &stream,
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, svc.listInput.ConnectionURL)
		assert.Equal(t, "nats://localhost:4222", *svc.listInput.ConnectionURL)
		require.NotNil(t, svc.listInput.Stream)
		assert.Equal(t, "ORDERS", *svc.listInput.Stream)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockHistoryService{listErr: errors.New("db error")}
		handler := New(svc)

		resp, err := handler.ListPublishHistory(t.Context(), connect.NewRequest(&historypb.ListPublishHistoryRequest{}))
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("EmptyResult_NoCursor", func(t *testing.T) {
		t.Parallel()
		svc := &mockHistoryService{
			listResult: &entities.List[entities.PublishHistories]{
				Items:      entities.PublishHistories{},
				NextCursor: nil,
			},
		}
		handler := New(svc)

		resp, err := handler.ListPublishHistory(t.Context(), connect.NewRequest(&historypb.ListPublishHistoryRequest{}))
		require.NoError(t, err)
		assert.Nil(t, resp.Msg.NextPageToken, "should not set cursor when empty")
	})
}
