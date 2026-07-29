// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore/bbstoretest"

	ptr "github.com/altessa-s/go-atlas/core/types/ptr"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
	settingsBbolt "github.com/dmit-4884/natscope/internal/storages/settings/bbolt"
)

func setupService(t *testing.T) settingssvc.Service {
	t.Helper()
	store, err := settingsBbolt.New(t.Context(), bbstoretest.NewMemoryDB(t))
	require.NoError(t, err)
	return New(store)
}

func TestGet_NoSettings(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	result, err := svc.Get(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Id)
	assert.Nil(t, result.Messages)
	assert.Nil(t, result.Live)
	assert.Nil(t, result.Display)
	assert.Nil(t, result.Publish)
}

func TestGet_ExistingSettings(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	// Save settings via Update first
	_, err := svc.Update(t.Context(), &entities.UserSettingsUpdate{
		Messages: &entities.MessageSettings{
			DefaultPageSize: ptr.Wrap(int32(50)),
		},
	})
	require.NoError(t, err)

	result, err := svc.Get(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Messages)
	require.NotNil(t, result.Messages.DefaultPageSize)
	assert.Equal(t, int32(50), *result.Messages.DefaultPageSize)
}

func TestUpdate_CreateNew(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	result, err := svc.Update(t.Context(), &entities.UserSettingsUpdate{
		Messages: &entities.MessageSettings{
			FetchMethod:     ptr.Wrap("direct"),
			DefaultPageSize: ptr.Wrap(int32(25)),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Id)
	require.NotNil(t, result.Messages)
	assert.Equal(t, "direct", *result.Messages.FetchMethod)
	assert.Equal(t, int32(25), *result.Messages.DefaultPageSize)
}

func TestUpdate_PartialUpdate(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	// Create initial settings with Messages and Live
	_, err := svc.Update(t.Context(), &entities.UserSettingsUpdate{
		Messages: &entities.MessageSettings{
			FetchMethod:     ptr.Wrap("direct"),
			DefaultPageSize: ptr.Wrap(int32(25)),
		},
		Live: &entities.LiveSettings{
			SubscriptionMode: ptr.Wrap("core_nats"),
		},
	})
	require.NoError(t, err)

	// Update only Messages.DefaultPageSize
	result, err := svc.Update(t.Context(), &entities.UserSettingsUpdate{
		Messages: &entities.MessageSettings{
			DefaultPageSize: ptr.Wrap(int32(100)),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Messages.DefaultPageSize updated
	require.NotNil(t, result.Messages)
	require.NotNil(t, result.Messages.DefaultPageSize)
	assert.Equal(t, int32(100), *result.Messages.DefaultPageSize)

	// Messages.FetchMethod preserved (WithIgnoreNilValues)
	require.NotNil(t, result.Messages.FetchMethod)
	assert.Equal(t, "direct", *result.Messages.FetchMethod)

	// Live settings untouched
	require.NotNil(t, result.Live)
	require.NotNil(t, result.Live.SubscriptionMode)
	assert.Equal(t, "core_nats", *result.Live.SubscriptionMode)
}

func TestUpdate_MultipleGroups(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	result, err := svc.Update(t.Context(), &entities.UserSettingsUpdate{
		Messages: &entities.MessageSettings{
			DefaultPageSize: ptr.Wrap(int32(50)),
		},
		Live: &entities.LiveSettings{
			SubscriptionMode: ptr.Wrap("jetstream_ordered"),
			MaxDisplayRate:   ptr.Wrap(int32(100)),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Messages)
	assert.Equal(t, int32(50), *result.Messages.DefaultPageSize)
	require.NotNil(t, result.Live)
	assert.Equal(t, "jetstream_ordered", *result.Live.SubscriptionMode)
	assert.Equal(t, int32(100), *result.Live.MaxDisplayRate)
}

func TestUpdate_BehaviorPartialUpdate(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	// Seed a Messages section plus one behavior toggle.
	_, err := svc.Update(t.Context(), &entities.UserSettingsUpdate{
		Messages: &entities.MessageSettings{
			FetchMethod: ptr.Wrap("direct"),
		},
		Behavior: &entities.BehaviorSettings{
			ConfirmDeleteConsumer: ptr.Wrap(false),
		},
	})
	require.NoError(t, err)

	// Update only a different behavior toggle.
	result, err := svc.Update(t.Context(), &entities.UserSettingsUpdate{
		Behavior: &entities.BehaviorSettings{
			ConfirmDeleteMessage: ptr.Wrap(false),
			SecureDeleteDefault:  ptr.Wrap(true),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// New toggles applied.
	require.NotNil(t, result.Behavior)
	require.NotNil(t, result.Behavior.ConfirmDeleteMessage)
	assert.False(t, *result.Behavior.ConfirmDeleteMessage)
	require.NotNil(t, result.Behavior.SecureDeleteDefault)
	assert.True(t, *result.Behavior.SecureDeleteDefault)

	// Previously-set behavior toggle preserved (WithIgnoreNilValues).
	require.NotNil(t, result.Behavior.ConfirmDeleteConsumer)
	assert.False(t, *result.Behavior.ConfirmDeleteConsumer)

	// Sibling Messages section untouched.
	require.NotNil(t, result.Messages)
	require.NotNil(t, result.Messages.FetchMethod)
	assert.Equal(t, "direct", *result.Messages.FetchMethod)

	// Behavior toggles left unset stay nil (tri-state default = confirm).
	assert.Nil(t, result.Behavior.ConfirmDeleteKvKey)
	assert.Nil(t, result.Behavior.ConfirmDeleteObject)
	assert.Nil(t, result.Behavior.ConfirmPurgeKvHistory)
}

func TestReset(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	// Save some settings
	_, err := svc.Update(t.Context(), &entities.UserSettingsUpdate{
		Messages: &entities.MessageSettings{
			DefaultPageSize: ptr.Wrap(int32(50)),
		},
	})
	require.NoError(t, err)

	// Reset
	result, err := svc.Reset(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Nil(t, result.Messages)
	assert.Nil(t, result.Live)
	assert.Nil(t, result.Display)
	assert.Nil(t, result.Publish)

	// Verify Get also returns defaults
	got, err := svc.Get(t.Context())
	require.NoError(t, err)
	assert.Nil(t, got.Messages)
	assert.Nil(t, got.Live)
}
