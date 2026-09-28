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

	// Update atomically loads the connection by id, lets mutate apply the
	// caller's change to it, and persists the result within a single storage
	// transaction — so concurrent partial updates cannot race (QA-008).
	// errs.ErrSavedConnectionNotFound if missing. mutate returns
	// authReplaced/tlsReplaced: whether it explicitly replaced (rather than
	// left untouched) the Auth/TLS subtree, so vault secrets belonging to the
	// previous config that are not part of the new one are purged instead of
	// merged forward (QA-006).
	Update(
		ctx context.Context,
		id string,
		mutate func(existing *entities.SavedConnection) (authReplaced, tlsReplaced bool),
	) (*entities.SavedConnection, error)

	// Delete permanently removes a connection; errs.ErrConnectionNotFound if
	// missing.
	Delete(ctx context.Context, id string) error
}
