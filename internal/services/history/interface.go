// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Service manages publish history; implementations must be thread-safe.
type Service interface {
	// Record creates a history entry.
	Record(ctx context.Context, in *entities.PublishHistoryCreate) (*entities.PublishHistory, error)

	// List returns paginated history entries.
	List(ctx context.Context, in *entities.PublishHistoryList) (*entities.List[entities.PublishHistories], error)
}
