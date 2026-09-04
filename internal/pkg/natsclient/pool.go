// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsclient

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	"golang.org/x/sync/singleflight"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// ConfigSource resolves a connection ID to its saved configuration. The pool
// stays storage-agnostic: the caller decides where configurations live and
// passes a resolver here.
type ConfigSource func(ctx context.Context, id string) (*entities.SavedConnection, error)

// Pool is an in-memory registry of live clients keyed by connection ID. It
// dials lazily through the Dialer on first use, deduplicates concurrent
// connects, and drops dead connections so they are re-dialed on next access.
type Pool struct {
	mu    sync.RWMutex
	group singleflight.Group

	clients map[string]Client
	dialer  Dialer
	source  ConfigSource
	logger  *slog.Logger
}

// NewPool creates a pool dialing via dialer with configurations resolved
// through source.
func NewPool(dialer Dialer, source ConfigSource) *Pool {
	return &Pool{
		clients: make(map[string]Client),
		dialer:  dialer,
		source:  source,
		logger:  slog.Default().With(slogx.Module("natsclient:pool")),
	}
}

// Client returns a live client for the ID, checking the in-memory pool first
// then loading the configuration and dialing; establishes it lazily if needed.
func (p *Pool) Client(ctx context.Context, connectionID string) (Client, error) {
	// Fast path: check the in-memory pool.
	p.mu.RLock()
	if c, ok := p.clients[connectionID]; ok {
		if c.IsConnected() {
			p.mu.RUnlock()
			return c, nil
		}
		p.mu.RUnlock()

		if c.IsReconnecting() {
			return nil, errs.ErrNATSConnectionClosed
		}

		p.logger.Warn("NATS connection is dead, reconnecting",
			slogx.String("connection_id", connectionID),
			slogx.String("status", c.Status()))
		p.Disconnect(connectionID)
	} else {
		p.mu.RUnlock()
	}

	// Slow path: singleflight dedups concurrent connects. DoChan + select-on-ctx
	// lets the caller bail on cancel while the dial finishes in the background.
	ch := p.group.DoChan(connectionID, func() (any, error) {
		// Re-check pool (another goroutine may have connected while we waited).
		p.mu.RLock()
		if c, ok := p.clients[connectionID]; ok {
			p.mu.RUnlock()
			return c, nil
		}
		p.mu.RUnlock()

		saved, err := p.source(ctx, connectionID)
		if err != nil {
			return nil, coreerrs.WrapOperation(err, "load connection config")
		}

		c, err := p.dialer.Dial(ctx, saved)
		if err != nil {
			return nil, err
		}

		p.mu.Lock()
		p.clients[connectionID] = c
		p.mu.Unlock()

		return c, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		c, ok := res.Val.(Client)
		if !ok {
			return nil, fmt.Errorf("%w: %T", errs.ErrUnexpectedSingleflightType, res.Val)
		}
		return c, nil
	}
}

// Disconnect closes and removes a live client from the pool.
func (p *Pool) Disconnect(connectionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	c, ok := p.clients[connectionID]
	if !ok {
		return
	}

	c.Close()
	delete(p.clients, connectionID)

	p.logger.Info("disconnected from NATS pool",
		slogx.String("connection_id", connectionID))
}

// Close closes all clients and clears the pool.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, c := range p.clients {
		c.Close()
	}

	p.clients = make(map[string]Client)
	p.logger.Debug("closed all NATS connections")
}
