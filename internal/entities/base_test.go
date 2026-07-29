// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check func(t *testing.T, e *BaseEntity)
	}{
		{
			name: "IdIsValidUUID",
			check: func(t *testing.T, e *BaseEntity) {
				_, err := uuid.Parse(e.Id)
				require.NoError(t, err, "Id should be a valid UUID")
			},
		},
		{
			name: "CreatedAtIsPositive",
			check: func(t *testing.T, e *BaseEntity) {
				assert.False(t, e.CreatedAt.IsZero(), "CreatedAt should be set")
			},
		},
		{
			name: "UpdatedAtIsPositive",
			check: func(t *testing.T, e *BaseEntity) {
				assert.False(t, e.UpdatedAt.IsZero(), "UpdatedAt should be set")
			},
		},
		{
			name: "CreatedAtEqualsUpdatedAt",
			check: func(t *testing.T, e *BaseEntity) {
				assert.Equal(t, e.CreatedAt, e.UpdatedAt, "CreatedAt and UpdatedAt should be equal on creation")
			},
		},
		{
			name: "DeletedAtIsNil",
			check: func(t *testing.T, e *BaseEntity) {
				assert.Nil(t, e.DeletedAt, "DeletedAt should be nil for new entity")
			},
		},
		{
			name: "TimestampsAreRecent",
			check: func(t *testing.T, e *BaseEntity) {
				now := time.Now().UnixMilli()
				assert.InDelta(t, now, e.CreatedAt.UnixMilli(), 1000, "CreatedAt should be within 1s of now")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New()
			require.NotNil(t, e)
			tt.check(t, e)
		})
	}
}

func TestNew_Uniqueness(t *testing.T) {
	t.Parallel()

	e1 := New()
	e2 := New()
	assert.NotEqual(t, e1.Id, e2.Id, "two new entities should have different Ids")
}

func TestBaseEntity_IsDeleted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		deletedAt *time.Time
		want      bool
	}{
		{name: "NotDeleted_NilDeletedAt", deletedAt: nil, want: false},
		{name: "Deleted_WithTimestamp", deletedAt: ptrTime(time.Now().UTC()), want: true},
		{name: "Deleted_ZeroTimestamp", deletedAt: ptrTime(time.Time{}), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &BaseEntity{DeletedAt: tt.deletedAt}
			assert.Equal(t, tt.want, e.IsDeleted())
		})
	}
}

func TestBaseEntity_UpdateTimestamps(t *testing.T) {
	t.Parallel()

	t.Run("UpdatesUpdatedAt", func(t *testing.T) {
		t.Parallel()
		e := New()
		oldCreatedAt := e.CreatedAt
		// Seed a known-old UpdatedAt instead of sleeping so the assertion is
		// deterministic and race-free under t.Parallel.
		e.UpdatedAt = time.UnixMilli(1).UTC()

		e.UpdateTimestamps()

		assert.True(t, e.UpdatedAt.After(time.UnixMilli(1).UTC()), "UpdatedAt should advance")
		assert.Equal(t, oldCreatedAt, e.CreatedAt, "CreatedAt should not change")
	})

	t.Run("SetsCreatedAtIfZero", func(t *testing.T) {
		t.Parallel()
		e := &BaseEntity{CreatedAt: time.Time{}}

		e.UpdateTimestamps()

		assert.False(t, e.CreatedAt.IsZero(), "CreatedAt should be set when it was zero")
		assert.Equal(t, e.CreatedAt, e.UpdatedAt, "CreatedAt and UpdatedAt should be equal")
	})

	t.Run("DoesNotOverrideCreatedAt", func(t *testing.T) {
		t.Parallel()
		e := &BaseEntity{CreatedAt: time.UnixMilli(12345).UTC()}

		e.UpdateTimestamps()

		assert.Equal(t, time.UnixMilli(12345).UTC(), e.CreatedAt, "CreatedAt should not be overwritten")
		assert.NotEqual(t, time.UnixMilli(12345).UTC(), e.UpdatedAt, "UpdatedAt should be updated")
	})
}

