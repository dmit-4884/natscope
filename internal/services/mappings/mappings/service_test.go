// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package mappings

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

// --- Mocks ---

type mockStorage struct {
	saveErr        error
	getResult      *entities.SubjectMapping
	getErr         error
	listResult     *entities.List[entities.SubjectMappings]
	listErr        error
	listAllResult  entities.SubjectMappings
	listAllErr     error
	deleteErr      error
	bulkSaveErr    error
	bulkSaveResult *entities.SubjectMappingBulkSaveResult

	saveCalled     bool
	deleteCalled   bool
	deleteID       string
	bulkSaveCalled bool
	bulkSaveItems  entities.SubjectMappings
	listAllCalled  bool
}

func (m *mockStorage) Save(_ context.Context, _ *entities.SubjectMapping) error {
	m.saveCalled = true
	return m.saveErr
}
func (m *mockStorage) Get(_ context.Context, _ string) (*entities.SubjectMapping, error) {
	return m.getResult, m.getErr
}
func (m *mockStorage) List(_ context.Context, _ *entities.SubjectMappingsList) (*entities.List[entities.SubjectMappings], error) {
	return m.listResult, m.listErr
}
func (m *mockStorage) ListAll(_ context.Context) (entities.SubjectMappings, error) {
	m.listAllCalled = true
	return m.listAllResult, m.listAllErr
}
func (m *mockStorage) Delete(_ context.Context, id string) error {
	m.deleteCalled = true
	m.deleteID = id
	return m.deleteErr
}
func (m *mockStorage) Exists(_ context.Context, _ string) (bool, error) { return false, nil }
func (m *mockStorage) BulkSave(
	_ context.Context,
	mappings entities.SubjectMappings,
) (*entities.SubjectMappingBulkSaveResult, error) {
	m.bulkSaveCalled = true
	m.bulkSaveItems = mappings
	if m.bulkSaveErr != nil {
		return nil, m.bulkSaveErr
	}
	if m.bulkSaveResult != nil {
		return m.bulkSaveResult, nil
	}
	return &entities.SubjectMappingBulkSaveResult{}, nil
}

// --- Tests ---

func TestService_Create(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		svc := New(store)

		result, err := svc.Create(t.Context(), &entities.SubjectMappingCreate{
			Pattern:     "orders.*",
			MessageType: "api.v1.Order",
			SourceID:    "src-1",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, store.saveCalled)
	})

	t.Run("DuplicatePattern", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{saveErr: errs.ErrMappingPatternAlreadyInUse}
		svc := New(store)

		_, err := svc.Create(t.Context(), &entities.SubjectMappingCreate{
			Pattern:     "orders.*",
			MessageType: "api.v1.Order",
			SourceID:    "src-1",
		})
		assert.ErrorIs(t, err, errs.ErrMappingPatternAlreadyInUse)
	})

	t.Run("MissingSourceID", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		svc := New(store)

		_, err := svc.Create(t.Context(), &entities.SubjectMappingCreate{
			Pattern:     "orders.*",
			MessageType: "api.v1.Order",
		})
		assert.ErrorIs(t, err, errs.ErrMappingSourceIDRequired)
	})
}

func TestService_Get(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		mapping := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
			m.Pattern = "orders.*"
		})
		store := &mockStorage{getResult: mapping}
		svc := New(store)

		result, err := svc.Get(t.Context(), mapping.Id)
		require.NoError(t, err)
		assert.Equal(t, "orders.*", result.Pattern)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{getErr: errs.ErrMappingNotFound}
		svc := New(store)

		_, err := svc.Get(t.Context(), "nonexistent")
		assert.ErrorIs(t, err, errs.ErrMappingNotFound)
	})
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		existing := &entities.SubjectMapping{
			BaseEntity: entities.BaseEntity{Id: "m-1"},
		}
		store := &mockStorage{getResult: existing}
		svc := New(store)

		err := svc.Delete(t.Context(), "m-1")
		require.NoError(t, err)
		assert.True(t, store.deleteCalled)
		assert.Equal(t, "m-1", store.deleteID)
	})

	t.Run("StorageError", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{deleteErr: errors.New("db error")}
		svc := New(store)

		err := svc.Delete(t.Context(), "m-1")
		assert.Error(t, err)
	})
}

func TestService_BulkSave(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		svc := New(store)

		mappings := entities.SubjectMappings{
			{Pattern: "a.*", MessageType: "TypeA", SourceID: "src-1"},
			{Pattern: "b.*", MessageType: "TypeB", SourceID: "src-1"},
		}

		_, err := svc.BulkSave(t.Context(), mappings)
		require.NoError(t, err)
		assert.True(t, store.bulkSaveCalled)
	})

	t.Run("GeneratesIdForEmptyId", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		svc := New(store)

		mappings := entities.SubjectMappings{
			{Pattern: "a.*", MessageType: "TypeA", SourceID: "src-1"}, // No Id set
		}

		_, err := svc.BulkSave(t.Context(), mappings)
		require.NoError(t, err)
		assert.Empty(t, mappings[0].Id, "input must not be mutated")
		require.Len(t, store.bulkSaveItems, 1)
		assert.NotEmpty(t, store.bulkSaveItems[0].Id, "Id should be generated for the storage call")
	})

	t.Run("MissingSourceID", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{}
		svc := New(store)

		_, err := svc.BulkSave(t.Context(), entities.SubjectMappings{
			{Pattern: "a.*", MessageType: "TypeA"},
		})
		assert.ErrorIs(t, err, errs.ErrMappingSourceIDRequired)
	})

	t.Run("StorageError", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{bulkSaveErr: errors.New("db error")}
		svc := New(store)

		_, err := svc.BulkSave(t.Context(), entities.SubjectMappings{})
		assert.Error(t, err)
	})
}

func TestService_GetAll(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		expected := entities.SubjectMappings{
			{BaseEntity: entities.BaseEntity{Id: "m1"}, Pattern: "orders.*"},
		}
		store := &mockStorage{listAllResult: expected}
		svc := New(store)

		result, err := svc.GetAll(t.Context())
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.True(t, store.listAllCalled)
	})

	t.Run("StorageError", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{listAllErr: errors.New("db error")}
		svc := New(store)

		_, err := svc.GetAll(t.Context())
		assert.Error(t, err)
	})
}
