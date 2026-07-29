// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Service manages settings; implementations must be thread-safe.
type Service interface {
	// Get returns settings. Returns default (empty) settings if none saved.
	Get(ctx context.Context) (*entities.UserSettings, error)

	// Update partially updates settings (creates if not exists).
	Update(ctx context.Context, in *entities.UserSettingsUpdate) (*entities.UserSettings, error)

	// Reset deletes settings, returning default (empty) settings.
	Reset(ctx context.Context) (*entities.UserSettings, error)
}
