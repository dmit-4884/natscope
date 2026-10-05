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

// TestConnection tests a NATS connection without saving or pooling it. Each server URL is diagnosed in order and
// connected to on its own; the first one that connects wins, otherwise the one that got furthest explains the failure.
// The call is bounded by maxTestConnectionTimeout and closes every dialed socket once ctx is done.
func (d *Dialer) TestConnection(
	ctx context.Context,
	in *entities.TestConnectionRequest,
) (*entities.TestConnectionResult, error) {
	result := &entities.TestConnectionResult{}
	connectTimeout := testConnectTimeout(in.ConnectTimeout)

	natsOpts, badStep, err := buildTestOptions(in, connectTimeout)
	if err != nil {
		result.Error = err.Error()
		diag := newDiagnosis(connectTimeout)
		diag.add(badStep, entities.CheckStatusFailed, err.Error(), "Fix this setting of the connection.", time.Now())
		result.Checks = diag.result()
		return result, nil
	}

	ctx, cancel := corectx.WithMaxTimeout(ctx, maxTestConnectionTimeout)
	defer cancel()

	cd := newCtxDialer(ctx, connectTimeout)
	stop := context.AfterFunc(ctx, cd.closeAll)
	defer stop()
	natsOpts = append(natsOpts, nats.SetCustomDialer(cd), nats.IgnoreDiscoveredServers())

	var best *diagnosis
	bestURL := ""
	for _, url := range in.URLs {
		if ctx.Err() != nil {
			break
		}
		diag := diagnoseNetwork(ctx, url, in.TLS, connectTimeout)
		if diag.reached {
			if conn := diag.connect(ctx, url, natsOpts, in.Auth); conn != nil {
				finishConnected(ctx, result, conn, diag, in)
				return result, nil
			}
		}
		if best == nil || diag.progress() > best.progress() {
			best, bestURL = diag, url
		}
	}
	if best == nil {
		best = newDiagnosis(connectTimeout)
		best.outOfTime = true
	}

	result.Error = best.failure()
	if len(in.URLs) > 1 && bestURL != "" {
		result.Error = natsutil.MaskURL(bestURL) + ": " + result.Error
	}
	result.Checks = best.result()
	return result, nil
}

// testConnectTimeout is the connect timeout of a test, the default when unset or not positive.
func testConnectTimeout(timeout *time.Duration) time.Duration {
	if timeout == nil || *timeout <= 0 {
		return defaultConnectTimeout
	}
	return *timeout
}

// finishConnected fills the result from a connection that succeeded, checks JetStream and closes the connection.
func finishConnected(ctx context.Context, result *entities.TestConnectionResult, conn *nats.Conn, diag *diagnosis, in *entities.TestConnectionRequest) {
	defer conn.Close()
	result.Success = true
	result.ConnectedURL = natsutil.MaskURL(conn.ConnectedUrl())
	result.ServerVersion = conn.ConnectedServerVersion()
	result.ServerName = conn.ConnectedServerName()
	result.ServerID = conn.ConnectedServerId()
	result.ClusterName = conn.ConnectedClusterName()
	result.MaxPayload = conn.MaxPayload()
	result.DiscoveredServers = diag.discovered(in.URLs)

	rttStart := time.Now()
	if flushErr := conn.Flush(); flushErr == nil {
		result.RTTMs = time.Since(rttStart).Milliseconds()
	}

	result.JetstreamEnabled = jetStreamCheck(ctx, diag, conn, in.Connection)
	result.Checks = diag.result()
}

// sanitizeTestError removes raw bytes echoed from the dialed endpoint from the error text.
func sanitizeTestError(err error) string {
	msg := err.Error()
	if strings.HasPrefix(msg, "malformed HTTP") {
		return "unexpected response from server: not a valid NATS or WebSocket endpoint"
	}
	return serverText(natsutil.MaskURL(msg))
}

// ctxDialer is a nats.CustomDialer that refuses dials once ctx is done and tracks sockets for closeAll.
type ctxDialer struct {
	ctx    context.Context
	dialer net.Dialer

	mu     sync.Mutex
	closed bool
	conns  []net.Conn
}

func newCtxDialer(ctx context.Context, timeout time.Duration) *ctxDialer {
	return &ctxDialer{ctx: ctx, dialer: net.Dialer{Timeout: timeout}}
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
