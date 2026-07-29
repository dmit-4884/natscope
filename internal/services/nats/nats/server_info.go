// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// GetServerInfo returns detailed NATS server information.
func (s *Service) GetServerInfo(ctx context.Context, connectionID string) (*entities.ServerInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetServerInfo(ctx)
}
