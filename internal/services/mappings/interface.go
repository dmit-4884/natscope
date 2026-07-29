// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package mappings

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"
)

// Service defines the interface for subject mappings management.
// Implementations must be thread-safe for concurrent access.
type Service interface {
	// Create creates a new subject mapping.
	// Returns errs.ErrMappingSourceIDRequired if source_id is empty.
	// Returns errs.ErrMappingPatternAlreadyInUse if (pattern, source_id) exists.
	Create(ctx context.Context, in *entities.SubjectMappingCreate) (*entities.SubjectMapping, error)

	// Get retrieves a mapping by Id.
	// Returns errs.ErrMappingNotFound if not found.
	Get(ctx context.Context, id string) (*entities.SubjectMapping, error)

	// Update updates an existing mapping.
	// Returns errs.ErrMappingNotFound if not found.
	Update(ctx context.Context, in *entities.SubjectMappingUpdate) (*entities.SubjectMapping, error)

	// List returns mappings for user with pagination.
	List(ctx context.Context, in *entities.SubjectMappingsList) (*entities.List[entities.SubjectMappings], error)

	// Delete deletes a mapping by Id.
	// Returns errs.ErrMappingNotFound if not found.
	Delete(ctx context.Context, id string) error

	// BulkSave replaces the full (pattern, source_id) set atomically: existing
	// mappings absent from the given set are deleted. Returns per-outcome counts.
	BulkSave(ctx context.Context, mappings entities.SubjectMappings) (*entities.SubjectMappingBulkSaveResult, error)

	// GetAll returns all mappings (cache → storage fallback).
	// Used for server-side message decoding.
	GetAll(ctx context.Context) (entities.SubjectMappings, error)

	// Resolver returns the current immutable mapping resolver; safe to hold
	// across goroutines. Swapped atomically on every mutation.
	Resolver(ctx context.Context) *natsutil.MappingResolver
}
