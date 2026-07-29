// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package sections holds one Section adapter per domain (connections, proto
// sources, mappings, templates, settings), each secret-free by design.
//
// The item DTOs are the private wire shape of the export file (kept local so the
// on-disk format can evolve independently).
package sections

import (
	"encoding/json"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

// sectionVersion is the per-section payload version; bump on incompatible
// item-shape changes.
const sectionVersion = 1

// itemsPayload is the common `{version, items}` envelope for item-based
// sections.
type itemsPayload[T any] struct {
	Version int `json:"version"`
	Items   []T `json:"items"`
}

func newItemsPayload[T any](items []T) itemsPayload[T] {
	return itemsPayload[T]{Version: sectionVersion, Items: items}
}

// decodeItems parses the envelope; malformed or newer-than-this-build payloads
// are rejected rather than parsed into garbage.
func decodeItems[T any](raw json.RawMessage) ([]T, error) {
	var p itemsPayload[T]
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, errs.ErrWorkspaceSectionInvalid
	}
	if p.Version > sectionVersion {
		return nil, errs.ErrWorkspaceSectionInvalid
	}
	return p.Items, nil
}

// upsertPlan computes Validate dry-run counts for upsert-by-name merge vs
// clear-and-create replace; merge never deletes existing rows.
func upsertPlan(
	existing map[string]struct{},
	incoming []string,
	strategy entities.WorkspaceStrategy,
) (created, updated, deleted int32, conflicts []string) {
	for _, name := range incoming {
		if _, ok := existing[name]; ok {
			conflicts = append(conflicts, name)
		}
	}
	if strategy == entities.WorkspaceStrategyReplace {
		return int32(len(incoming)), 0, int32(len(existing)), conflicts
	}
	updated = int32(len(conflicts))
	created = int32(len(incoming)) - updated
	return created, updated, 0, conflicts
}

// createOnlyPlan is the Validate dry-run for sections that only create on
// merge (existing secrets stay untouched); replace clears and recreates.
func createOnlyPlan(
	existing map[string]struct{},
	incoming []string,
	strategy entities.WorkspaceStrategy,
) (created, deleted int32, conflicts []string) {
	for _, name := range incoming {
		if _, ok := existing[name]; ok {
			conflicts = append(conflicts, name)
		}
	}
	if strategy == entities.WorkspaceStrategyReplace {
		return int32(len(incoming)), int32(len(existing)), conflicts
	}
	// Merge: only the non-colliding names are created; collisions are skipped.
	return int32(len(incoming) - len(conflicts)), 0, conflicts
}
