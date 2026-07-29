// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"path/filepath"
	"testing"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
	"github.com/dmit-4884/natscope/internal/pkg/secrets"

	sourcesbbolt "github.com/dmit-4884/natscope/internal/storages/proto/sources/bbolt"
)

// The git token is kept in the keychain vault, not in the database.
func newSourcesStorage(t *testing.T) *sourcesbbolt.Storage {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := sourcesbbolt.New(t.Context(), db, secrets.NewMemory())
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return s
}

// TestSources_LastCompileAndChildrenRoundTrip guards the last_compile_* name-map
// and the six child tables (string lists + diagnostics + roots).
func TestSources_LastCompileAndChildrenRoundTrip(t *testing.T) {
	s := newSourcesStorage(t)
	ctx := t.Context()

	in := entities.ProtoSourceNew(func(p *entities.ProtoSource) {
		p.Name = "src"
		p.SourceType = entities.SourceTypeGit
		p.Enabled = true
		p.Repository = "https://example.com/repo.git"
		p.Token = ptr.Wrap("plain-token")
		p.WatcherEnabled = false
		p.Files = []string{"a/x.proto", "a/y.proto"}
		p.IncludeDirs = []string{"inc1", "inc2"}
		p.ImportRoots = []string{"root1"}
		p.ExcludePrefixes = []string{"gen", "pb"}
		p.LastCompile = &entities.ProtoCompileResult{
			At: 1700000000, Ok: true, Error: ptr.Wrap("warn"),
			MessageCount: 1434, FileCount: 283, RootsOrigin: "buf",
			Diagnostics: []entities.CompileDiagnostic{
				{Severity: entities.DiagnosticWarning, File: "a/x.proto", Line: 12, Column: 3, Message: "m", MissingImport: "google/x.proto", Hint: "add inc"},
			},
			Roots: []string{"r1", "r2", "r3"},
		}
	})
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if !got.Enabled || got.WatcherEnabled || got.SourceType != entities.SourceTypeGit {
		t.Fatalf("scalar mismatch: %+v", got)
	}
	if got.Token == nil || *got.Token != "plain-token" {
		t.Fatalf("token round-trip: %v", got.Token)
	}
	if len(got.Files) != 2 || got.Files[1] != "a/y.proto" || len(got.IncludeDirs) != 2 ||
		len(got.ImportRoots) != 1 || len(got.ExcludePrefixes) != 2 || got.ExcludePrefixes[1] != "pb" {
		t.Fatalf("string-list children mismatch: %+v", got)
	}
	lc := got.LastCompile
	if lc == nil || lc.At != 1700000000 || !lc.Ok || *lc.Error != "warn" ||
		lc.MessageCount != 1434 || lc.FileCount != 283 || lc.RootsOrigin != "buf" {
		t.Fatalf("last_compile scalars mismatch: %+v", lc)
	}
	if len(lc.Diagnostics) != 1 {
		t.Fatalf("diagnostics len: %d", len(lc.Diagnostics))
	}
	d := lc.Diagnostics[0]
	if d.Severity != entities.DiagnosticWarning || d.File != "a/x.proto" || d.Line != 12 || d.Column != 3 ||
		d.Message != "m" || d.MissingImport != "google/x.proto" || d.Hint != "add inc" {
		t.Fatalf("diagnostic mismatch: %+v", d)
	}
	if len(lc.Roots) != 3 || lc.Roots[2] != "r3" {
		t.Fatalf("roots mismatch: %v", lc.Roots)
	}
}
