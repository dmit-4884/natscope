// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package selections is the storage contract for proto version selections.
package selections

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for proto selections storage.
type Storage interface {
	// Save creates or updates a proto selection (upsert).
	Save(ctx context.Context, in *entities.ProtoSelection) error

	// Get retrieves a selection by Id.
	// Returns errs.ErrProtoSelectionNotFound if not found.
	Get(ctx context.Context, id string) (*entities.ProtoSelection, error)

	// GetBySource retrieves a selection by source Id.
	// Returns errs.ErrProtoSelectionNotFound if not found.
	GetBySource(ctx context.Context, sourceID string) (*entities.ProtoSelection, error)

	// List returns selections with pagination.
	List(ctx context.Context, in *entities.ProtoSelectionsList) (*entities.List[entities.ProtoSelections], error)

	// GetAll returns all selections.
	GetAll(ctx context.Context) (entities.ProtoSelections, error)

	// Delete deletes a selection.
	// Returns errs.ErrProtoSelectionNotFound if not found.
	Delete(ctx context.Context, id string) error

	// DeleteBySource deletes all selections for a source.
	DeleteBySource(ctx context.Context, sourceID string) (int64, error)
}
