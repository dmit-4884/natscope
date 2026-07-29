// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	settingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/settings/v1/settings"
)

// --- Mocks ---

type mockSettingsService struct {
	getResult    *entities.UserSettings
	getErr       error
	updateResult *entities.UserSettings
	updateErr    error
	resetResult  *entities.UserSettings
	resetErr     error
}

func (m *mockSettingsService) Get(_ context.Context) (*entities.UserSettings, error) {
	return m.getResult, m.getErr
}

func (m *mockSettingsService) Update(_ context.Context, _ *entities.UserSettingsUpdate) (*entities.UserSettings, error) {
	return m.updateResult, m.updateErr
}

func (m *mockSettingsService) Reset(_ context.Context) (*entities.UserSettings, error) {
	return m.resetResult, m.resetErr
}

// --- Tests ---

func TestHandler_GetSettings(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockSettingsService{getResult: entities.UserSettingsNew()}
		handler := New(svc)

		resp, err := handler.GetSettings(t.Context(), connect.NewRequest(&settingspb.GetSettingsRequest{}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Settings)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockSettingsService{getErr: errs.ErrSettingsNotFound}
		handler := New(svc)

		_, err := handler.GetSettings(t.Context(), connect.NewRequest(&settingspb.GetSettingsRequest{}))
		assert.ErrorIs(t, err, errs.ErrSettingsNotFound)
	})
}

func TestHandler_UpdateSettings(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockSettingsService{updateResult: entities.UserSettingsNew()}
		handler := New(svc)

		resp, err := handler.UpdateSettings(t.Context(), connect.NewRequest(&settingspb.UpdateSettingsRequest{}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Settings)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockSettingsService{updateErr: errors.New("db error")}
		handler := New(svc)

		_, err := handler.UpdateSettings(t.Context(), connect.NewRequest(&settingspb.UpdateSettingsRequest{}))
		assert.Error(t, err)
	})
}

func TestHandler_ResetSettings(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockSettingsService{resetResult: entities.UserSettingsNew()}
		handler := New(svc)

		resp, err := handler.ResetSettings(t.Context(), connect.NewRequest(&settingspb.ResetSettingsRequest{}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Settings)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockSettingsService{resetErr: errs.ErrSettingsNotFound}
		handler := New(svc)

		_, err := handler.ResetSettings(t.Context(), connect.NewRequest(&settingspb.ResetSettingsRequest{}))
		assert.ErrorIs(t, err, errs.ErrSettingsNotFound)
	})
}
