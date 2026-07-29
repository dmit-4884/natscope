// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

// writeProto drops a .proto file in dir and returns its path.
func writeProto(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

// --- ReadFilesFromPaths ---

func TestReadFilesFromPaths_RejectsRelative(t *testing.T) {
	entries, diags := ReadFilesFromPaths([]string{"relative/path.proto"})
	assert.Empty(t, entries)
	require.Len(t, diags, 1)
	assert.Equal(t, entities.DiagnosticError, diags[0].Severity)
	assert.Contains(t, diags[0].Message, "absolute")
}

func TestReadFilesFromPaths_RejectsNonProto(t *testing.T) {
	dir := t.TempDir()
	notProto := filepath.Join(dir, "readme.md")
	require.NoError(t, os.WriteFile(notProto, []byte("# x"), 0o644))

	entries, diags := ReadFilesFromPaths([]string{notProto})
	assert.Empty(t, entries)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, ".proto")
}

func TestReadFilesFromPaths_RejectsMissing(t *testing.T) {
	entries, diags := ReadFilesFromPaths([]string{"/nonexistent/foo.proto"})
	assert.Empty(t, entries)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "not accessible")
}

func TestReadFilesFromPaths_DuplicateBasename(t *testing.T) {
	dir := t.TempDir()
	a := writeProto(t, filepath.Join(dir, "x"), "foo.proto", `syntax = "proto3"; package x;`)
	b := writeProto(t, filepath.Join(dir, "y"), "foo.proto", `syntax = "proto3"; package y;`)

	entries, diags := ReadFilesFromPaths([]string{a, b})
	require.Len(t, entries, 1, "first file accepted, duplicate basename rejected")
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "duplicate")
	assert.Contains(t, diags[0].Hint, "Include Directories")
}

// --- ValidateIncludeDirs ---

func TestValidateIncludeDirs_RejectsRelative(t *testing.T) {
	diags := ValidateIncludeDirs([]string{"relative/dir"})
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "absolute")
}

func TestValidateIncludeDirs_RejectsMissing(t *testing.T) {
	diags := ValidateIncludeDirs([]string{"/totally/missing/dir"})
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "not accessible")
}

func TestValidateIncludeDirs_RejectsFile(t *testing.T) {
	dir := t.TempDir()
	notDir := filepath.Join(dir, "x.proto")
	require.NoError(t, os.WriteFile(notDir, []byte("x"), 0o644))

	diags := ValidateIncludeDirs([]string{notDir})
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "not a directory")
}

func TestValidateIncludeDirs_AcceptsDirectory(t *testing.T) {
	diags := ValidateIncludeDirs([]string{t.TempDir()})
	assert.Empty(t, diags)
}

func TestValidateIncludeDirs_SkipsBlank(t *testing.T) {
	assert.Empty(t, ValidateIncludeDirs([]string{"", "   "}))
}
