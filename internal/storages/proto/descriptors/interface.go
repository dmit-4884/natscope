// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package descriptors is the storage contract for compiled proto descriptors.
package descriptors

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for proto descriptors storage.
type Storage interface {
	// Save creates or updates a proto descriptor.
	Save(ctx context.Context, in *entities.ProtoDescriptor) error

	// GetById retrieves a descriptor by Id.
	// Returns errs.ErrProtoDescriptorNotFound if not found.
	GetById(ctx context.Context, id string) (*entities.ProtoDescriptor, error)

	// GetBySourceTag retrieves a descriptor by source+tag combination.
	// Returns errs.ErrProtoDescriptorNotFound if not found.
	GetBySourceTag(ctx context.Context, sourceID, tag string) (*entities.ProtoDescriptor, error)

	// FindByFingerprint returns the first descriptor whose DescriptorSet hashes
	// to the given fingerprint (sha256 hex). Used for pinned-snapshot resolution.
	// Returns errs.ErrProtoDescriptorNotFound if not found.
	FindByFingerprint(ctx context.Context, fingerprint string) (*entities.ProtoDescriptor, error)

	// GetAll retrieves all descriptors.
	GetAll(ctx context.Context) (entities.ProtoDescriptors, error)

	// List returns descriptors with pagination.
	List(ctx context.Context, in *entities.ProtoDescriptorsList) (*entities.List[entities.ProtoDescriptors], error)

	// Update updates a descriptor.
	// Returns errs.ErrProtoDescriptorNotFound if not found.
	Update(ctx context.Context, in *entities.ProtoDescriptor) error

	// Delete deletes a descriptor by Id.
	Delete(ctx context.Context, id string) error

	// DeleteBySource deletes all descriptors for a source.
	DeleteBySource(ctx context.Context, sourceID string) (int64, error)

	// Exists checks if a descriptor exists by Id.
	Exists(ctx context.Context, id string) (bool, error)
}
