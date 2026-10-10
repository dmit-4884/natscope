// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package connections

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Service manages saved connections; implementations must be thread-safe.
type Service interface {
	// Create returns errs.ErrConnectionNameAlreadyInUse if name is taken.
	Create(ctx context.Context, in *entities.SavedConnectionCreate) (*entities.SavedConnection, error)

	// ValidateCreate checks in as Create would, without saving it; a taken name is not checked.
	ValidateCreate(in *entities.SavedConnectionCreate) error

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

	// GetSidebarLayout returns the connection's sidebar layout, empty when none
	// was saved; errs.ErrSavedConnectionNotFound for an unknown connection.
	GetSidebarLayout(ctx context.Context, connectionID string) (*entities.SidebarLayout, error)

	// UpdateSidebarLayout replaces the sections set in `in` and keeps the others;
	// errs.ErrSavedConnectionNotFound for an unknown connection.
	UpdateSidebarLayout(ctx context.Context, in *entities.SidebarLayoutUpdate) (*entities.SidebarLayout, error)

	// ListCliContexts reads the nats CLI contexts on this host, or the uploaded files when any are given.
	ListCliContexts(ctx context.Context, files []entities.CliContextFile) (*entities.CliContexts, error)

	// ImportCliContexts creates a connection per named context; a missing, unusable or already saved one is skipped.
	ImportCliContexts(ctx context.Context, names []string, files []entities.CliContextFile) (*entities.CliContextImport, error)
}
