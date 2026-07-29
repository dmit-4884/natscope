// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore/bbstoretest"

	ptr "github.com/altessa-s/go-atlas/core/types/ptr"
	templatessvc "github.com/dmit-4884/natscope/internal/services/templates"
	templatesBbolt "github.com/dmit-4884/natscope/internal/storages/templates/bbolt"
)

// setupService wires the real Service over an isolated in-memory bbolt store.
func setupService(t *testing.T) templatessvc.Service {
	t.Helper()
	store, err := templatesBbolt.New(t.Context(), bbstoretest.NewMemoryDB(t))
	require.NoError(t, err)
	return New(store)
}

func TestCreate(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	created, err := svc.Create(t.Context(), &entities.MessageTemplateCreate{
		Name:        "orders",
		Subject:     "orders.new",
		MessageType: "api.v1.Order",
		Data:        `{"id":1}`,
		Headers:     map[string]string{"k": "v"},
		Wildcards:   []string{"a"},
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotEmpty(t, created.Id)
	assert.Equal(t, "orders", created.Name)
	assert.Equal(t, "orders.new", created.Subject)
	assert.Equal(t, map[string]string{"k": "v"}, created.Headers)

	// Persisted and retrievable by id.
	got, err := svc.Get(t.Context(), created.Id)
	require.NoError(t, err)
	assert.Equal(t, created.Id, got.Id)
	assert.Equal(t, "api.v1.Order", got.MessageType)
}

func TestCreate_NormalizesTrim(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	created, err := svc.Create(t.Context(), &entities.MessageTemplateCreate{
		Name:    "  spaced  ",
		Subject: "  s.x  ",
	})
	require.NoError(t, err)
	assert.Equal(t, "spaced", created.Name)
	assert.Equal(t, "s.x", created.Subject)
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	_, err := svc.Get(t.Context(), "missing")
	assert.ErrorIs(t, err, errs.ErrMessageTemplateNotFound)
}

func TestUpdate_PartialPreservesSiblings(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	created, err := svc.Create(t.Context(), &entities.MessageTemplateCreate{
		Name:    "orig",
		Subject: "orig.subject",
		Data:    "payload",
	})
	require.NoError(t, err)

	// Patch only Name; Subject and Data must survive (WithIgnoreNilValues).
	updated, err := svc.Update(t.Context(), &entities.MessageTemplateUpdate{
		Id:   created.Id,
		Name: ptr.Wrap("renamed"),
	})
	require.NoError(t, err)
	assert.Equal(t, "renamed", updated.Name)
	assert.Equal(t, "orig.subject", updated.Subject)
	assert.Equal(t, "payload", updated.Data)
}

func TestUpdate_HeadersReplaceSemantics(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	created, err := svc.Create(t.Context(), &entities.MessageTemplateCreate{
		Name:    "h",
		Headers: map[string]string{"old": "1"},
	})
	require.NoError(t, err)

	// nil Headers keeps the stored map.
	kept, err := svc.Update(t.Context(), &entities.MessageTemplateUpdate{Id: created.Id, Name: ptr.Wrap("h2")})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"old": "1"}, kept.Headers)

	// non-nil Headers replaces wholesale.
	replaced, err := svc.Update(t.Context(), &entities.MessageTemplateUpdate{
		Id:      created.Id,
		Headers: map[string]string{"new": "2"},
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"new": "2"}, replaced.Headers)
}

func TestUpdate_NotFound(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	_, err := svc.Update(t.Context(), &entities.MessageTemplateUpdate{
		Id:   "missing",
		Name: ptr.Wrap("x"),
	})
	assert.ErrorIs(t, err, errs.ErrMessageTemplateNotFound)
}

func TestDelete(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	created, err := svc.Create(t.Context(), &entities.MessageTemplateCreate{Name: "gone"})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(t.Context(), created.Id))

	_, err = svc.Get(t.Context(), created.Id)
	assert.ErrorIs(t, err, errs.ErrMessageTemplateNotFound)
}

func TestDelete_NotFound(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	err := svc.Delete(t.Context(), "missing")
	assert.ErrorIs(t, err, errs.ErrMessageTemplateNotFound)
}

func TestList(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	for _, name := range []string{"a", "b", "c"} {
		_, err := svc.Create(t.Context(), &entities.MessageTemplateCreate{Name: name})
		require.NoError(t, err)
	}

	out, err := svc.List(t.Context(), &entities.MessageTemplatesList{
		ListBase: entities.ListBase{IncludeTotalCount: true},
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Len(t, out.Items, 3)
	require.NotNil(t, out.Total)
	assert.Equal(t, int64(3), *out.Total)
}

func TestBulkCreate(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	n, err := svc.BulkCreate(t.Context(), []*entities.MessageTemplateCreate{
		{Name: "one"},
		{Name: "two"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	out, err := svc.List(t.Context(), &entities.MessageTemplatesList{})
	require.NoError(t, err)
	assert.Len(t, out.Items, 2)
}

func TestDeleteAll(t *testing.T) {
	t.Parallel()
	svc := setupService(t)

	for _, name := range []string{"x", "y"} {
		_, err := svc.Create(t.Context(), &entities.MessageTemplateCreate{Name: name})
		require.NoError(t, err)
	}

	n, err := svc.DeleteAll(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	out, err := svc.List(t.Context(), &entities.MessageTemplatesList{})
	require.NoError(t, err)
	assert.Empty(t, out.Items)
}
