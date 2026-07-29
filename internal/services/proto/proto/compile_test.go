// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

func TestCompileResolved(t *testing.T) {
	s := newTestService(nil, nil, nil, nil, nil)

	src := &entities.ProtoSource{Name: "t", SourceType: entities.SourceTypeLocal, Enabled: true}

	t.Run("nested root compiles end to end", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			{Path: "proto/common/types.proto", Content: "syntax = \"proto3\";\npackage common;\nmessage T { string id = 1; }\n"},
			{Path: "proto/api/svc.proto", Content: "syntax = \"proto3\";\npackage api;\nimport \"common/types.proto\";\nmessage S { common.T t = 1; }\n"},
		}
		out, err := s.compile(t.Context(), src, files, nil, "")
		require.NoError(t, err)
		assert.False(t, out.HasErrors())
		assert.Len(t, out.FDS, 2)
		assert.Equal(t, []string{"proto"}, out.Roots)
		assert.Equal(t, protoutils.RootsOriginInferred, out.Origin)
	})

	t.Run("vendored third_party resolves without hardcode", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			{Path: "third_party/acme/base.proto", Content: "syntax = \"proto3\";\npackage acme;\nmessage B { int32 x = 1; }\n"},
			{Path: "api/svc.proto", Content: "syntax = \"proto3\";\npackage api;\nimport \"acme/base.proto\";\nmessage S { acme.B b = 1; }\n"},
		}
		out, err := s.compile(t.Context(), src, files, nil, "")
		require.NoError(t, err)
		assert.False(t, out.HasErrors())
		assert.Contains(t, out.Roots, "third_party")
	})

	t.Run("wkt via standard imports", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			{Path: "a.proto", Content: "syntax = \"proto3\";\nimport \"google/protobuf/timestamp.proto\";\nmessage A { google.protobuf.Timestamp at = 1; }\n"},
		}
		out, err := s.compile(t.Context(), src, files, nil, "")
		require.NoError(t, err)
		assert.False(t, out.HasErrors())
	})

	t.Run("missing import produces diagnostic with hint not raw error", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			{Path: "a.proto", Content: "syntax = \"proto3\";\nimport \"nope/missing.proto\";\nmessage A {}\n"},
		}
		out, err := s.compile(t.Context(), src, files, nil, "")
		require.NoError(t, err)
		require.True(t, out.HasErrors())
		assert.Nil(t, out.FDS)
		var found bool
		for _, d := range out.Diags {
			if d.MissingImport == "nope/missing.proto" {
				found = true
				assert.NotEmpty(t, d.Hint)
			}
		}
		assert.True(t, found)
	})

	t.Run("multiple errors all collected", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			{Path: "a.proto", Content: "syntax = \"proto3\";\nmessage A { brokenfield }\n"},
			{Path: "b.proto", Content: "syntax = \"proto3\";\nmessage B { also broken }\n"},
		}
		out, err := s.compile(t.Context(), src, files, nil, "")
		require.NoError(t, err)
		errs := 0
		for _, d := range out.Diags {
			if d.Severity == entities.DiagnosticError {
				errs++
			}
		}
		assert.GreaterOrEqual(t, errs, 2, "both files should produce a diagnostic, not just the first")
	})

	t.Run("exclude prefixes filter before resolution", func(t *testing.T) {
		srcEx := &entities.ProtoSource{Name: "t", SourceType: entities.SourceTypeLocal, Enabled: true,
			ExcludePrefixes: []string{"gen"}}
		files := []entities.ProtoFileEntry{
			{Path: "gen/dup.proto", Content: "syntax = \"proto3\";\npackage p;\nmessage D {}\n"},
			{Path: "src/dup.proto", Content: "syntax = \"proto3\";\npackage p;\nmessage D {}\n"},
		}
		out, err := s.compile(t.Context(), srcEx, files, nil, "")
		require.NoError(t, err)
		assert.False(t, out.HasErrors())
		assert.Len(t, out.FDS, 1)
	})

	t.Run("all files excluded is an error not silent success", func(t *testing.T) {
		srcEx := &entities.ProtoSource{Name: "t", SourceType: entities.SourceTypeLocal, Enabled: true,
			ExcludePrefixes: []string{"gen"}}
		files := []entities.ProtoFileEntry{
			{Path: "gen/a.proto", Content: "syntax = \"proto3\";\nmessage A {}\n"},
		}
		out, err := s.compile(t.Context(), srcEx, files, nil, "")
		require.NoError(t, err)
		assert.True(t, out.HasErrors(), "0 targets from non-empty input = error")
	})

	t.Run("buf config drives roots (tier 0)", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			{Path: "protos/x.proto", Content: "syntax = \"proto3\";\nmessage X {}\n"},
		}
		configs := []entities.ProtoFileEntry{
			{Path: "buf.yaml", Content: "version: v2\nmodules:\n  - path: protos\n"},
		}
		out, err := s.compile(t.Context(), src, files, configs, "")
		require.NoError(t, err)
		assert.Equal(t, protoutils.RootsOriginBuf, out.Origin)
		assert.Equal(t, []string{"protos"}, out.Roots)
	})

	t.Run("manual import roots override (tier manual)", func(t *testing.T) {
		srcMan := &entities.ProtoSource{Name: "t", SourceType: entities.SourceTypeLocal, Enabled: true,
			ImportRoots: []string{"weird"}}
		files := []entities.ProtoFileEntry{
			{Path: "weird/x.proto", Content: "syntax = \"proto3\";\nmessage X {}\n"},
		}
		out, err := s.compile(t.Context(), srcMan, files, nil, "")
		require.NoError(t, err)
		assert.Equal(t, protoutils.RootsOriginManual, out.Origin)
	})
}
