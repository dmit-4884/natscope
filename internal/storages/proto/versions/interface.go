// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package versions is the storage contract for proto source versions.
package versions

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for proto versions storage.
type Storage interface {
	// Save creates a new proto version.
	// Returns errs.ErrProtoVersionAlreadyExists if version already exists.
	Save(ctx context.Context, in *entities.ProtoVersion) error

	// Get retrieves a version by Id.
	// Returns errs.ErrProtoVersionNotFound if not found.
	Get(ctx context.Context, id string) (*entities.ProtoVersion, error)

	// GetBySourceAndTag retrieves a version by source Id and tag.
	// Returns errs.ErrProtoVersionNotFound if not found.
	GetBySourceAndTag(ctx context.Context, sourceID, tag string) (*entities.ProtoVersion, error)

	// List returns versions for a source with pagination.
	List(ctx context.Context, in *entities.ProtoVersionsList) (*entities.List[entities.ProtoVersions], error)

	// GetTagsBySource returns all tags for a source.
	GetTagsBySource(ctx context.Context, sourceID string) ([]string, error)

	// Exists checks if a version exists.
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsBySourceAndTag checks if a version exists by source and tag.
	ExistsBySourceAndTag(ctx context.Context, sourceID, tag string) (bool, error)

	// Delete deletes a version (hard delete since it's cached data).
	Delete(ctx context.Context, id string) error

	// DeleteBySource deletes all versions for a source.
	DeleteBySource(ctx context.Context, sourceID string) (int64, error)
}
