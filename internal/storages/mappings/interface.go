// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package mappings is the storage contract for subject mappings.
package mappings

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for subject mappings storage.
type Storage interface {
	// Save creates a new subject mapping.
	// Returns errs.ErrMappingPatternAlreadyInUse if pattern is already mapped.
	Save(ctx context.Context, in *entities.SubjectMapping) error

	// Get retrieves a mapping by Id.
	// Returns errs.ErrMappingNotFound if not found.
	Get(ctx context.Context, id string) (*entities.SubjectMapping, error)

	// List returns mappings with pagination.
	List(ctx context.Context, in *entities.SubjectMappingsList) (*entities.List[entities.SubjectMappings], error)

	// ListAll returns all mappings without pagination.
	ListAll(ctx context.Context) (entities.SubjectMappings, error)

	// Delete deletes a mapping.
	// Returns errs.ErrMappingNotFound if not found.
	Delete(ctx context.Context, id string) error

	// Exists checks if a mapping exists.
	Exists(ctx context.Context, id string) (bool, error)

	// BulkSave replaces the full (pattern, source_id) set atomically: items
	// present are created or updated in place (preserving id/created_at for
	// existing pairs), and existing items absent from the given set are
	// deleted. Returns per-outcome counts.
	BulkSave(ctx context.Context, mappings entities.SubjectMappings) (*entities.SubjectMappingBulkSaveResult, error)
}
