// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package sources is the storage contract for proto source definitions.
package sources

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for proto sources storage.
type Storage interface {
	// Save creates a new proto source.
	// Returns errs.ErrProtoSourceNameAlreadyInUse if name is already taken.
	Save(ctx context.Context, in *entities.ProtoSource) error

	// Get retrieves a source by Id.
	// Returns errs.ErrProtoSourceNotFound if not found.
	Get(ctx context.Context, id string, includeDeleted ...bool) (*entities.ProtoSource, error)

	// GetByName retrieves a source by name.
	// Returns errs.ErrProtoSourceNotFound if not found.
	GetByName(ctx context.Context, name string, includeDeleted ...bool) (*entities.ProtoSource, error)

	// List returns sources with pagination.
	List(ctx context.Context, in *entities.ProtoSourcesList) (*entities.List[entities.ProtoSources], error)

	// Update updates a source.
	// Returns errs.ErrProtoSourceNotFound if not found.
	Update(ctx context.Context, in *entities.ProtoSource) error

	// SoftDelete soft deletes a source.
	// Returns errs.ErrProtoSourceNotFound if not found.
	SoftDelete(ctx context.Context, in *entities.SoftDelete) error

	// Exists checks if a source exists.
	Exists(ctx context.Context, id string, includeDeleted ...bool) (bool, error)
}
