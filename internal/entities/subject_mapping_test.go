// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubjectMappingNew(t *testing.T) {
	t.Parallel()

	t.Run("WithoutInit", func(t *testing.T) {
		t.Parallel()
		m := SubjectMappingNew()
		require.NotNil(t, m)
		assert.NotEmpty(t, m.Id)
		assert.Positive(t, m.CreatedAt)
		assert.Empty(t, m.Pattern)
		assert.Empty(t, m.MessageType)
	})

	t.Run("WithInit", func(t *testing.T) {
		t.Parallel()
		m := SubjectMappingNew(func(m *SubjectMapping) {
			m.Pattern = "orders.*"
			m.MessageType = "api.v1.Order"
		})
		assert.Equal(t, "orders.*", m.Pattern)
		assert.Equal(t, "api.v1.Order", m.MessageType)
	})

	t.Run("WithNilInit", func(t *testing.T) {
		t.Parallel()
		m := SubjectMappingNew(nil)
		require.NotNil(t, m)
	})
}

func TestSubjectMapping_ApplyUpdate(t *testing.T) {
	t.Parallel()

	t.Run("UpdatesFields", func(t *testing.T) {
		t.Parallel()
		m := SubjectMappingNew(func(m *SubjectMapping) {
			m.Pattern = "old.*"
			m.MessageType = "OldType"
		})
		// Seed a known-old UpdatedAt instead of sleeping (deterministic, race-free).
		m.UpdatedAt = time.UnixMilli(1).UTC()
		m.ApplyUpdate(&SubjectMappingUpdate{
			Pattern:     new("new.*"),
			MessageType: new("NewType"),
		})

		assert.Equal(t, "new.*", m.Pattern)
		assert.Equal(t, "NewType", m.MessageType)
		assert.True(t, m.UpdatedAt.After(time.UnixMilli(1).UTC()))
	})

	t.Run("FramingReplacesWholesale", func(t *testing.T) {
		t.Parallel()
		m := SubjectMappingNew(func(m *SubjectMapping) {
			m.Framing = Framing{Kind: FramingConfluent, SchemaID: 7, Prefix: []byte("p"), Suffix: []byte("s")}
		})
		m.ApplyUpdate(&SubjectMappingUpdate{Framing: &Framing{Kind: FramingGRPC}})

		assert.Equal(t, Framing{Kind: FramingGRPC}, m.Framing)
	})

	t.Run("NilFramingKeepsExisting", func(t *testing.T) {
		t.Parallel()
		want := Framing{Kind: FramingConfluent, SchemaID: 7}
		m := SubjectMappingNew(func(m *SubjectMapping) { m.Framing = want })
		m.ApplyUpdate(&SubjectMappingUpdate{Pattern: new("x")})

		assert.Equal(t, want, m.Framing)
	})

	t.Run("NilReceiver", func(t *testing.T) {
		t.Parallel()
		var m *SubjectMapping
		m.ApplyUpdate(&SubjectMappingUpdate{Pattern: new("x")})
	})

	t.Run("NilRequest", func(t *testing.T) {
		t.Parallel()
		m := SubjectMappingNew()
		oldUpdatedAt := m.UpdatedAt
		m.ApplyUpdate(nil)
		assert.Equal(t, oldUpdatedAt, m.UpdatedAt)
	})
}
