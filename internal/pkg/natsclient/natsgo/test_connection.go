// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	corectx "github.com/altessa-s/go-atlas/core/context"
)

// maxTestConnectionTimeout bounds how long a single TestConnection call may
// block regardless of the caller's own deadline, the requested connect
// timeout, or how many URLs are probed: without a cap, an unauthenticated
// caller could park goroutines/sockets for days (see connect-timeout unit bug)
// or for urls-count * per-url timeout.
const maxTestConnectionTimeout = 30 * time.Second

// TestConnection tests a NATS connection without saving or pooling it. The
// overall call is bounded by maxTestConnectionTimeout and aborts promptly on
// ctx cancellation: a custom dialer tracks every socket it opens and closes
// them all once ctx is done, so nats.go's per-URL retry loop fails fast
// instead of blocking for urls-count * per-url timeout.
func (d *Dialer) TestConnection(
	ctx context.Context,
	in *entities.TestConnectionRequest,
) (*entities.TestConnectionResult, error) {
	result := &entities.TestConnectionResult{}

	natsOpts, err := buildTestOptions(in)
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	ctx, cancel := corectx.WithMaxTimeout(ctx, maxTestConnectionTimeout)
	defer cancel()

	cd := newCtxDialer(ctx)
	stop := context.AfterFunc(ctx, cd.closeAll)
	defer stop()

	natsOpts = append(natsOpts, nats.SetCustomDialer(cd))

	url := strings.Join(in.URLs, ",")

	conn, err := nats.Connect(url, natsOpts...)
	if err != nil {
		result.Error = sanitizeTestError(err)
		return result, nil
	}
	defer conn.Close()

	result.Success = true
	result.ConnectedURL = natsutil.MaskURL(conn.ConnectedUrl())
	result.ServerVersion = conn.ConnectedServerVersion()
	result.ServerName = conn.ConnectedServerName()
	result.ServerID = conn.ConnectedServerId()
	result.ClusterName = conn.ConnectedClusterName()
	result.MaxPayload = conn.MaxPayload()
	result.DiscoveredServers = conn.DiscoveredServers()

	// Measure RTT
	start := time.Now()
	if flushErr := conn.Flush(); flushErr == nil {
		result.RTTMs = time.Since(start).Milliseconds()
	}

	// Check JetStream availability
	js, err := conn.JetStream()
	if err == nil {
		_, err = js.AccountInfo()
		result.JetstreamEnabled = err == nil
	}

	return result, nil
}

// sanitizeTestError normalizes error text that would otherwise echo raw bytes
// read from the dialed endpoint back to the caller. nats.go's websocket dialer
// parses the initial response as HTTP; against a non-HTTP/non-NATS listener the
// stdlib embeds the first line it read (a banner, a JSON blob, ...) verbatim in
// the error, turning TestConnection into a banner-grab primitive.
func sanitizeTestError(err error) string {
	msg := err.Error()
	if strings.HasPrefix(msg, "malformed HTTP") {
		return "unexpected response from server: not a valid NATS or WebSocket endpoint"
	}
	return natsutil.MaskURL(msg)
}

// ctxDialer is a nats.CustomDialer that refuses to dial once ctx is done and
// tracks every socket it opens so closeAll can cut them all at once. Without
// this, closing the caller's ctx (client disconnect, RPC deadline) leaves
// nats.go's server-list loop to run out its own per-URL timeouts one by one.
type ctxDialer struct {
	ctx    context.Context
	dialer net.Dialer

	mu     sync.Mutex
	closed bool
	conns  []net.Conn
}

func newCtxDialer(ctx context.Context) *ctxDialer {
	return &ctxDialer{ctx: ctx}
}

// Dial implements nats.CustomDialer.
func (d *ctxDialer) Dial(network, address string) (net.Conn, error) {
	if err := d.ctx.Err(); err != nil {
		return nil, err
	}

	conn, err := d.dialer.DialContext(d.ctx, network, address)
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		_ = conn.Close()
		return nil, d.ctx.Err()
	}
	d.conns = append(d.conns, conn)
	d.mu.Unlock()

	return conn, nil
}

// closeAll closes every socket dialed so far and blocks further dials; called
// once ctx is done.
func (d *ctxDialer) closeAll() {
	d.mu.Lock()
	d.closed = true
	conns := d.conns
	d.conns = nil
	d.mu.Unlock()

	for _, c := range conns {
		_ = c.Close()
	}
}
