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

// maxTestConnectionTimeout caps a single TestConnection call regardless of ctx, connect timeout or URL count.
const maxTestConnectionTimeout = 30 * time.Second

// TestConnection tests a NATS connection without saving or pooling it. The call is bounded by
// maxTestConnectionTimeout and closes every dialed socket once ctx is done.
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

// sanitizeTestError removes raw bytes echoed from the dialed endpoint from the error text.
func sanitizeTestError(err error) string {
	msg := err.Error()
	if strings.HasPrefix(msg, "malformed HTTP") {
		return "unexpected response from server: not a valid NATS or WebSocket endpoint"
	}
	return natsutil.MaskURL(msg)
}

// ctxDialer is a nats.CustomDialer that refuses dials once ctx is done and tracks sockets for closeAll.
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

// closeAll closes every dialed socket and blocks further dials.
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
