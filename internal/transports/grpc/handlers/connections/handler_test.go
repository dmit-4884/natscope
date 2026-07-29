// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	ptr "github.com/altessa-s/go-atlas/core/types/ptr"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
)

// --- Mocks ---

type mockConnService struct {
	createResult *entities.SavedConnection
	createErr    error
	getResult    *entities.SavedConnection
	getErr       error
	listResult   *entities.List[entities.SavedConnections]
	listErr      error
	updateResult *entities.SavedConnection
	updateErr    error
	deleteErr    error
	dupResult    *entities.SavedConnection
	dupErr       error
}

func (m *mockConnService) Create(_ context.Context, in *entities.SavedConnectionCreate) (*entities.SavedConnection, error) {
	if m.createResult != nil {
		return m.createResult, m.createErr
	}
	conn := entities.SavedConnectionNew(func(c *entities.SavedConnection) {
		c.Name = in.Name
		c.URLs = in.URLs
	})
	return conn, m.createErr
}

func (m *mockConnService) Get(_ context.Context, _ string) (*entities.SavedConnection, error) {
	return m.getResult, m.getErr
}

func (m *mockConnService) List(_ context.Context, _ *entities.SavedConnectionsList) (*entities.List[entities.SavedConnections], error) {
	return m.listResult, m.listErr
}

func (m *mockConnService) Update(_ context.Context, _ *entities.SavedConnectionUpdate) (*entities.SavedConnection, error) {
	if m.updateResult != nil {
		return m.updateResult, m.updateErr
	}
	return entities.SavedConnectionNew(), m.updateErr
}

func (m *mockConnService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockConnService) Duplicate(_ context.Context, _ string, _ string) (*entities.SavedConnection, error) {
	if m.dupResult != nil {
		return m.dupResult, m.dupErr
	}
	return entities.SavedConnectionNew(), m.dupErr
}

func (m *mockConnService) TestConnection(_ context.Context, _ *entities.TestConnectionRequest) (*entities.TestConnectionResult, error) {
	return &entities.TestConnectionResult{}, nil
}

// --- Tests ---

func TestHandler_Create(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{}
		handler := New(svc)

		resp, err := handler.CreateConnection(t.Context(), connect.NewRequest(&connectionspb.CreateConnectionRequest{
			Name: "test-conn",
			Urls: []string{"nats://localhost:4222"},
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Msg.Connection)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{createErr: errs.ErrConnectionNameAlreadyInUse}
		handler := New(svc)

		_, err := handler.CreateConnection(t.Context(), connect.NewRequest(&connectionspb.CreateConnectionRequest{Name: "dup"}))
		assert.ErrorIs(t, err, errs.ErrConnectionNameAlreadyInUse)
	})
}

func TestHandler_List(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{
			listResult: &entities.List[entities.SavedConnections]{
				Items:      entities.SavedConnections{entities.SavedConnectionNew()},
				NextCursor: ptr.Wrap("next"),
			},
		}
		handler := New(svc)

		resp, err := handler.ListConnections(t.Context(), connect.NewRequest(&connectionspb.ListConnectionsRequest{}))
		require.NoError(t, err)
		assert.Len(t, resp.Msg.Connections, 1)
		require.NotNil(t, resp.Msg.NextPageToken)
		assert.Equal(t, "next", *resp.Msg.NextPageToken)
	})

	t.Run("WithPagination", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{
			listResult: &entities.List[entities.SavedConnections]{},
		}
		handler := New(svc)

		_, err := handler.ListConnections(t.Context(), connect.NewRequest(&connectionspb.ListConnectionsRequest{
			PageSize:  10,
			PageToken: "c1",
		}))
		require.NoError(t, err)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{listErr: errors.New("db error")}
		handler := New(svc)

		_, err := handler.ListConnections(t.Context(), connect.NewRequest(&connectionspb.ListConnectionsRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_Update(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		existing := entities.SavedConnectionNew()
		svc := &mockConnService{getResult: existing}
		handler := New(svc)

		resp, err := handler.UpdateConnection(t.Context(), connect.NewRequest(&connectionspb.UpdateConnectionRequest{
			Id:   existing.Id,
			Name: ptr.Wrap("updated"),
			Urls: []string{"nats://new:4222"},
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Msg.Connection)
	})

	t.Run("OwnershipViolation", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{updateErr: errs.ErrSavedConnectionNotFound}
		handler := New(svc)

		_, err := handler.UpdateConnection(t.Context(), connect.NewRequest(&connectionspb.UpdateConnectionRequest{Id: "some-id"}))
		assert.ErrorIs(t, err, errs.ErrSavedConnectionNotFound)
	})

	t.Run("UpdateFails_NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{updateErr: errs.ErrSavedConnectionNotFound}
		handler := New(svc)

		_, err := handler.UpdateConnection(t.Context(), connect.NewRequest(&connectionspb.UpdateConnectionRequest{Id: "x"}))
		assert.ErrorIs(t, err, errs.ErrSavedConnectionNotFound)
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{}
		handler := New(svc)

		resp, err := handler.DeleteConnection(t.Context(), connect.NewRequest(&connectionspb.DeleteConnectionRequest{Id: "conn-1"}))
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{deleteErr: errs.ErrSavedConnectionNotFound}
		handler := New(svc)

		_, err := handler.DeleteConnection(t.Context(), connect.NewRequest(&connectionspb.DeleteConnectionRequest{Id: "x"}))
		assert.ErrorIs(t, err, errs.ErrSavedConnectionNotFound)
	})

	t.Run("DeleteFails", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{deleteErr: errors.New("db error")}
		handler := New(svc)

		_, err := handler.DeleteConnection(t.Context(), connect.NewRequest(&connectionspb.DeleteConnectionRequest{Id: "conn-1"}))
		assert.Error(t, err)
	})
}
