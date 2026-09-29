// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

func TestService_GetSidebarLayout(t *testing.T) {
	t.Parallel()

	t.Run("empty when nothing was saved", func(t *testing.T) {
		t.Parallel()
		svc := New(&mockStorage{exists: true}, &mockLayouts{}, nil)

		got, err := svc.GetSidebarLayout(t.Context(), "conn-1")
		require.NoError(t, err)
		assert.Equal(t, &entities.SidebarLayout{ConnectionID: "conn-1"}, got)
	})

	t.Run("saved layout", func(t *testing.T) {
		t.Parallel()
		saved := &entities.SidebarLayout{ConnectionID: "conn-1", Streams: entities.SectionLayout{Pinned: []string{"ORDERS"}}}
		svc := New(&mockStorage{exists: true}, &mockLayouts{layouts: map[string]*entities.SidebarLayout{"conn-1": saved}}, nil)

		got, err := svc.GetSidebarLayout(t.Context(), "conn-1")
		require.NoError(t, err)
		assert.Equal(t, saved, got)
	})

	t.Run("unknown connection", func(t *testing.T) {
		t.Parallel()
		svc := New(&mockStorage{}, &mockLayouts{}, nil)

		_, err := svc.GetSidebarLayout(t.Context(), "conn-1")
		require.ErrorIs(t, err, errs.ErrSavedConnectionNotFound)
	})
}

func TestService_UpdateSidebarLayout(t *testing.T) {
	t.Parallel()

	t.Run("replaces only the sections that are set", func(t *testing.T) {
		t.Parallel()
		layouts := &mockLayouts{layouts: map[string]*entities.SidebarLayout{
			"conn-1": {ConnectionID: "conn-1", KV: entities.SectionLayout{Pinned: []string{"config"}}},
		}}
		svc := New(&mockStorage{exists: true}, layouts, nil)

		got, err := svc.UpdateSidebarLayout(t.Context(), &entities.SidebarLayoutUpdate{
			ConnectionID: " conn-1 ",
			Streams:      &entities.SectionLayout{Pinned: []string{"ORDERS"}, Order: []string{"EVENTS"}},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"ORDERS"}, got.Streams.Pinned)
		assert.Equal(t, []string{"EVENTS"}, got.Streams.Order)
		assert.Equal(t, []string{"config"}, got.KV.Pinned, "an unset section is kept")
		assert.False(t, got.UpdatedAt.IsZero())
	})

	t.Run("drops blank and repeated names and pinned names from the order", func(t *testing.T) {
		t.Parallel()
		svc := New(&mockStorage{exists: true}, &mockLayouts{}, nil)

		got, err := svc.UpdateSidebarLayout(t.Context(), &entities.SidebarLayoutUpdate{
			ConnectionID: "conn-1",
			Objects: &entities.SectionLayout{
				Pinned: []string{"assets", " ", "assets", "logs"},
				Order:  []string{"logs", "backups", "backups ", "", "media"},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"assets", "logs"}, got.Objects.Pinned)
		assert.Equal(t, []string{"backups", "media"}, got.Objects.Order)
	})

	t.Run("unknown connection", func(t *testing.T) {
		t.Parallel()
		layouts := &mockLayouts{}
		svc := New(&mockStorage{}, layouts, nil)

		_, err := svc.UpdateSidebarLayout(t.Context(), &entities.SidebarLayoutUpdate{
			ConnectionID: "conn-1",
			Streams:      &entities.SectionLayout{Pinned: []string{"ORDERS"}},
		})
		require.ErrorIs(t, err, errs.ErrSavedConnectionNotFound)
		assert.Empty(t, layouts.layouts, "no layout is stored for an unknown connection")
	})
}

func TestService_DeleteRemovesSidebarLayout(t *testing.T) {
	t.Parallel()

	t.Run("layout is deleted with the connection", func(t *testing.T) {
		t.Parallel()
		layouts := &mockLayouts{}
		svc := New(&mockStorage{}, layouts, &mockNATSService{})

		require.NoError(t, svc.Delete(t.Context(), "conn-1"))
		assert.Equal(t, []string{"conn-1"}, layouts.deleted)
	})

	t.Run("a layout failure does not fail the delete", func(t *testing.T) {
		t.Parallel()
		layouts := &mockLayouts{deleteErr: errors.New("disk full")}
		svc := New(&mockStorage{}, layouts, &mockNATSService{})

		require.NoError(t, svc.Delete(t.Context(), "conn-1"))
	})
}
