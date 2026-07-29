// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/core/collections/slices"

	"github.com/dmit-4884/natscope/internal/entities"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, content := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	return root
}

func entryPaths(in []entities.ProtoFileEntry) []string {
	return slices.To(in, func(f entities.ProtoFileEntry) string { return f.Path })
}

func TestWalkProtoTree(t *testing.T) {
	t.Run("collects nested protos with slash paths", func(t *testing.T) {
		root := writeTree(t, map[string]string{
			"a.proto":           "syntax",
			"x/b.proto":         "syntax",
			"x/y/z/deep.proto":  "syntax",
			"x/readme.md":       "not a proto",
			"x/y/data.protobuf": "wrong ext",
		})
		res, err := WalkProtoTree(root, WalkOptions{})
		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]string{"a.proto", "x/b.proto", "x/y/z/deep.proto"},
			entryPaths(res.Files))
		for _, f := range res.Files {
			assert.False(t, strings.Contains(f.Path, "\\"), "path must be slash-normalized: %q", f.Path)
		}
	})

	t.Run("skips .git and node_modules at any depth", func(t *testing.T) {
		root := writeTree(t, map[string]string{
			"ok.proto":                     "syntax",
			".git/objects/x.proto":         "git",
			"sub/node_modules/dep/x.proto": "npm",
			"sub/ok2.proto":                "syntax",
		})
		res, err := WalkProtoTree(root, WalkOptions{})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"ok.proto", "sub/ok2.proto"}, entryPaths(res.Files))
	})

	t.Run("exclude prefixes drop whole subtrees with report", func(t *testing.T) {
		root := writeTree(t, map[string]string{
			"keep.proto":      "syntax",
			"gen/a.proto":     "generated",
			"gen/sub/b.proto": "generated",
		})
		res, err := WalkProtoTree(root, WalkOptions{ExcludePrefixes: []string{"gen"}})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"keep.proto"}, entryPaths(res.Files))
		require.Len(t, res.Skipped, 1)
		assert.Equal(t, "gen", res.Skipped[0].Path)
		assert.Contains(t, res.Skipped[0].Reason, "excluded by prefix")
	})

	t.Run("skips symlinked proto files without reading through", func(t *testing.T) {
		root := writeTree(t, map[string]string{"real.proto": "syntax"})
		secret := filepath.Join(t.TempDir(), "secret.txt")
		require.NoError(t, os.WriteFile(secret, []byte("TOP-SECRET"), 0o600))
		require.NoError(t, os.Symlink(secret, filepath.Join(root, "leak.proto")))

		res, err := WalkProtoTree(root, WalkOptions{})
		require.NoError(t, err)

		assert.ElementsMatch(t, []string{"real.proto"}, entryPaths(res.Files))
		for _, f := range res.Files {
			assert.NotContains(t, f.Content, "TOP-SECRET", "symlink must not be read through")
		}

		var sawSkip bool
		for _, s := range res.Skipped {
			if s.Path == "leak.proto" {
				sawSkip = true
				assert.Contains(t, s.Reason, "symlink")
			}
		}
		assert.True(t, sawSkip, "symlinked .proto should be recorded as skipped")
	})

	t.Run("oversize proto skipped with report not error", func(t *testing.T) {
		root := writeTree(t, map[string]string{
			"small.proto": "syntax",
			"big.proto":   strings.Repeat("x", 1024*1024+1),
		})
		res, err := WalkProtoTree(root, WalkOptions{})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"small.proto"}, entryPaths(res.Files))
		require.Len(t, res.Skipped, 1)
		assert.Equal(t, "big.proto", res.Skipped[0].Path)
		assert.Contains(t, res.Skipped[0].Reason, "exceeds")
	})

	t.Run("collects buf configs when asked", func(t *testing.T) {
		root := writeTree(t, map[string]string{
			"buf.yaml":     "version: v2",
			"buf.lock":     "version: v2",
			"sub/buf.yaml": "version: v1",
			"a.proto":      "syntax",
			"buf.gen.yaml": "not collected",
		})
		res, err := WalkProtoTree(root, WalkOptions{CollectConfigs: true})
		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]string{"buf.yaml", "buf.lock", "sub/buf.yaml"},
			entryPaths(res.Configs))
	})

	t.Run("configs not collected by default", func(t *testing.T) {
		root := writeTree(t, map[string]string{"buf.yaml": "version: v2", "a.proto": "s"})
		res, err := WalkProtoTree(root, WalkOptions{})
		require.NoError(t, err)
		assert.Empty(t, res.Configs)
	})

	t.Run("missing root is an error", func(t *testing.T) {
		_, err := WalkProtoTree(filepath.Join(t.TempDir(), "nope"), WalkOptions{})
		assert.Error(t, err)
	})

	t.Run("file count and sizes populated", func(t *testing.T) {
		root := writeTree(t, map[string]string{"a.proto": "12345"})
		res, err := WalkProtoTree(root, WalkOptions{})
		require.NoError(t, err)
		require.Len(t, res.Files, 1)
		assert.EqualValues(t, 5, res.Files[0].Size)
		assert.Equal(t, "12345", res.Files[0].Content)
	})
}

func TestWalkProtoTree_TotalFileLimit(t *testing.T) {
	files := map[string]string{}
	for i := range 5 {
		files[fmt.Sprintf("f%d.proto", i)] = "syntax"
	}
	root := writeTree(t, files)

	_, err := WalkProtoTree(root, WalkOptions{MaxTotalFiles: 3})

	require.ErrorIs(t, err, ErrWalkLimitExceeded)
}

func TestWalkProtoTree_TotalByteLimit(t *testing.T) {
	body := strings.Repeat("x", 1000)
	files := map[string]string{}
	for i := range 5 {
		files[fmt.Sprintf("f%d.proto", i)] = body
	}
	root := writeTree(t, files)

	_, err := WalkProtoTree(root, WalkOptions{MaxTotalBytes: 2000})

	require.ErrorIs(t, err, ErrWalkLimitExceeded)
}

func TestWalkProtoTree_WithinLimits(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.proto": "syntax",
		"b.proto": "syntax",
		"c.proto": "syntax",
	})

	res, err := WalkProtoTree(root, WalkOptions{MaxTotalFiles: 3, MaxTotalBytes: 1 << 20})

	require.NoError(t, err)
	assert.Len(t, res.Files, 3)
}
