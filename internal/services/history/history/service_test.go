// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

// --- Mocks ---

type mockStorage struct {
	saveErr    error
	listResult *entities.List[entities.PublishHistories]
	listErr    error

	saveCalled bool
	saveInput  *entities.PublishHistory
}

func (m *mockStorage) Save(_ context.Context, in *entities.PublishHistory) error {
	m.saveCalled = true
	m.saveInput = in
	return m.saveErr
}

func (m *mockStorage) List(_ context.Context, _ *entities.PublishHistoryList) (*entities.List[entities.PublishHistories], error) {
	return m.listResult, m.listErr
}

// --- Tests ---

func TestService_Record(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *entities.PublishHistoryCreate
		saveErr error
		wantErr bool
	}{
		{
			name: "Success",
			input: &entities.PublishHistoryCreate{
				ConnectionURL: "nats://localhost:4222",
				Stream:        "ORDERS",
				Subject:       "orders.created",
				MessageType:   "api.v1.Order",
				PayloadJSON:   `{"id": 1}`,
				PayloadSize:   8,
				Success:       true,
			},
		},
		{
			name: "StorageError",
			input: &entities.PublishHistoryCreate{
				Subject: "test",
				Success: false,
			},
			saveErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &mockStorage{saveErr: tt.saveErr}
			svc := New(store)

			result, err := svc.Record(t.Context(), tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotEmpty(t, result.Id)
				assert.True(t, store.saveCalled)
			}
		})
	}
}

func TestService_Record_PayloadSize(t *testing.T) {
	t.Parallel()

	t.Run("KeepsEncodedSizeFromCaller", func(t *testing.T) {
		t.Parallel()

		store := &mockStorage{}
		svc := New(store)

		got, err := svc.Record(t.Context(), &entities.PublishHistoryCreate{
			ConnectionURL: "nats://localhost:4222",
			Stream:        "ORDERS",
			Subject:       "orders.created",
			EncodingType:  entities.EncodingTypeProtobuf,
			MessageType:   "api.v1.Order",
			PayloadJSON:   `{"id":1}`,
			PayloadSize:   2,
			Success:       true,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, got.PayloadSize, "encoded payload size must win over the JSON source length")
		require.NotNil(t, store.saveInput)
		assert.Equal(t, 2, store.saveInput.PayloadSize)
	})

	t.Run("FallsBackToJSONLengthWhenOmitted", func(t *testing.T) {
		t.Parallel()

		store := &mockStorage{}
		svc := New(store)

		got, err := svc.Record(t.Context(), &entities.PublishHistoryCreate{
			ConnectionURL: "nats://localhost:4222",
			Stream:        "ORDERS",
			Subject:       "orders.created",
			EncodingType:  entities.EncodingTypeJSON,
			PayloadJSON:   `{"id":1}`,
			Success:       true,
		})

		require.NoError(t, err)
		assert.Equal(t, len(`{"id":1}`), got.PayloadSize)
	})
}

func TestService_List(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		expected := &entities.List[entities.PublishHistories]{
			Items: entities.PublishHistories{
				{BaseEntity: entities.BaseEntity{Id: "h1"}},
			},
		}
		store := &mockStorage{listResult: expected}
		svc := New(store)

		result, err := svc.List(t.Context(), &entities.PublishHistoryList{})
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
	})

	t.Run("StorageError", func(t *testing.T) {
		t.Parallel()
		store := &mockStorage{listErr: errors.New("db error")}
		svc := New(store)

		_, err := svc.List(t.Context(), &entities.PublishHistoryList{})
		assert.Error(t, err)
	})
}
