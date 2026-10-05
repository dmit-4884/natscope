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
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
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
	testResult   *entities.TestConnectionResult
	layout       *entities.SidebarLayout
	layoutErr    error
	layoutUpdate *entities.SidebarLayoutUpdate
	cliContexts  *entities.CliContexts
}

func (m *mockConnService) ListCliContexts(context.Context, []entities.CliContextFile) (*entities.CliContexts, error) {
	return m.cliContexts, nil
}

func (m *mockConnService) ImportCliContexts(context.Context, []string, []entities.CliContextFile) (*entities.CliContextImport, error) {
	return &entities.CliContextImport{}, nil
}

func (m *mockConnService) GetSidebarLayout(_ context.Context, _ string) (*entities.SidebarLayout, error) {
	return m.layout, m.layoutErr
}

func (m *mockConnService) UpdateSidebarLayout(_ context.Context, in *entities.SidebarLayoutUpdate) (*entities.SidebarLayout, error) {
	m.layoutUpdate = in
	return m.layout, m.layoutErr
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
	if m.testResult != nil {
		return m.testResult, nil
	}
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

func TestHandler_TestConnection_OptionalFields(t *testing.T) {
	t.Parallel()

	t.Run("failure reports only the error", func(t *testing.T) {
		t.Parallel()
		h := New(&mockConnService{testResult: &entities.TestConnectionResult{Error: "no servers available"}})
		resp, err := h.TestConnection(t.Context(), connect.NewRequest(&connectionspb.TestConnectionRequest{}))
		require.NoError(t, err)
		assert.False(t, resp.Msg.GetSuccess())
		assert.Equal(t, "no servers available", resp.Msg.GetError())
		assert.Nil(t, resp.Msg.RttMs)
		assert.Nil(t, resp.Msg.ServerVersion)
	})

	t.Run("success omits the error", func(t *testing.T) {
		t.Parallel()
		h := New(&mockConnService{testResult: &entities.TestConnectionResult{Success: true, RTTMs: 0, ServerVersion: "2.14.2"}})
		resp, err := h.TestConnection(t.Context(), connect.NewRequest(&connectionspb.TestConnectionRequest{}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.GetSuccess())
		assert.Nil(t, resp.Msg.Error)
		require.NotNil(t, resp.Msg.RttMs)
		assert.Equal(t, "2.14.2", resp.Msg.GetServerVersion())
	})
}

func TestHandler_SidebarLayout(t *testing.T) {
	t.Parallel()

	layout := &entities.SidebarLayout{
		ConnectionID: "conn-1",
		Streams:      entities.SectionLayout{Pinned: []string{"ORDERS"}, Order: []string{"EVENTS"}},
		KV:           entities.SectionLayout{Pinned: []string{"config"}},
	}

	t.Run("get", func(t *testing.T) {
		t.Parallel()
		h := New(&mockConnService{layout: layout})

		resp, err := h.GetSidebarLayout(t.Context(), connect.NewRequest(&connectionspb.GetSidebarLayoutRequest{ConnectionId: "conn-1"}))
		require.NoError(t, err)
		assert.Equal(t, []string{"ORDERS"}, resp.Msg.GetLayout().GetStreams().GetPinned())
		assert.Equal(t, []string{"EVENTS"}, resp.Msg.GetLayout().GetStreams().GetOrder())
		assert.Equal(t, []string{"config"}, resp.Msg.GetLayout().GetKv().GetPinned())
		assert.NotNil(t, resp.Msg.GetLayout().GetObjects(), "an empty section still reaches the wire")
	})

	t.Run("update converts only the sections that are set", func(t *testing.T) {
		t.Parallel()
		svc := &mockConnService{layout: layout}
		h := New(svc)

		_, err := h.UpdateSidebarLayout(t.Context(), connect.NewRequest(&connectionspb.UpdateSidebarLayoutRequest{
			ConnectionId: "conn-1",
			Kv:           &connectionspb.SectionLayout{Pinned: []string{"config"}, Order: []string{"flags"}},
		}))
		require.NoError(t, err)

		got := svc.layoutUpdate
		require.NotNil(t, got)
		assert.Equal(t, "conn-1", got.ConnectionID)
		assert.Nil(t, got.Streams)
		assert.Nil(t, got.Objects)
		require.NotNil(t, got.KV)
		assert.Equal(t, []string{"config"}, got.KV.Pinned)
		assert.Equal(t, []string{"flags"}, got.KV.Order)
	})

	t.Run("unknown connection", func(t *testing.T) {
		t.Parallel()
		h := New(&mockConnService{layoutErr: errs.ErrSavedConnectionNotFound})

		_, err := h.GetSidebarLayout(t.Context(), connect.NewRequest(&connectionspb.GetSidebarLayoutRequest{ConnectionId: "conn-1"}))
		assert.ErrorIs(t, err, errs.ErrSavedConnectionNotFound)
	})
}

func TestHandler_ListCliContexts_DescribesWithoutSecrets(t *testing.T) {
	t.Parallel()
	token := "s3cret"
	svc := &mockConnService{cliContexts: &entities.CliContexts{Dir: "/cfg/nats/context", Contexts: []entities.CliContext{
		{Name: "prod", Selected: true, Warnings: []string{"SOCKS proxies are not supported"}, Connection: &entities.SavedConnectionCreate{
			URLs: []string{"tls://p:4222"},
			Auth: &entities.AuthConfig{Method: entities.AuthMethodToken, Token: &token},
			TLS:  &entities.TlsConfig{TlsFirst: true},
		}},
		{Name: "garbage", Warnings: []string{"this is not a nats CLI context"}},
	}}}

	resp, err := New(svc).ListCliContexts(t.Context(), connect.NewRequest(&connectionspb.ListCliContextsRequest{}))
	require.NoError(t, err)

	assert.Equal(t, "/cfg/nats/context", resp.Msg.GetDirectory())
	prod := resp.Msg.GetContexts()[0]
	assert.True(t, prod.GetImportable())
	assert.True(t, prod.GetSelected())
	assert.True(t, prod.GetTls())
	assert.Equal(t, natspb.AuthMethod_AUTH_METHOD_TOKEN, prod.GetAuthMethod())
	assert.NotContains(t, prod.String(), token)
	assert.False(t, resp.Msg.GetContexts()[1].GetImportable())
}
