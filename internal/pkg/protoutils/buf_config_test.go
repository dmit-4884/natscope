// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

func cfg(p, content string) entities.ProtoFileEntry {
	return entities.ProtoFileEntry{Path: p, Content: content, Size: int64(len(content))}
}

func TestDetectBufLayout(t *testing.T) {
	t.Run("no configs -> not found", func(t *testing.T) {
		_, ok := DetectBufLayout(nil)
		assert.False(t, ok)
		_, ok = DetectBufLayout([]entities.ProtoFileEntry{cfg("buf.gen.yaml", "x")})
		assert.False(t, ok)
	})

	t.Run("buf.yaml v2 with modules", func(t *testing.T) {
		layout, ok := DetectBufLayout([]entities.ProtoFileEntry{cfg("buf.yaml",
			"version: v2\nmodules:\n  - path: proto\n  - path: vendor/protos\nlint:\n  use:\n    - STANDARD\n")})
		require.True(t, ok)
		assert.Equal(t, []string{"proto", "vendor/protos"}, layout.Roots)
	})

	t.Run("buf.yaml v2 without modules -> module at its dir", func(t *testing.T) {
		layout, ok := DetectBufLayout([]entities.ProtoFileEntry{cfg("buf.yaml", "version: v2\nlint:\n  use:\n    - STANDARD\n")})
		require.True(t, ok)
		assert.Empty(t, layout.Roots) // root "" — implicit source root
	})

	t.Run("buf.yaml v2 in subdir resolves module paths relative to it", func(t *testing.T) {
		layout, ok := DetectBufLayout([]entities.ProtoFileEntry{cfg("api/buf.yaml",
			"version: v2\nmodules:\n  - path: protos\n")})
		require.True(t, ok)
		assert.Equal(t, []string{"api/protos"}, layout.Roots)
	})

	t.Run("buf.yaml v1 -> root is its dir", func(t *testing.T) {
		layout, ok := DetectBufLayout([]entities.ProtoFileEntry{
			cfg("proto/buf.yaml", "version: v1\nname: buf.build/acme/x\n"),
		})
		require.True(t, ok)
		assert.Equal(t, []string{"proto"}, layout.Roots)
	})

	t.Run("buf.work.yaml wins over buf.yaml files", func(t *testing.T) {
		layout, ok := DetectBufLayout([]entities.ProtoFileEntry{
			cfg("buf.work.yaml", "version: v1\ndirectories:\n  - apis\n  - vendor\n"),
			cfg("apis/buf.yaml", "version: v1\n"),
			cfg("vendor/buf.yaml", "version: v1\n"),
		})
		require.True(t, ok)
		assert.Equal(t, []string{"apis", "vendor"}, layout.Roots)
	})

	t.Run("shallowest buf.work.yaml is authoritative", func(t *testing.T) {
		layout, ok := DetectBufLayout([]entities.ProtoFileEntry{
			cfg("deep/nested/buf.work.yaml", "version: v1\ndirectories:\n  - x\n"),
			cfg("buf.work.yaml", "version: v1\ndirectories:\n  - top\n"),
		})
		require.True(t, ok)
		assert.Equal(t, []string{"top"}, layout.Roots)
	})

	t.Run("multiple v1 buf.yaml union", func(t *testing.T) {
		layout, ok := DetectBufLayout([]entities.ProtoFileEntry{
			cfg("a/buf.yaml", "version: v1\n"),
			cfg("b/buf.yaml", "version: v1\n"),
		})
		require.True(t, ok)
		assert.ElementsMatch(t, []string{"a", "b"}, layout.Roots)
	})

	t.Run("locks collected", func(t *testing.T) {
		layout, ok := DetectBufLayout([]entities.ProtoFileEntry{
			cfg("buf.yaml", "version: v2\n"),
			cfg("buf.lock", "version: v2\ndeps: []\n"),
		})
		require.True(t, ok)
		require.Len(t, layout.Locks, 1)
		assert.Contains(t, string(layout.Locks[0]), "deps")
	})

	t.Run("malformed yaml -> found with empty roots", func(t *testing.T) {
		layout, ok := DetectBufLayout([]entities.ProtoFileEntry{cfg("buf.yaml", ":::garbage:::")})
		require.True(t, ok)
		assert.Empty(t, layout.Roots)
	})

	t.Run("quoted and commented values", func(t *testing.T) {
		layout, ok := DetectBufLayout([]entities.ProtoFileEntry{cfg("buf.work.yaml",
			"version: v1\ndirectories:\n  - \"apis\" # main\n")})
		require.True(t, ok)
		assert.Equal(t, []string{"apis"}, layout.Roots)
	})
}
