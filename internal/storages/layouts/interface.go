// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package layouts is the storage contract for sidebar layouts, one document
// per saved connection. The bbolt implementation lives in the [bbolt]
// sub-package.
package layouts

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for sidebar layout storage.
type Storage interface {
	// Get returns the connection's layout; errs.ErrSidebarLayoutNotFound when
	// nothing was saved for it.
	Get(ctx context.Context, connectionID string) (*entities.SidebarLayout, error)

	// Update loads the layout (or an empty one), applies mutate and persists the
	// result in one transaction.
	Update(ctx context.Context, connectionID string, mutate func(existing *entities.SidebarLayout)) (*entities.SidebarLayout, error)

	// Delete removes the connection's layout; errs.ErrSidebarLayoutNotFound when
	// nothing was saved for it.
	Delete(ctx context.Context, connectionID string) error
}
