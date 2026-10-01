// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbstore_test

import (
	"testing"

	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
)

func TestDropBuckets(t *testing.T) {
	db := newDB(t)
	spec := func(bucket string) bbstore.Spec { return bbstore.Spec{Bucket: bucket, NotFound: errWidgetNotFound} }

	old, err := bbstore.Open[widget, *widget](t.Context(), db, spec("old"))
	if err != nil {
		t.Fatalf("open old: %v", err)
	}
	if err := old.Save(t.Context(), &widget{Base: bbstore.Base{ID: "w1"}, Name: "stale"}); err != nil {
		t.Fatalf("save old: %v", err)
	}
	kept, err := bbstore.Open[widget, *widget](t.Context(), db, spec("kept"))
	if err != nil {
		t.Fatalf("open kept: %v", err)
	}
	if err := kept.Save(t.Context(), &widget{Base: bbstore.Base{ID: "w2"}, Name: "keep"}); err != nil {
		t.Fatalf("save kept: %v", err)
	}

	if err := db.DropBuckets(t.Context(), "old", "never-existed"); err != nil {
		t.Fatalf("drop: %v", err)
	}

	reopened, err := bbstore.Open[widget, *widget](t.Context(), db, spec("old"))
	if err != nil {
		t.Fatalf("reopen old: %v", err)
	}
	if all, err := reopened.ListAll(t.Context()); err != nil || len(all) != 0 {
		t.Fatalf("dropped bucket must come back empty: %v %d", err, len(all))
	}
	if all, err := kept.ListAll(t.Context()); err != nil || len(all) != 1 {
		t.Fatalf("other buckets survive: %v %d", err, len(all))
	}
}
