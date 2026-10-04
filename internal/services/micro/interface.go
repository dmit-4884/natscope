// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package micro

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Service discovers NATS Micro services.
type Service interface {
	// ListServices lists the running services with the user's access to their info and stats.
	ListServices(ctx context.Context, connectionID string) (*entities.MicroDiscovery, error)
}
