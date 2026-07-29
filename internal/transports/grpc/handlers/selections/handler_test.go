// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package selections

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	selectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/selections"
)

// --- Mocks ---

// mockProtoService embeds the selections handler's narrow proto roles and
// overrides only the methods it calls.
type mockProtoService struct {
	protosvc.SelectionManager
	protosvc.Registry
	listResult   entities.ProtoSelections
	listErr      error
	selectResult *entities.ProtoSelection
	selectErr    error
	deleteErr    error
	loadResult   *entities.ProtoLoadResult
	loadErr      error
	stats        *entities.ProtoStats
}

func (m *mockProtoService) ListSelections(_ context.Context) (entities.ProtoSelections, error) {
	return m.listResult, m.listErr
}

func (m *mockProtoService) Select(_ context.Context, _ *entities.ProtoSelectionCreate) (*entities.ProtoSelection, error) {
	return m.selectResult, m.selectErr
}

func (m *mockProtoService) DeleteSelection(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockProtoService) LoadAllSelections(_ context.Context) (*entities.ProtoLoadResult, error) {
	return m.loadResult, m.loadErr
}

func (m *mockProtoService) Stats(_ context.Context) *entities.ProtoStats {
	return m.stats
}

// --- Tests ---

func TestHandler_ListSelections(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{listResult: entities.ProtoSelections{
			entities.ProtoSelectionNew(func(s *entities.ProtoSelection) { s.SourceID = "src-1" }),
		}}
		handler := New(svc, svc)

		resp, err := handler.ListSelections(t.Context(), connect.NewRequest(&selectionspb.ListSelectionsRequest{}))
		require.NoError(t, err)
		assert.Len(t, resp.Msg.Selections, 1)
	})

	t.Run("FilterBySource", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{listResult: entities.ProtoSelections{
			entities.ProtoSelectionNew(func(s *entities.ProtoSelection) { s.SourceID = "src-1" }),
			entities.ProtoSelectionNew(func(s *entities.ProtoSelection) { s.SourceID = "src-2" }),
		}}
		handler := New(svc, svc)

		resp, err := handler.ListSelections(t.Context(), connect.NewRequest(&selectionspb.ListSelectionsRequest{
			SourceId: ptrString("src-1"),
		}))
		require.NoError(t, err)
		assert.Len(t, resp.Msg.Selections, 1)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{listErr: errors.New("db error")}
		handler := New(svc, svc)

		_, err := handler.ListSelections(t.Context(), connect.NewRequest(&selectionspb.ListSelectionsRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_SelectVersion(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{selectResult: entities.ProtoSelectionNew(func(s *entities.ProtoSelection) {
			s.SourceID = "src-1"
			s.Tag = "v1.0.0"
		})}
		handler := New(svc, svc)

		resp, err := handler.SelectVersion(t.Context(), connect.NewRequest(&selectionspb.SelectVersionRequest{
			SourceId: "src-1",
			Tag:      "v1.0.0",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Selection)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{selectErr: errs.ErrProtoVersionNotFound}
		handler := New(svc, svc)

		_, err := handler.SelectVersion(t.Context(), connect.NewRequest(&selectionspb.SelectVersionRequest{
			SourceId: "src-1",
			Tag:      "missing",
		}))
		assert.ErrorIs(t, err, errs.ErrProtoVersionNotFound)
	})
}

func TestHandler_DeleteSelection(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{}
		handler := New(svc, svc)

		resp, err := handler.DeleteSelection(t.Context(), connect.NewRequest(&selectionspb.DeleteSelectionRequest{Id: "sel-1"}))
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{deleteErr: errs.ErrProtoSelectionNotFound}
		handler := New(svc, svc)

		_, err := handler.DeleteSelection(t.Context(), connect.NewRequest(&selectionspb.DeleteSelectionRequest{Id: "x"}))
		assert.ErrorIs(t, err, errs.ErrProtoSelectionNotFound)
	})
}

func TestHandler_LoadSelections(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{loadResult: &entities.ProtoLoadResult{MessageCount: 42}}
		handler := New(svc, svc)

		resp, err := handler.LoadSelections(t.Context(), connect.NewRequest(&selectionspb.LoadSelectionsRequest{}))
		require.NoError(t, err)
		assert.Equal(t, int32(42), resp.Msg.MessageCount)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{loadErr: errors.New("compile failed")}
		handler := New(svc, svc)

		_, err := handler.LoadSelections(t.Context(), connect.NewRequest(&selectionspb.LoadSelectionsRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_GetSelectionStatus(t *testing.T) {
	t.Parallel()

	t.Run("Loaded", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{stats: &entities.ProtoStats{MessagesCount: 7}}
		handler := New(svc, svc)

		resp, err := handler.GetSelectionStatus(t.Context(), connect.NewRequest(&selectionspb.GetSelectionStatusRequest{}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.Loaded)
		assert.Equal(t, int32(7), resp.Msg.MessageCount)
	})

	t.Run("NotLoaded", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoService{stats: &entities.ProtoStats{}}
		handler := New(svc, svc)

		resp, err := handler.GetSelectionStatus(t.Context(), connect.NewRequest(&selectionspb.GetSelectionStatusRequest{}))
		require.NoError(t, err)
		assert.False(t, resp.Msg.Loaded)
	})
}

// ptrString returns a pointer to s (for oneof request fields).
func ptrString(s string) *string { return &s }
