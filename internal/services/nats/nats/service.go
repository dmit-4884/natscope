// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsclient"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	connectionsStorage "github.com/dmit-4884/natscope/internal/storages/connections"
)

// Service resolves a connection ID to a live [natsclient.Client] via the pool
// and delegates every operation to it. All SDK usage and error translation
// live in the client implementation; this layer is pure orchestration.
type Service struct {
	pool *natsclient.Pool

	// dialer serves ad-hoc dials that bypass the pool (TestConnection).
	dialer natsclient.Dialer
}

// New creates a new NATS service backed by the given dialer and connection
// storage. Connections are established lazily on first use; the storage stays
// a service-layer dependency and reaches the pool only as a config resolver.
func New(dialer natsclient.Dialer, connStore connectionsStorage.Storage) *Service {
	source := func(ctx context.Context, id string) (*entities.SavedConnection, error) {
		return connStore.Get(ctx, id)
	}
	return &Service{
		pool:   natsclient.NewPool(dialer, source),
		dialer: dialer,
	}
}

// client returns a live client for the connection ID.
func (s *Service) client(ctx context.Context, connectionID string) (natsclient.Client, error) {
	return s.pool.Client(ctx, connectionID)
}

// DisconnectFromPool closes and removes a live connection from the in-memory
// pool.
func (s *Service) DisconnectFromPool(connectionID string) {
	s.pool.Disconnect(connectionID)
}

// Close closes all connections and clears the pool.
func (s *Service) Close() {
	s.pool.Close()
}

// Compile-time checks that Service satisfies every segregated NATS role.
var (
	_ natssvc.ConnectionManager = (*Service)(nil)
	_ natssvc.StreamReader      = (*Service)(nil)
	_ natssvc.StreamManager     = (*Service)(nil)
	_ natssvc.ConsumerManager   = (*Service)(nil)
	_ natssvc.Publisher         = (*Service)(nil)
	_ natssvc.Subscriber        = (*Service)(nil)
	_ natssvc.StatsReader       = (*Service)(nil)
	_ natssvc.KVStore           = (*Service)(nil)
	_ natssvc.ObjectStore       = (*Service)(nil)
)
