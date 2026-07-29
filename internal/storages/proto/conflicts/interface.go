// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package conflicts is the storage contract for proto schema conflicts.
// Rewritten in full on every reload — the runtime merge is the source of truth,
// so the storage layer does no append/dedup.
package conflicts

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for proto schema conflict persistence.
type Storage interface {
	// ReplaceAll overwrites the conflict list, after a compile/reload produces a
	// fresh report.
	ReplaceAll(ctx context.Context, conflicts entities.SchemaConflicts) error

	// GetAll returns all conflicts.
	GetAll(ctx context.Context) (entities.SchemaConflicts, error)
}
