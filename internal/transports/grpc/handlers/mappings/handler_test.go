// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package mappings

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	ptr "github.com/altessa-s/go-atlas/core/types/ptr"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
)

// --- Mocks ---

type mockMappingsService struct {
	createResult   *entities.SubjectMapping
	createErr      error
	getResult      *entities.SubjectMapping
	getErr         error
	updateResult   *entities.SubjectMapping
	updateErr      error
	listResult     *entities.List[entities.SubjectMappings]
	listErr        error
	deleteErr      error
	bulkSaveErr    error
	bulkSaveResult *entities.SubjectMappingBulkSaveResult
	getAllResult   entities.SubjectMappings
	getAllErr      error

	createCalled   bool
	deleteCalled   bool
	deleteID       string
	bulkSaveCalled bool
	bulkSaveCount  int
}

func (m *mockMappingsService) Create(_ context.Context, in *entities.SubjectMappingCreate) (*entities.SubjectMapping, error) {
	m.createCalled = true
	if m.createResult != nil {
		return m.createResult, m.createErr
	}
	mapping := entities.SubjectMappingNew(func(sm *entities.SubjectMapping) {
		sm.Pattern = in.Pattern
		sm.MessageType = in.MessageType
		sm.SourceID = in.SourceID
	})
	return mapping, m.createErr
}

func (m *mockMappingsService) Get(_ context.Context, _ string) (*entities.SubjectMapping, error) {
	return m.getResult, m.getErr
}

func (m *mockMappingsService) Update(_ context.Context, _ *entities.SubjectMappingUpdate) (*entities.SubjectMapping, error) {
	return m.updateResult, m.updateErr
}

func (m *mockMappingsService) List(_ context.Context, _ *entities.SubjectMappingsList) (*entities.List[entities.SubjectMappings], error) {
	return m.listResult, m.listErr
}

func (m *mockMappingsService) Delete(_ context.Context, id string) error {
	m.deleteCalled = true
	m.deleteID = id
	return m.deleteErr
}

func (m *mockMappingsService) BulkSave(
	_ context.Context,
	mappings entities.SubjectMappings,
) (*entities.SubjectMappingBulkSaveResult, error) {
	m.bulkSaveCalled = true
	m.bulkSaveCount = len(mappings)
	if m.bulkSaveErr != nil {
		return nil, m.bulkSaveErr
	}
	if m.bulkSaveResult != nil {
		return m.bulkSaveResult, nil
	}
	return &entities.SubjectMappingBulkSaveResult{Created: len(mappings)}, nil
}

func (m *mockMappingsService) GetAll(_ context.Context) (entities.SubjectMappings, error) {
	return m.getAllResult, m.getAllErr
}

func (m *mockMappingsService) Resolver(_ context.Context) *natsutil.MappingResolver {
	return natsutil.NewMappingResolver(nil)
}

// stubProtoService is a minimal protosvc.Registry stub; only MappingHealth is exercised explicitly.
type stubProtoService struct {
	protosvc.Registry
	healthResult []entities.SubjectMappingHealth
	healthErr    error
}

func (s *stubProtoService) MappingHealth(_ context.Context, ids []string) ([]entities.SubjectMappingHealth, error) {
	if s.healthResult != nil {
		return s.healthResult, s.healthErr
	}
	out := make([]entities.SubjectMappingHealth, len(ids))
	for i, id := range ids {
		out[i] = entities.SubjectMappingHealth{Id: id, Health: entities.MappingHealthOK}
	}
	return out, s.healthErr
}

// --- Tests ---

func TestHandler_Create(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockMappingsService{}
		handler := New(svc, &stubProtoService{})

		resp, err := handler.CreateMapping(t.Context(), connect.NewRequest(&mappingspb.CreateMappingRequest{
			Pattern:     "orders.*",
			MessageType: "api.v1.Order",
			SourceId:    "src-1",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Msg.Mapping)
		assert.True(t, svc.createCalled)
		assert.Equal(t, "src-1", resp.Msg.Mapping.SourceId)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockMappingsService{createErr: errs.ErrMappingPatternAlreadyInUse}
		handler := New(svc, &stubProtoService{})

		resp, err := handler.CreateMapping(t.Context(), connect.NewRequest(&mappingspb.CreateMappingRequest{
			Pattern:     "dup.*",
			MessageType: "api.v1.Order",
			SourceId:    "src-1",
		}))
		assert.ErrorIs(t, err, errs.ErrMappingPatternAlreadyInUse)
		assert.Nil(t, resp)
	})
}

