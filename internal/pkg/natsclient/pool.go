// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsclient

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	"golang.org/x/sync/singleflight"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// maxStaleDials bounds the re-dials when connections keep being disconnected mid-dial.
const maxStaleDials = 3

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

	clients   map[string]Client
	gens      map[string]uint64
	dialer    Dialer
	source    ConfigSource
	logger    *slog.Logger
	listeners []func(connectionID string)
}

// NewPool creates a pool dialing via dialer with configurations resolved
// through source.
func NewPool(dialer Dialer, source ConfigSource) *Pool {
	return &Pool{
		clients: make(map[string]Client),
		gens:    make(map[string]uint64),
		dialer:  dialer,
		source:  source,
		logger:  slog.Default().With(slogx.Module("natsclient:pool")),
	}
}

// Client returns a live client for the ID, checking the in-memory pool first
// then loading the configuration and dialing; establishes it lazily if needed.
func (p *Pool) Client(ctx context.Context, connectionID string) (Client, error) {
	// Fast path: check the in-memory pool; the client is asked about its state outside the pool lock.
	p.mu.RLock()
	c, ok := p.clients[connectionID]
	p.mu.RUnlock()
	if ok {
		if c.IsConnected() {
			return c, nil
		}
		if c.IsReconnecting() {
			return nil, errs.ErrNATSConnectionClosed
		}

		p.logger.Warn("NATS connection is dead, reconnecting",
			slogx.String("connection_id", connectionID),
			slogx.String("status", c.Status()))
		p.drop(connectionID, c)
	}

	// Slow path: singleflight dedups concurrent connects. DoChan + select-on-ctx
	// lets the caller bail on cancel while the dial finishes in the background.
	ch := p.group.DoChan(connectionID, func() (any, error) {
		return p.dial(ctx, connectionID)
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

// dial loads the configuration and dials, again when a Disconnect made the configuration it read stale.
func (p *Pool) dial(ctx context.Context, connectionID string) (Client, error) {
	for attempt := 1; ; attempt++ {
		p.mu.RLock()
		if c, ok := p.clients[connectionID]; ok {
			p.mu.RUnlock()
			return c, nil
		}
		gen := p.gens[connectionID]
		p.mu.RUnlock()

		saved, err := p.source(ctx, connectionID)
		if err != nil {
			return nil, coreerrs.WrapOperation(err, "load connection config")
		}

		c, err := p.dialer.Dial(ctx, saved)
		if err != nil {
			return nil, err
		}
		if saved.ReadOnly {
			c = newReadOnlyClient(c)
		}

		p.mu.Lock()
		if p.gens[connectionID] == gen {
			p.clients[connectionID] = c
			p.mu.Unlock()
			return c, nil
		}
		p.mu.Unlock()
		c.Close()

		if attempt == maxStaleDials {
			return nil, errs.ErrNATSConnectionClosed
		}
	}
}

// Pooled returns the client registered for the ID, whatever its connection state, without dialing.
func (p *Pool) Pooled(connectionID string) (Client, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	c, ok := p.clients[connectionID]
	return c, ok
}

// OnDisconnect registers fn to run, outside the pool lock, whenever a connection leaves the pool.
func (p *Pool) OnDisconnect(fn func(connectionID string)) {
	p.mu.Lock()
	p.listeners = append(p.listeners, fn)
	p.mu.Unlock()
}

// Disconnect closes and removes a live client, then notifies OnDisconnect listeners. A dial in flight is discarded
// too, as it read the configuration this disconnect makes stale.
func (p *Pool) Disconnect(connectionID string) {
	p.mu.Lock()
	p.gens[connectionID]++
	c, ok := p.clients[connectionID]
	if !ok {
		p.mu.Unlock()
		return
	}
	p.removeLocked(connectionID, c)
}

// drop removes dead, a client found dead, unless another caller already replaced it.
func (p *Pool) drop(connectionID string, dead Client) {
	p.mu.Lock()
	if c, ok := p.clients[connectionID]; !ok || c != dead {
		p.mu.Unlock()
		return
	}
	p.removeLocked(connectionID, dead)
}

// removeLocked removes c, unlocks the pool, closes c and notifies OnDisconnect listeners.
func (p *Pool) removeLocked(connectionID string, c Client) {
	delete(p.clients, connectionID)
	listeners := slices.Clone(p.listeners)
	p.mu.Unlock()
	c.Close()

	p.logger.Info("disconnected from NATS pool",
		slogx.String("connection_id", connectionID))

	for _, fn := range listeners {
		fn(connectionID)
	}
}

// Close clears the pool and closes all clients.
func (p *Pool) Close() {
	p.mu.Lock()
	clients := p.clients
	p.clients = make(map[string]Client)
	p.mu.Unlock()

	for _, c := range clients {
		c.Close()
	}
	p.logger.Debug("closed all NATS connections")
}
