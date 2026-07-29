// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtoSelectionNew(t *testing.T) {
	t.Parallel()

	t.Run("WithoutInit", func(t *testing.T) {
		t.Parallel()
		s := ProtoSelectionNew()
		require.NotNil(t, s)
		assert.NotEmpty(t, s.Id)
		assert.Positive(t, s.CreatedAt)
		assert.Empty(t, s.SourceID)
		assert.Empty(t, s.Tag)
	})

	t.Run("WithInit", func(t *testing.T) {
		t.Parallel()
		s := ProtoSelectionNew(func(s *ProtoSelection) {
			s.SourceID = "src-1"
			s.Tag = "v1.0.0"
		})
		assert.Equal(t, "src-1", s.SourceID)
		assert.Equal(t, "v1.0.0", s.Tag)
	})

	t.Run("WithNilInit", func(t *testing.T) {
		t.Parallel()
		s := ProtoSelectionNew(nil)
		require.NotNil(t, s)
	})
}

func TestProtoVersionNew(t *testing.T) {
	t.Parallel()

	t.Run("WithoutInit", func(t *testing.T) {
		t.Parallel()
		v := ProtoVersionNew()
		require.NotNil(t, v)
		assert.NotEmpty(t, v.Id)
		assert.False(t, v.CreatedAt.IsZero())
		assert.Empty(t, v.SourceID)
		assert.Empty(t, v.Tag)
		assert.Nil(t, v.Files)
		assert.Nil(t, v.FetchedBy)
	})

	t.Run("WithInit", func(t *testing.T) {
		t.Parallel()
		v := ProtoVersionNew(func(v *ProtoVersion) {
			v.SourceID = "src-1"
			v.Tag = "v2.0.0"
			v.Files = []ProtoFileEntry{
				{Path: "proto/order.proto", Content: "syntax = \"proto3\";", Size: 19},
			}
			v.FetchedAt = time.UnixMilli(12345).UTC()
			v.FetchedBy = ptrString("user@test.com")
		})
		assert.Equal(t, "src-1", v.SourceID)
		assert.Equal(t, "v2.0.0", v.Tag)
		require.Len(t, v.Files, 1)
		assert.Equal(t, "proto/order.proto", v.Files[0].Path)
		assert.Equal(t, int64(19), v.Files[0].Size)
	})

	t.Run("WithNilInit", func(t *testing.T) {
		t.Parallel()
		v := ProtoVersionNew(nil)
		require.NotNil(t, v)
	})
}

func TestProtoDescriptorNew(t *testing.T) {
	t.Parallel()

	t.Run("WithoutInit", func(t *testing.T) {
		t.Parallel()
		d := ProtoDescriptorNew()
		require.NotNil(t, d)
		assert.NotEmpty(t, d.Id)
		assert.Positive(t, d.CreatedAt)
		assert.Empty(t, d.SourceID)
		assert.Nil(t, d.DescriptorSet)
		assert.Nil(t, d.MessageTypes)
	})

	t.Run("WithInit", func(t *testing.T) {
		t.Parallel()
		d := ProtoDescriptorNew(func(d *ProtoDescriptor) {
			d.SourceID = "src-1"
			d.Tag = "v1.0.0"
			d.DescriptorSet = []byte{0x0A, 0x0B}
			d.MessageTypes = []string{"api.v1.Order", "api.v1.User"}
			d.CompiledAt = 99999
		})
		assert.Equal(t, "v1.0.0", d.Tag)
		assert.Len(t, d.MessageTypes, 2)
		assert.Equal(t, int64(99999), d.CompiledAt)
	})

	t.Run("WithNilInit", func(t *testing.T) {
		t.Parallel()
		d := ProtoDescriptorNew(nil)
		require.NotNil(t, d)
	})
}