func TestHandler_List(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockMappingsService{
			listResult: &entities.List[entities.SubjectMappings]{
				Items: entities.SubjectMappings{
					entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
						m.Pattern = "orders.*"
						m.SourceID = "src-1"
					}),
				},
				NextCursor: ptr.Wrap("next"),
			},
		}
		handler := New(svc, &stubProtoService{})

		resp, err := handler.ListMappings(t.Context(), connect.NewRequest(&mappingspb.ListMappingsRequest{}))
		require.NoError(t, err)
		assert.Len(t, resp.Msg.Mappings, 1)
		assert.Equal(t, "src-1", resp.Msg.Mappings[0].SourceId)
		require.NotNil(t, resp.Msg.NextPageToken)
		assert.Equal(t, "next", *resp.Msg.NextPageToken)
	})

	t.Run("WithPagination", func(t *testing.T) {
		t.Parallel()
		svc := &mockMappingsService{
			listResult: &entities.List[entities.SubjectMappings]{},
		}
		handler := New(svc, &stubProtoService{})

		_, err := handler.ListMappings(t.Context(), connect.NewRequest(&mappingspb.ListMappingsRequest{
			PageSize:  10,
			PageToken: "c1",
		}))
		require.NoError(t, err)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockMappingsService{listErr: errors.New("db error")}
		handler := New(svc, &stubProtoService{})

		_, err := handler.ListMappings(t.Context(), connect.NewRequest(&mappingspb.ListMappingsRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockMappingsService{
			getResult: &entities.SubjectMapping{
				BaseEntity: entities.BaseEntity{Id: "m-1"},
			},
		}
		handler := New(svc, &stubProtoService{})

		resp, err := handler.DeleteMapping(t.Context(), connect.NewRequest(&mappingspb.DeleteMappingRequest{Id: "m-1"}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, svc.deleteCalled)
		assert.Equal(t, "m-1", svc.deleteID)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockMappingsService{deleteErr: errs.ErrMappingNotFound}
		handler := New(svc, &stubProtoService{})

		_, err := handler.DeleteMapping(t.Context(), connect.NewRequest(&mappingspb.DeleteMappingRequest{Id: "x"}))
		assert.ErrorIs(t, err, errs.ErrMappingNotFound)
	})
}

func TestHandler_BulkSave(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockMappingsService{}
		handler := New(svc, &stubProtoService{})

		resp, err := handler.BatchSaveMappings(t.Context(), connect.NewRequest(&mappingspb.BatchSaveMappingsRequest{
			Mappings: []*mappingspb.MappingBulkItem{
				{Pattern: "a.*", MessageType: "TypeA", SourceId: "src-1"},
				{Pattern: "b.*", MessageType: "TypeB", SourceId: "src-1"},
			},
		}))
		require.NoError(t, err)
		assert.Equal(t, int32(2), resp.Msg.Created)
		assert.True(t, svc.bulkSaveCalled)
		assert.Equal(t, 2, svc.bulkSaveCount)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockMappingsService{bulkSaveErr: errors.New("db error")}
		handler := New(svc, &stubProtoService{})

		_, err := handler.BatchSaveMappings(t.Context(), connect.NewRequest(&mappingspb.BatchSaveMappingsRequest{
			Mappings: []*mappingspb.MappingBulkItem{
				{Pattern: "a.*", MessageType: "TypeA", SourceId: "src-1"},
			},
		}))
		assert.Error(t, err)
	})

	t.Run("EmptyMappings", func(t *testing.T) {
		t.Parallel()
		svc := &mockMappingsService{}
		handler := New(svc, &stubProtoService{})

		resp, err := handler.BatchSaveMappings(t.Context(), connect.NewRequest(&mappingspb.BatchSaveMappingsRequest{
			Mappings: []*mappingspb.MappingBulkItem{},
		}))
		require.NoError(t, err)
		assert.Equal(t, int32(0), resp.Msg.Created)
	})
}

func TestHandler_HealthBatch(t *testing.T) {
	t.Parallel()

	t.Run("EmptyIds", func(t *testing.T) {
		t.Parallel()
		handler := New(&mockMappingsService{}, &stubProtoService{})
		resp, err := handler.BatchCheckMappingHealth(t.Context(), connect.NewRequest(&mappingspb.BatchCheckMappingHealthRequest{}))
		require.NoError(t, err)
		assert.Empty(t, resp.Msg.Items)
	})

	t.Run("PassesIdsThrough", func(t *testing.T) {
		t.Parallel()
		proto := &stubProtoService{healthResult: []entities.SubjectMappingHealth{
			{Id: "a", Health: entities.MappingHealthOK},
			{Id: "b", Health: entities.MappingHealthSelectionMissing, Detail: "no selection"},
		}}
		handler := New(&mockMappingsService{}, proto)

		resp, err := handler.BatchCheckMappingHealth(t.Context(), connect.NewRequest(&mappingspb.BatchCheckMappingHealthRequest{
			Ids: []string{"a", "b"},
		}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.Items, 2)
		assert.Equal(t, "ok", resp.Msg.Items[0].Health)
		assert.Equal(t, "selection_missing", resp.Msg.Items[1].Health)
		assert.Equal(t, "no selection", resp.Msg.Items[1].Detail)
	})
}
