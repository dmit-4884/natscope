// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package history is the storage contract for publish history.
package history

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for publish history storage.
type Storage interface {
	// Save creates a new history entry.
	Save(ctx context.Context, in *entities.PublishHistory) error

	// List returns history entries with pagination.
	List(ctx context.Context, in *entities.PublishHistoryList) (*entities.List[entities.PublishHistories], error)

	// Prune keeps the newest limit entries once more than limit+slack exist and
	// returns how many were removed.
	Prune(ctx context.Context, limit, slack int) (int64, error)
}
