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

func TestConflicts_RoundTrip(t *testing.T) {
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
			c.Kind = entities.ConflictDifferentShape
			c.Severity = entities.SeverityError
			c.Symbol = "api.v1.Order"
			c.First = entities.SchemaRef{SourceID: "src-w", Revision: "v2", File: "order.proto"}
			c.Second = entities.SchemaRef{SourceID: "src-l", Revision: "v1", File: "old/order.proto"}
			c.Reason = "shape differs"
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
	if c.Kind != entities.ConflictDifferentShape || c.Severity != entities.SeverityError || c.Symbol != "api.v1.Order" ||
		c.Reason != "shape differs" {
		t.Fatalf("scalar mismatch: %+v", c)
	}
	if c.First != (entities.SchemaRef{SourceID: "src-w", Revision: "v2", File: "order.proto"}) {
		t.Fatalf("first mismatch: %+v", c.First)
	}
	if c.Second != (entities.SchemaRef{SourceID: "src-l", Revision: "v1", File: "old/order.proto"}) {
		t.Fatalf("second mismatch: %+v", c.Second)
	}
}
