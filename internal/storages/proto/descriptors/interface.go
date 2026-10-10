// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package descriptors is the storage contract for compiled proto schemas.
package descriptors

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for compiled proto schema storage.
type Storage interface {
	// Save stores a schema, replacing the one for the same source and revision.
	Save(ctx context.Context, in *entities.ProtoDescriptor) error

	// GetBySourceRevision returns errs.ErrProtoDescriptorNotFound when absent.
	GetBySourceRevision(ctx context.Context, sourceID, revision string) (*entities.ProtoDescriptor, error)

	// GetByFingerprint returns errs.ErrProtoDescriptorNotFound when absent.
	GetByFingerprint(ctx context.Context, sourceID, fingerprint string) (*entities.ProtoDescriptor, error)

	// ListBySource returns every stored schema of a source.
	ListBySource(ctx context.Context, sourceID string) (entities.ProtoDescriptors, error)

	// DeleteBySourceRevision deletes one stored schema; a missing one is not an error.
	DeleteBySourceRevision(ctx context.Context, sourceID, revision string) error

	// DeleteBySource deletes every stored schema of a source.
	DeleteBySource(ctx context.Context, sourceID string) (int64, error)
}
