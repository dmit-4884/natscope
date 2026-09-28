// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package settings is the storage contract for user settings.
package settings

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for user settings storage.
type Storage interface {
	// Save creates or updates user settings (upsert).
	Save(ctx context.Context, in *entities.UserSettings) error

	// Get retrieves the settings.
	// Returns errs.ErrSettingsNotFound if no custom settings exist.
	Get(ctx context.Context) (*entities.UserSettings, error)

	// Update atomically loads settings (or a fresh default if none saved yet),
	// lets mutate apply the caller's change, and persists the result within a
	// single storage transaction — so concurrent partial updates to different
	// setting groups cannot race.
	Update(ctx context.Context, mutate func(existing *entities.UserSettings)) (*entities.UserSettings, error)

	// Delete removes user settings.
	Delete(ctx context.Context) error
}
