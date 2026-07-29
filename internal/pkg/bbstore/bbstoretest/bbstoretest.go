// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbstoretest provides a throwaway bbolt database for tests. Each call
// returns an isolated database that is closed automatically when the test
// finishes.
package bbstoretest

import (
	"path/filepath"
	"testing"

	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
)

// NewMemoryDB returns a private bbolt database backed by a temp file (bbolt
// has no in-memory mode); removed via t.TempDir() cleanup after the test.
func NewMemoryDB(tb testing.TB) *bbstore.DB {
	tb.Helper()
	db, err := bbstore.NewDB(filepath.Join(tb.TempDir(), "test.bolt"))
	if err != nil {
		tb.Fatalf("bbstoretest: new db: %v", err)
	}
	tb.Cleanup(func() { _ = db.Close() })
	return db
}
