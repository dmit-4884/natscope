// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package templates

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Service manages templates; implementations must be thread-safe.
type Service interface {
	Create(ctx context.Context, in *entities.MessageTemplateCreate) (*entities.MessageTemplate, error)

	// Get returns errs.ErrMessageTemplateNotFound if absent.
	Get(ctx context.Context, id string) (*entities.MessageTemplate, error)

	// Update applies a partial patch; errs.ErrMessageTemplateNotFound if absent.
	Update(ctx context.Context, in *entities.MessageTemplateUpdate) (*entities.MessageTemplate, error)

	List(ctx context.Context, in *entities.MessageTemplatesList) (*entities.List[entities.MessageTemplates], error)

	// Delete returns errs.ErrMessageTemplateNotFound if absent.
	Delete(ctx context.Context, id string) error

	// BulkCreate inserts each input as a new template (fresh ids/timestamps);
	// returns count created.
	BulkCreate(ctx context.Context, in []*entities.MessageTemplateCreate) (int, error)

	// DeleteAll removes every stored template; returns deleted count.
	DeleteAll(ctx context.Context) (int64, error)
}
