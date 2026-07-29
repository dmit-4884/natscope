// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Service manages saved connections; implementations must be thread-safe.
type Service interface {
	// Create returns errs.ErrConnectionNameAlreadyInUse if name is taken.
	Create(ctx context.Context, in *entities.SavedConnectionCreate) (*entities.SavedConnection, error)

	// Get returns errs.ErrSavedConnectionNotFound if not found.
	Get(ctx context.Context, id string) (*entities.SavedConnection, error)

	List(ctx context.Context, in *entities.SavedConnectionsList) (*entities.List[entities.SavedConnections], error)

	// Update returns errs.ErrSavedConnectionNotFound if not found.
	Update(ctx context.Context, in *entities.SavedConnectionUpdate) (*entities.SavedConnection, error)

	// Delete returns errs.ErrSavedConnectionNotFound if not found.
	Delete(ctx context.Context, id string) error

	// Duplicate returns errs.ErrSavedConnectionNotFound if source not found.
	Duplicate(ctx context.Context, id string, newName string) (*entities.SavedConnection, error)

	// TestConnection probes NATS. With ConnectionID set, it probes the saved
	// config (ignoring caller URLs/auth/TLS) and persists outcome to Meta;
	// empty, it probes the request's URLs/auth/TLS ad-hoc. Meta write failures
	// are non-fatal — the probe result is always returned.
	TestConnection(ctx context.Context, in *entities.TestConnectionRequest) (*entities.TestConnectionResult, error)
}
