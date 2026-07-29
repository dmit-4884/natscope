// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package connections is the storage contract for saved NATS connections.
package connections

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for saved connections storage.
type Storage interface {
	// Save creates a connection; errs.ErrConnectionNameAlreadyInUse if name taken.
	Save(ctx context.Context, in *entities.SavedConnection) error

	// Get returns a connection by Id; errs.ErrConnectionNotFound if missing.
	Get(ctx context.Context, id string, includeDeleted ...bool) (*entities.SavedConnection, error)

	// List returns connections with pagination.
	List(ctx context.Context, in *entities.SavedConnectionsList) (*entities.List[entities.SavedConnections], error)

	// Update updates a connection; errs.ErrConnectionNotFound if missing.
	Update(ctx context.Context, in *entities.SavedConnection) error

	// Delete permanently removes a connection; errs.ErrConnectionNotFound if
	// missing.
	Delete(ctx context.Context, id string) error
}