func TestBaseEntity_BeforeUpdate(t *testing.T) {
	t.Parallel()

	e := New()
	oldEtag := e.Etag
	// Seed a known-old UpdatedAt instead of sleeping (deterministic, race-free).
	e.UpdatedAt = time.UnixMilli(1).UTC()

	e.BeforeUpdate()

	assert.True(t, e.UpdatedAt.After(time.UnixMilli(1).UTC()), "UpdatedAt should advance")
	assert.NotEqual(t, oldEtag, e.Etag, "BeforeUpdate must rotate the Etag")
	assert.NotEmpty(t, e.Etag, "Etag must remain populated after rotation")
}

func TestBaseEntity_UpdateEtag(t *testing.T) {
	t.Parallel()

	e := New()
	first := e.Etag
	require.NotEmpty(t, first, "New() must seed an Etag")

	e.UpdateEtag()
	assert.NotEqual(t, first, e.Etag, "UpdateEtag must rotate the Etag")
	assert.NotEmpty(t, e.Etag)
}

func TestBaseEntity_Restore(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	deletedAt := now.Add(-1 * time.Second)
	e := &BaseEntity{
		Id:        uuid.New().String(),
		Etag:      "old",
		CreatedAt: now.Add(-5 * time.Second),
		UpdatedAt: now.Add(-5 * time.Second),
		DeletedAt: &deletedAt,
	}

	e.Restore()

	assert.Nil(t, e.DeletedAt, "Restore must clear DeletedAt")
	assert.NotEqual(t, "old", e.Etag, "Restore must rotate the Etag")
	assert.False(t, e.UpdatedAt.Before(now), "Restore must refresh UpdatedAt")
}

func TestBaseEntity_Equal(t *testing.T) {
	t.Parallel()

	t.Run("BothNil", func(t *testing.T) {
		t.Parallel()
		var a, b *BaseEntity
		assert.True(t, a.Equal(b))
	})

	t.Run("OneNil", func(t *testing.T) {
		t.Parallel()
		a := New()
		var b *BaseEntity
		assert.False(t, a.Equal(b))
	})

	t.Run("SameValues", func(t *testing.T) {
		t.Parallel()
		a := New()
		b := *a // copy
		assert.True(t, a.Equal(&b))
	})

	t.Run("DifferentEtag", func(t *testing.T) {
		t.Parallel()
		a := New()
		b := *a
		b.Etag = "different"
		assert.False(t, a.Equal(&b))
	})

	t.Run("DifferentDeletedAt", func(t *testing.T) {
		t.Parallel()
		a := New()
		b := *a
		ts := time.UnixMilli(123).UTC()
		b.DeletedAt = &ts
		assert.False(t, a.Equal(&b))
	})

	t.Run("BothDeletedAtNil", func(t *testing.T) {
		t.Parallel()
		a := New()
		b := *a
		assert.True(t, a.Equal(&b))
	})

	t.Run("EqualDeletedAtTimestamps", func(t *testing.T) {
		t.Parallel()
		a := New()
		ts := time.UnixMilli(123).UTC()
		a.DeletedAt = &ts
		b := *a
		ts2 := time.UnixMilli(123).UTC()
		b.DeletedAt = &ts2
		assert.True(t, a.Equal(&b))
	})
}

func TestNew_SeedsEtag(t *testing.T) {
	t.Parallel()

	e := New()
	require.NotEmpty(t, e.Etag, "New() must seed an Etag")
	// 32 hex chars (UUID v4 without hyphens).
	assert.Len(t, e.Etag, 32)
}

func TestBaseEntity_GetID(t *testing.T) {
	t.Parallel()

	e := New()
	assert.Equal(t, e.Id, e.GetID(), "GetID should return the entity Id")
}

// helpers

func ptrInt64(i int64) *int64 {
	return &i
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func ptrString(s string) *string {
	return &s
}
