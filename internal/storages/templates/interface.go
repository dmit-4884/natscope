// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package templates is the storage contract for message templates.
package templates

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Storage defines the contract for message template persistence.
type Storage interface {
	// Save inserts a new template.
	Save(ctx context.Context, in *entities.MessageTemplate) error

	// Get returns the template by id, or errs.ErrMessageTemplateNotFound.
	Get(ctx context.Context, id string) (*entities.MessageTemplate, error)

	// Update loads the template by id, applies mutate and persists the result in one transaction.
	// Returns errs.ErrMessageTemplateNotFound if id is not present.
	Update(ctx context.Context, id string, mutate func(existing *entities.MessageTemplate)) (*entities.MessageTemplate, error)

	// List returns templates with cursor-based pagination.
	List(ctx context.Context, in *entities.MessageTemplatesList) (*entities.List[entities.MessageTemplates], error)

	// Delete removes the template by id, or errs.ErrMessageTemplateNotFound.
	Delete(ctx context.Context, id string) error

	// DeleteAll removes every stored template and returns the count removed.
	DeleteAll(ctx context.Context) (int64, error)
}
