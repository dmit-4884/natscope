// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtoFileSetNew(t *testing.T) {
	t.Parallel()

	t.Run("WithoutInit", func(t *testing.T) {
		t.Parallel()
		v := ProtoFileSetNew()
		require.NotNil(t, v)
		assert.NotEmpty(t, v.Id)
		assert.Empty(t, v.SourceID)
		assert.Empty(t, v.Revision)
		assert.Nil(t, v.Files)
	})

	t.Run("WithInit", func(t *testing.T) {
		t.Parallel()
		v := ProtoFileSetNew(func(v *ProtoFileSet) {
			v.SourceID = "src-1"
			v.Revision = "abc"
			v.Files = []ProtoFileEntry{
				{Path: "proto/order.proto", Content: "syntax = \"proto3\";", Size: 19},
			}
			v.FetchedAt = time.UnixMilli(12345).UTC()
		})
		assert.Equal(t, "src-1", v.SourceID)
		assert.Equal(t, "abc", v.Revision)
		require.Len(t, v.Files, 1)
		assert.Equal(t, "proto/order.proto", v.Files[0].Path)
	})

	t.Run("WithNilInit", func(t *testing.T) {
		t.Parallel()
		require.NotNil(t, ProtoFileSetNew(nil))
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
			d.Revision = "v1.0.0"
			d.DescriptorSet = []byte{0x0A, 0x0B}
			d.MessageTypes = []string{"api.v1.Order", "api.v1.User"}
			d.CompiledAt = 99999
		})
		assert.Equal(t, "v1.0.0", d.Revision)
		assert.Len(t, d.MessageTypes, 2)
		assert.Equal(t, int64(99999), d.CompiledAt)
	})

	t.Run("WithNilInit", func(t *testing.T) {
		t.Parallel()
		d := ProtoDescriptorNew(nil)
		require.NotNil(t, d)
	})
}
