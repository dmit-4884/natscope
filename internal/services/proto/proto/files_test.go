// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

// writeProto drops a .proto file in dir and returns its path.
func writeProto(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

// --- compileFiles ---

func TestCompileFiles_Success(t *testing.T) {
	dir := t.TempDir()
	p := writeProto(t, dir, "ok.proto", `
syntax = "proto3";
package test;
message Hello { string name = 1; }
`)

	entries, diags := protoutils.ReadFilesFromPaths([]string{p})
	require.Empty(t, diags)
	require.Len(t, entries, 1)

	fds, compileDiags, err := compileFiles(t.Context(), entries, nil)
	require.NoError(t, err)
	assert.Empty(t, compileDiags)
	require.Len(t, fds, 1)
	assert.Equal(t, "ok.proto", fds[0].Path())
}

func TestCompileFiles_SyntaxError(t *testing.T) {
	dir := t.TempDir()
	p := writeProto(t, dir, "bad.proto", `
syntax = "proto3";
package test;
message Broken {
  string name 1;  // missing equals
}
`)

	entries, diags := protoutils.ReadFilesFromPaths([]string{p})
	require.Empty(t, diags)
	fds, compileDiags, err := compileFiles(t.Context(), entries, nil)
	require.NoError(t, err)
	assert.Empty(t, fds)
	require.NotEmpty(t, compileDiags, "syntax error must surface as diagnostic")
	first := compileDiags[0]
	assert.Equal(t, entities.DiagnosticError, first.Severity)
	assert.Equal(t, "bad.proto", first.File)
	assert.Greater(t, first.Line, 0, "line position must be set")
}

func TestCompileFiles_StrictWKT_NoImplicit(t *testing.T) {
	// Files type is strict: importing google/protobuf/timestamp.proto must NOT
	// auto-resolve; the user must provide it explicitly.
	dir := t.TempDir()
	p := writeProto(t, dir, "with_ts.proto", `
syntax = "proto3";
package test;
import "google/protobuf/timestamp.proto";
message WithTs { google.protobuf.Timestamp ts = 1; }
`)
	entries, _ := protoutils.ReadFilesFromPaths([]string{p})
	fds, diags, err := compileFiles(t.Context(), entries, nil)
	require.NoError(t, err)
	assert.Empty(t, fds, "must not compile without explicit timestamp.proto")

	var wkt *entities.CompileDiagnostic
	for i := range diags {
		if diags[i].MissingImport == "google/protobuf/timestamp.proto" {
			wkt = &diags[i]
			break
		}
	}
	require.NotNil(t, wkt, "expected missing-import diagnostic for google/protobuf/timestamp.proto, got %#v", diags)
	assert.Contains(t, wkt.Hint, "Files type is strict",
		"hint must explain that WKT are not auto-included in Files mode")
}

func TestCompileFiles_MissingImport_Hint(t *testing.T) {
	// Non-WKT missing import should produce a generic "add to Include Dirs" hint.
	dir := t.TempDir()
	p := writeProto(t, dir, "needs_fake.proto", `
syntax = "proto3";
package test;
import "totally/fake/dep.proto";
message NeedsFake { string x = 1; }
`)

	entries, preDiags := protoutils.ReadFilesFromPaths([]string{p})
	require.Empty(t, preDiags)

	fds, compileDiags, err := compileFiles(t.Context(), entries, nil)
	require.NoError(t, err)
	assert.Empty(t, fds)

	var found *entities.CompileDiagnostic
	for i := range compileDiags {
		if compileDiags[i].MissingImport == "totally/fake/dep.proto" {
			found = &compileDiags[i]
			break
		}
	}
	require.NotNil(t, found, "expected MissingImport diagnostic, got %#v", compileDiags)
	assert.Contains(t, strings.ToLower(found.Hint), "include directories",
		"non-WKT missing-import hint must mention Include Directories")
	assert.NotContains(t, found.Hint, "strict",
		"non-WKT hint should not mention strict mode (WKT-specific message)")
}

func TestCompileFiles_IncludeDirsResolve(t *testing.T) {
	// leaf.proto imports "shared/common.proto"; adding the deps dir as an
	// include dir must satisfy the import.
	tmp := t.TempDir()
	leaf := writeProto(t, filepath.Join(tmp, "main"), "leaf.proto", `
syntax = "proto3";
package leaf;
import "shared/common.proto";
message Leaf { shared.Common c = 1; }
`)
	_ = writeProto(t, filepath.Join(tmp, "deps", "shared"), "common.proto", `
syntax = "proto3";
package shared;
message Common { string s = 1; }
`)

	entries, diags := protoutils.ReadFilesFromPaths([]string{leaf})
	require.Empty(t, diags)

	// Without include dirs → fails.
	_, withoutDiags, err := compileFiles(t.Context(), entries, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, withoutDiags, "should fail without include dirs")

	// With the right include dir → succeeds.
	fds, withDiags, err := compileFiles(t.Context(), entries, []string{filepath.Join(tmp, "deps")})
	require.NoError(t, err)
	assert.Empty(t, withDiags)
	assert.NotEmpty(t, fds, "should compile with include dirs satisfying the import")
}

// --- ValidateFiles (inline path; no storage needed) ---

func TestService_ValidateFiles_Inline_Success(t *testing.T) {
	dir := t.TempDir()
	p := writeProto(t, dir, "inline.proto", `
syntax = "proto3";
package inline;
message M { int32 x = 1; }
`)

	s := &Service{}
	result, diags, err := s.ValidateFiles(t.Context(), nil, []string{p}, nil)
	require.NoError(t, err)
	assert.Empty(t, diags)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.FileDescriptors)
	assert.Equal(t, 1, result.MessageTypes)
}

func TestService_ValidateFiles_Inline_PreCompileError(t *testing.T) {
	s := &Service{}
	result, diags, err := s.ValidateFiles(t.Context(), nil, []string{"/nonexistent/foo.proto"}, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
	require.NotEmpty(t, diags)
	assert.Equal(t, entities.DiagnosticError, diags[0].Severity)
}

// --- Integration with the user's real proto repo. Skipped if not present. ---

func TestService_ValidateFiles_RealRepo(t *testing.T) {
	repo := "/Users/dmit4884/GolandProjects/proto"
	if _, err := os.Stat(repo); err != nil {
		t.Skipf("proto repo not present at %s", repo)
	}
	candidate := filepath.Join(repo, "bus", "organizations", "organizations_bus_orgs.proto")
	if _, err := os.Stat(candidate); err != nil {
		t.Skipf("specific candidate file missing: %v", err)
	}

	s := &Service{}
	// Either outcome (with or without include dirs) is acceptable; this just
	// confirms the diagnostics path returns useful information.
	_, diagsNoDeps, err := s.ValidateFiles(t.Context(), nil, []string{candidate}, nil)
	require.NoError(t, err)
	t.Logf("without include dirs: %d diagnostics", len(diagsNoDeps))

	includeDirs := []string{repo, filepath.Join(repo, "third_party")}
	res, diagsWithDeps, err := s.ValidateFiles(t.Context(), nil, []string{candidate}, includeDirs)
	require.NoError(t, err)
	t.Logf("with include dirs: %d diagnostics, result=%v", len(diagsWithDeps), res)
}
