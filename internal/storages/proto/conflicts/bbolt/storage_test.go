// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"path/filepath"
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	conflictsbbolt "github.com/dmit-4884/natscope/internal/storages/proto/conflicts/bbolt"
)

// TestConflicts_WinnerLoserRoundTrip guards the winner/loser nested refs.
func TestConflicts_WinnerLoserRoundTrip(t *testing.T) {
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := conflictsbbolt.New(t.Context(), db)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	ctx := t.Context()

	in := entities.SchemaConflicts{
		entities.SchemaConflictNew(func(c *entities.SchemaConflict) {
			c.Kind = entities.SameSymbolDifferentShape
			c.Severity = entities.SeverityError
			c.Symbol = "api.v1.Order"
			c.Winner = entities.SchemaRef{SourceID: "src-w", Tag: "v2", File: "order.proto"}
			c.Loser = entities.SchemaRef{SourceID: "src-l", Tag: "v1", File: "old/order.proto"}
			c.Reason = "shape differs"
			c.Policy = "first-wins"
		}),
	}
	if err := s.ReplaceAll(ctx, in); err != nil {
		t.Fatalf("replace all: %v", err)
	}
	got, err := s.GetAll(ctx)
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 conflict, got %d", len(got))
	}
	c := got[0]
	if c.Kind != entities.SameSymbolDifferentShape || c.Severity != entities.SeverityError || c.Symbol != "api.v1.Order" ||
		c.Reason != "shape differs" || c.Policy != "first-wins" {
		t.Fatalf("scalar mismatch: %+v", c)
	}
	if c.Winner != (entities.SchemaRef{SourceID: "src-w", Tag: "v2", File: "order.proto"}) {
		t.Fatalf("winner mismatch: %+v", c.Winner)
	}
	if c.Loser != (entities.SchemaRef{SourceID: "src-l", Tag: "v1", File: "old/order.proto"}) {
		t.Fatalf("loser mismatch: %+v", c.Loser)
	}
}
