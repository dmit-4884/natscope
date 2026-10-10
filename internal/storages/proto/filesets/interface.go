// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package filesets is the storage contract for raw .proto files of a source revision.
package filesets

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for raw proto file storage.
type Storage interface {
	// Save stores a file set, replacing the one for the same source and revision.
	Save(ctx context.Context, in *entities.ProtoFileSet) error

	// GetBySourceRevision returns errs.ErrProtoFileSetNotFound when absent.
	GetBySourceRevision(ctx context.Context, sourceID, revision string) (*entities.ProtoFileSet, error)

	// DeleteBySourceRevision deletes one file set; a missing one is not an error.
	DeleteBySourceRevision(ctx context.Context, sourceID, revision string) error

	// DeleteBySource deletes every file set of a source.
	DeleteBySource(ctx context.Context, sourceID string) (int64, error)
}
