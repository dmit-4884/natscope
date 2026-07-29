// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package workspace

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	workspacepb "github.com/dmit-4884/natscope/proto/gen/services/grpc/workspace/v1/workspace"
)

// --- Mocks ---

type mockWorkspaceSvc struct {
	listResult     []entities.WorkspaceSectionInfo
	listErr        error
	exportResult   []byte
	exportErr      error
	validateResult []entities.WorkspaceSectionReport
	validateErr    error
	importResult   []entities.WorkspaceSectionResult
	importErr      error
	// gotStrategy records the last strategy passed to Validate/Import.
	gotStrategy entities.WorkspaceStrategy
}

func (m *mockWorkspaceSvc) ListSections(_ context.Context) ([]entities.WorkspaceSectionInfo, error) {
	return m.listResult, m.listErr
}

func (m *mockWorkspaceSvc) Export(_ context.Context, _ []string) ([]byte, error) {
	return m.exportResult, m.exportErr
}

func (m *mockWorkspaceSvc) Validate(
	_ context.Context, _ []byte, _ []string, strategy entities.WorkspaceStrategy,
) ([]entities.WorkspaceSectionReport, error) {
	m.gotStrategy = strategy
	return m.validateResult, m.validateErr
}

func (m *mockWorkspaceSvc) Import(
	_ context.Context, _ []byte, _ []string, strategy entities.WorkspaceStrategy,
) ([]entities.WorkspaceSectionResult, error) {
	m.gotStrategy = strategy
	return m.importResult, m.importErr
}

// --- Tests ---

func TestHandler_ListSections(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockWorkspaceSvc{listResult: []entities.WorkspaceSectionInfo{
			{Key: "mappings", Title: "Mappings", Count: 2},
		}}
		handler := New(svc)

		resp, err := handler.ListSections(t.Context(), connect.NewRequest(&workspacepb.ListSectionsRequest{}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.Sections, 1)
		assert.Equal(t, "mappings", resp.Msg.Sections[0].GetKey())
		assert.Equal(t, int32(2), resp.Msg.Sections[0].GetCount())
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockWorkspaceSvc{listErr: errors.New("boom")}
		handler := New(svc)

		_, err := handler.ListSections(t.Context(), connect.NewRequest(&workspacepb.ListSectionsRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_ExportWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockWorkspaceSvc{exportResult: []byte(`{"version":1}`)}
		handler := New(svc)

		resp, err := handler.ExportWorkspace(t.Context(), connect.NewRequest(&workspacepb.ExportWorkspaceRequest{
			SectionKeys: []string{"mappings"},
		}))
		require.NoError(t, err)
		assert.Equal(t, []byte(`{"version":1}`), resp.Msg.Payload)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockWorkspaceSvc{exportErr: errors.New("boom")}
		handler := New(svc)

		_, err := handler.ExportWorkspace(t.Context(), connect.NewRequest(&workspacepb.ExportWorkspaceRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_ValidateWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("Success_ReplaceStrategy", func(t *testing.T) {
		t.Parallel()
		svc := &mockWorkspaceSvc{validateResult: []entities.WorkspaceSectionReport{
			{Key: "mappings", Created: 2, Deleted: 1},
		}}
		handler := New(svc)

		resp, err := handler.ValidateWorkspace(t.Context(), connect.NewRequest(&workspacepb.ValidateWorkspaceRequest{
			Payload:  []byte(`{"version":1}`),
			Strategy: workspacepb.Strategy_STRATEGY_REPLACE,
		}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.Reports, 1)
		assert.Equal(t, "mappings", resp.Msg.Reports[0].GetKey())
		assert.Equal(t, entities.WorkspaceStrategyReplace, svc.gotStrategy)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockWorkspaceSvc{validateErr: errors.New("boom")}
		handler := New(svc)

		_, err := handler.ValidateWorkspace(t.Context(), connect.NewRequest(&workspacepb.ValidateWorkspaceRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_ImportWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("Success_MergeStrategy", func(t *testing.T) {
		t.Parallel()
		svc := &mockWorkspaceSvc{importResult: []entities.WorkspaceSectionResult{
			{Key: "mappings", Created: 2, Updated: 1},
		}}
		handler := New(svc)

		resp, err := handler.ImportWorkspace(t.Context(), connect.NewRequest(&workspacepb.ImportWorkspaceRequest{
			Payload:  []byte(`{"version":1}`),
			Strategy: workspacepb.Strategy_STRATEGY_MERGE,
		}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.Results, 1)
		assert.Equal(t, "mappings", resp.Msg.Results[0].GetKey())
		assert.Equal(t, entities.WorkspaceStrategyMerge, svc.gotStrategy)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockWorkspaceSvc{importErr: errors.New("boom")}
		handler := New(svc)

		_, err := handler.ImportWorkspace(t.Context(), connect.NewRequest(&workspacepb.ImportWorkspaceRequest{}))
		assert.Error(t, err)
	})
}
